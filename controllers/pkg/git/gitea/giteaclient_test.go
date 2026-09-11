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

package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/go-logr/logr"
	git "github.com/nephio-project/nephio/controllers/pkg/git"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	// context.Background rather than ctrl.SetupSignalHandler: that installs a
	// process-wide handler and panics on a second call, so the package could
	// only be run once.
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
				t.Errorf("GetClient() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// giteaServer answers what the SDK sends, so the adapter is driven over HTTP
// rather than through a mock that agrees with whatever the test asserts. Tokens
// are held as IDs carrying names because the two resolve from the same path
// segment, and a fake that cannot tell them apart cannot show the difference.
type giteaServer struct {
	mu       sync.Mutex
	byID     map[int64]string
	seen     []string
	failList bool
}

func newGiteaServer(t *testing.T, byID map[int64]string) (*httptest.Server, *giteaServer) {
	t.Helper()
	state := &giteaServer{byID: byID}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The SDK reads the version before it will send a token name.
		if strings.HasSuffix(r.URL.Path, "/version") {
			_, _ = w.Write([]byte(`{"version":"1.19.3"}`))
			return
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		state.seen = append(state.seen, r.Method+" "+r.URL.RequestURI())
		if r.Method == http.MethodDelete {
			state.serveDelete(w, path.Base(r.URL.Path))
			return
		}
		state.serveList(w, r)
	}))
	t.Cleanup(server.Close)
	return server, state
}

// serveDelete resolves the path segment the way Gitea's handler does: as an ID
// through ParseInt(s, 0, 64), and as a name only when that yields zero.
func (s *giteaServer) serveDelete(w http.ResponseWriter, segment string) {
	id, err := strconv.ParseInt(segment, 0, 64)
	if err != nil || id == 0 {
		id = 0
		for candidate, name := range s.byID {
			if name == segment {
				id = candidate
				break
			}
		}
	}
	if _, ok := s.byID[id]; !ok {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"The target couldn't be found."}`))
		return
	}
	delete(s.byID, id)
	w.WriteHeader(http.StatusNoContent)
}

// serveList pages the way Gitea pages: page 0 returns every token, any other
// page returns limit rows, and limit defaults to 30.
func (s *giteaServer) serveList(w http.ResponseWriter, r *http.Request) {
	if s.failList {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"internal server error"}`))
		return
	}

	ids := make([]int64, 0, len(s.byID))
	for id := range s.byID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	if page, _ := strconv.Atoi(r.URL.Query().Get("page")); page != 0 {
		size, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if size <= 0 {
			size = 30
		}
		start := min((page-1)*size, len(ids))
		ids = ids[start:min(start+size, len(ids))]
	}

	tokens := make([]*gitea.AccessToken, 0, len(ids))
	for _, id := range ids {
		tokens = append(tokens, &gitea.AccessToken{ID: id, Name: s.byID[id]})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tokens)
}

// names reports what the server still holds, so a test can assert on what a
// delete took rather than only on what it reported.
func (s *giteaServer) names() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var left []string
	for _, name := range s.byID {
		left = append(left, name)
	}
	sort.Strings(left)
	return left
}

func (s *giteaServer) requests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.seen...)
}

func giteaAdapter(t *testing.T, server *httptest.Server) *gc {
	t.Helper()
	c, err := gitea.NewClient(server.URL,
		gitea.SetHTTPClient(server.Client()), gitea.SetBasicAuth("someone", "pw"))
	require.NoError(t, err)
	return &gc{giteaClient: c, l: logr.Discard()}
}

func TestDeleteAccessTokenByName(t *testing.T) {
	t.Parallel()
	server, state := newGiteaServer(t, map[int64]string{1: "a-token"})
	r := giteaAdapter(t, server)

	resp, err := r.DeleteAccessToken("a-token")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Empty(t, state.names())
	require.Equal(t, []string{"DELETE /api/v1/users/someone/tokens/a-token"}, state.requests(),
		"one request, and no listing to find an ID first")
}

func TestDeleteAccessTokenByID(t *testing.T) {
	t.Parallel()
	server, state := newGiteaServer(t, map[int64]string{42: "a-token"})
	r := giteaAdapter(t, server)

	resp, err := r.DeleteAccessToken(int64(42))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Empty(t, state.names())
	require.Equal(t, []string{"DELETE /api/v1/users/someone/tokens/42"}, state.requests())
}

func TestDeleteAccessTokenAbsentReports404(t *testing.T) {
	t.Parallel()
	server, _ := newGiteaServer(t, map[int64]string{})
	r := giteaAdapter(t, server)

	resp, err := r.DeleteAccessToken("a-token")
	require.Error(t, err)
	require.NotNil(t, resp, "the caller needs the status to tell absent from broken")
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteAccessTokenRejectsOtherTypesWithoutPanicking(t *testing.T) {
	t.Parallel()
	server, state := newGiteaServer(t, map[int64]string{})
	r := giteaAdapter(t, server)

	resp, err := r.DeleteAccessToken(float64(3))
	require.Error(t, err)
	require.Nil(t, resp, "nothing was sent, so there is no response to report")
	require.Empty(t, state.requests())
}

func TestDeleteAccessTokenSurvivesATransportError(t *testing.T) {
	t.Parallel()
	server, _ := newGiteaServer(t, map[int64]string{1: "a-token"})
	r := giteaAdapter(t, server)
	server.Close()

	require.NotPanics(t, func() {
		resp, err := r.DeleteAccessToken("a-token")
		require.Error(t, err)
		require.Nil(t, resp)
	})
}

// TestDeleteAccessTokenTakesTheNamedToken covers the names Gitea reads as IDs.
// Sent unresolved, each of these deletes whichever token holds that ID and
// answers 204, so the reconciler drops its finalizer having destroyed another
// Token's credential and left its own in place.
func TestDeleteAccessTokenTakesTheNamedToken(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		tokens  map[int64]string
		deletes string
		left    []string
	}{
		{
			name:    "decimal",
			tokens:  map[int64]string{34: "some-other-cluster", 41: "34"},
			deletes: "34",
			left:    []string{"some-other-cluster"},
		},
		{
			// Base 0 reads a leading zero as octal, which a decimal test for
			// "is this numeric" misses.
			name:    "leading zero reads as octal",
			tokens:  map[int64]string{7: "some-other-cluster", 42: "007"},
			deletes: "007",
			left:    []string{"some-other-cluster"},
		},
		{
			// 0x22 is hex 34, and is a valid Kubernetes object name.
			name:    "0x prefix reads as hex",
			tokens:  map[int64]string{34: "some-other-cluster", 43: "0x22"},
			deletes: "0x22",
			left:    []string{"some-other-cluster"},
		},
		{
			// ParseInt yields zero here, so Gitea looks the segment up as a
			// name by itself and the request goes straight out.
			name:    "zero is a name",
			tokens:  map[int64]string{44: "0"},
			deletes: "0",
			left:    nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server, state := newGiteaServer(t, tt.tokens)
			r := giteaAdapter(t, server)

			resp, err := r.DeleteAccessToken(tt.deletes)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, http.StatusNoContent, resp.StatusCode)
			require.Equal(t, tt.left, state.names(), "the token carrying the name goes, and only that one")
		})
	}
}

// TestDeleteAccessTokenLeavesTheIDAloneWhenTheNameIsGone is the same collision
// with the named token already deleted: the ID it reads as still exists, so an
// unresolved name would take that token and answer 204.
func TestDeleteAccessTokenLeavesTheIDAloneWhenTheNameIsGone(t *testing.T) {
	t.Parallel()
	server, state := newGiteaServer(t, map[int64]string{34: "some-other-cluster"})
	r := giteaAdapter(t, server)

	resp, err := r.DeleteAccessToken("34")
	require.NoError(t, err, "no token carries that name, which is what deletion asks for")
	require.Nil(t, resp, "nothing was deleted, so there is no delete response")
	require.Equal(t, []string{"some-other-cluster"}, state.names())
	require.Equal(t, []string{"GET /api/v1/users/someone/tokens?limit=0&page=0"}, state.requests(),
		"the list settles it and no DELETE follows")
}

// TestDeleteAccessTokenResolvesPastTheFirstPage pins the pagination the lookup
// rests on. A page holds 30 rows and the SDK's zero value asks for page 1, so a
// token past that row would read as absent and the finalizer would come off a
// credential that is still live.
func TestDeleteAccessTokenResolvesPastTheFirstPage(t *testing.T) {
	t.Parallel()
	tokens := map[int64]string{}
	for id := int64(1); id <= 34; id++ {
		tokens[id] = fmt.Sprintf("cluster-%02d", id)
	}
	tokens[99] = "34" // sorts last, so a 30-row page never reaches it

	server, state := newGiteaServer(t, tokens)
	r := giteaAdapter(t, server)

	resp, err := r.DeleteAccessToken("34")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.NotContains(t, state.names(), "34")
	require.Contains(t, state.names(), "cluster-34", "ID 34 is a different token")
	require.Equal(t, []string{
		"GET /api/v1/users/someone/tokens?limit=0&page=0",
		"DELETE /api/v1/users/someone/tokens/99",
	}, state.requests(), "page 0 turns pagination off, and the delete goes by the resolved ID")
}

// TestDeleteAccessTokenReportsAFailedLookup keeps a lookup that did not happen
// from reading as a token that is not there.
func TestDeleteAccessTokenReportsAFailedLookup(t *testing.T) {
	t.Parallel()
	server, state := newGiteaServer(t, map[int64]string{34: "some-other-cluster"})
	state.failList = true
	r := giteaAdapter(t, server)

	resp, err := r.DeleteAccessToken("34")
	require.Error(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	require.Equal(t, []string{"some-other-cluster"}, state.names())
	require.Len(t, state.requests(), 1, "no DELETE follows a lookup that failed")
}

func TestDeleteAccessTokenLookupSurvivesATransportError(t *testing.T) {
	t.Parallel()
	server, _ := newGiteaServer(t, map[int64]string{41: "34"})
	r := giteaAdapter(t, server)
	server.Close()

	require.NotPanics(t, func() {
		resp, err := r.DeleteAccessToken("34")
		require.Error(t, err)
		require.Nil(t, resp)
	})
}

func TestGiteaReadsNameAsID(t *testing.T) {
	t.Parallel()
	for name, readsAsID := range map[string]bool{
		"34":                   true,
		"007":                  true,  // octal 7
		"0x22":                 true,  // hex 34
		"0b11":                 true,  // binary 3
		"0":                    false, // yields an ID of zero, so Gitea looks it up as a name
		"0x0":                  false,
		"edge-cluster":         false,
		"cluster34":            false,
		"34-topology":          false,
		"":                     false,
		"08":                   false, // 8 is not an octal digit, so no ID comes out
		"99999999999999999999": false, // out of int64 range, and Gitea's parse fails the same way
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, readsAsID, giteaReadsNameAsID(name))
		})
	}
}
