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
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/go-logr/logr"
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

// TestCreateAccessTokenKeepsThePATClient covers the credential the user and
// repository calls run as. Minting an installation token used to replace the
// shared client, so everything after the first Token reconcile authenticated
// as the installation, which the authenticated-user endpoint rejects.
func TestCreateAccessTokenKeepsThePATClient(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var seen []string

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.URL.Path+" "+r.Header.Get("Authorization"))
		mu.Unlock()

		if strings.Contains(r.URL.Path, "access_tokens") {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "installation-token"})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"login": "someone", "id": 1})
	}))
	defer server.Close()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// server.Client() on both: the server is TLS, so a client that does not
	// carry its certificate cannot reach it, which is what gives the injection
	// below something to fail on.
	patClient := github.NewClient(server.Client()).WithAuthToken("personal-access-token")
	patClient.BaseURL, err = url.Parse(server.URL + "/")
	require.NoError(t, err)

	client := &gc{
		githubClient:  patClient,
		appID:         "1",
		installID:     "2",
		privateKey:    key,
		tokenEndpoint: server.URL,
		tokenClient:   server.Client(),
		l:             logr.Discard(),
	}

	_, _, err = client.GetMyUserInfo()
	require.NoError(t, err)

	token, _, err := client.CreateAccessToken(gittypes.CreateAccessTokenOption{Name: "a-token"})
	require.NoError(t, err)
	require.Equal(t, "installation-token", token.Token)

	_, _, err = client.GetMyUserInfo()
	require.NoError(t, err, "the second user call must still run as the PAT")

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, seen, 3)
	require.Equal(t, "/user Bearer personal-access-token", seen[0])
	require.Contains(t, seen[1], "/app/installations/2/access_tokens")
	require.NotContains(t, seen[1], "personal-access-token", "the mint runs as the app, not the PAT")
	require.Equal(t, "/user Bearer personal-access-token", seen[2])

	// The base URL survives too: replacing the client discarded it, so a
	// GitHub Enterprise endpoint silently became api.github.com.
	require.Equal(t, server.URL+"/", client.githubClient.BaseURL.String())
}
