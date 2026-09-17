/*
 Copyright 2025 The Nephio Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 You may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v66/github"
	git "github.com/nephio-project/nephio/controllers/pkg/git"
	gittypes "github.com/nephio-project/nephio/controllers/pkg/git/types"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	// context.Background rather than ctrl.SetupSignalHandler: that installs a
	// process-wide handler and panics on a second call, so the package could
	// only ever be run once.
	ctx := context.Background()

	type args struct {
		ctx    context.Context
		client resource.APIPatchingApplicator
	}
	tests := []struct {
		name    string
		args    args
		want    git.Client
		wantErr bool
	}{

		{
			name:    "ctx nil check",
			args:    args{nil, resource.NewAPIPatchingApplicator(nil)},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "client nil check",
			args:    args{ctx, resource.NewAPIPatchingApplicator(nil)},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetClient(tt.args.ctx, tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCreateAccessToken covers the credential the user and repository calls run
// as. Minting an installation token used to replace the shared client, so
// everything after the first Token reconcile authenticated as the installation,
// which the authenticated-user endpoint rejects.
// request is one call the test server saw, kept apart so an assertion can name
// what it is checking rather than matching a joined string.
type request struct {
	method        string
	path          string
	authorization string
}

// githubAPI answers the calls this test makes and records them. Assertions run
// on the test goroutine: require calls FailNow, which is only defined there.
func githubAPI(t *testing.T, mint func(w http.ResponseWriter)) (*httptest.Server, func() []request) {
	t.Helper()
	var mu sync.Mutex
	var seen []request

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, request{r.Method, r.URL.Path, r.Header.Get("Authorization")})
		mu.Unlock()

		switch {
		case strings.Contains(r.URL.Path, "access_tokens"):
			mint(w)
		case strings.HasPrefix(r.URL.Path, "/repos/"):
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name": "a-repo", "clone_url": "https://example.invalid/a-repo.git"})
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"login": "someone", "id": 1})
		}
	}))
	t.Cleanup(server.Close)

	return server, func() []request {
		mu.Lock()
		defer mu.Unlock()
		return append([]request(nil), seen...)
	}
}

func mintsAToken(w http.ResponseWriter) {
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"token": "installation-token"})
}

// githubClient builds the client the singleton would hand out, pointed at the
// test server. server.Client() on both: the server is TLS, so a client that
// does not carry its certificate cannot reach it, which is what gives the
// injection something to fail on.
func githubClient(t *testing.T, server *httptest.Server) (*gc, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	patClient := github.NewClient(server.Client()).WithAuthToken("personal-access-token")
	patClient.BaseURL, err = url.Parse(server.URL + "/")
	require.NoError(t, err)

	return &gc{
		githubClient:  patClient,
		appID:         "1",
		installID:     "2",
		privateKey:    key,
		tokenEndpoint: server.URL,
		tokenClient:   server.Client(),
		l:             logr.Discard(),
	}, key
}

// requireAppJWT checks the mint ran as the app. Asserting only that the PAT is
// absent would also accept no credential at all, or any other string.
func requireAppJWT(t *testing.T, authorization, appID string, key *rsa.PrivateKey) {
	t.Helper()
	raw, found := strings.CutPrefix(authorization, "Bearer ")
	require.True(t, found, "the mint carried no bearer credential")

	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("signed with %v, not RSA", token.Header["alg"])
		}
		return &key.PublicKey, nil
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(appID))

	require.NoError(t, err, "the mint did not carry a JWT this app signed")
	require.True(t, token.Valid)
	require.NotNil(t, claims.ExpiresAt, "an app JWT has to expire")
	require.True(t, claims.ExpiresAt.After(time.Now()))
}

func TestCreateAccessToken(t *testing.T) {
	t.Parallel()

	t.Run("the default token client is bounded", func(t *testing.T) {
		// Nothing configures tokenClient in production, so the fallback is
		// what mints run through.
		require.Positive(t, defaultTokenClient.Timeout,
			"a mint that is never answered must not hold the reconcile open")
	})

	t.Run("the mint runs as the app and everything else as the PAT", func(t *testing.T) {
		server, requests := githubAPI(t, mintsAToken)
		client, key := githubClient(t, server)

		_, _, err := client.GetMyUserInfo()
		require.NoError(t, err)

		token, _, err := client.CreateAccessToken(gittypes.CreateAccessTokenOption{Name: "a-token"})
		require.NoError(t, err)
		require.Equal(t, "installation-token", token.Token)

		_, _, err = client.GetMyUserInfo()
		require.NoError(t, err, "the second user call must still run as the PAT")

		// The repository calls this change is about, not only the user one.
		_, _, err = client.GetRepo("someone", "a-repo")
		require.NoError(t, err)

		seen := requests()
		require.Len(t, seen, 4)
		require.Equal(t, request{"GET", "/user", "Bearer personal-access-token"}, seen[0])
		require.Equal(t, "POST", seen[1].method)
		require.Equal(t, "/app/installations/2/access_tokens", seen[1].path)
		requireAppJWT(t, seen[1].authorization, client.appID, key)
		require.Equal(t, request{"GET", "/user", "Bearer personal-access-token"}, seen[2])
		require.Equal(t, request{"GET", "/repos/someone/a-repo", "Bearer personal-access-token"}, seen[3])

		// The base URL survives too: replacing the client discarded it.
		require.Equal(t, server.URL+"/", client.githubClient.BaseURL.String())
	})

	t.Run("a repository call after a failed mint still runs as the PAT", func(t *testing.T) {
		server, requests := githubAPI(t, func(w http.ResponseWriter) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
		})
		client, _ := githubClient(t, server)

		_, _, err := client.CreateAccessToken(gittypes.CreateAccessTokenOption{Name: "a-token"})
		require.Error(t, err)

		_, _, err = client.GetRepo("someone", "a-repo")
		require.NoError(t, err)

		seen := requests()
		require.Equal(t, request{"GET", "/repos/someone/a-repo", "Bearer personal-access-token"},
			seen[len(seen)-1])
	})

	t.Run("an answer with no token is not a credential", func(t *testing.T) {
		for name, mint := range map[string]func(w http.ResponseWriter){
			"an empty token": func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"token":""}`))
			},
			"no token field": func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{}`))
			},
			"a null token": func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"token":null}`))
			},
		} {
			t.Run(name, func(t *testing.T) {
				server, _ := githubAPI(t, mint)
				client, _ := githubClient(t, server)

				// It would otherwise be written into a Secret as the password.
				_, _, err := client.CreateAccessToken(gittypes.CreateAccessTokenOption{Name: "a-token"})
				require.Error(t, err)
			})
		}
	})
}
