// Copyright 2023 The Nephio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package token

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	git "github.com/nephio-project/nephio/controllers/pkg/git"
	"github.com/nephio-project/nephio/controllers/pkg/git/types"

	mocks "github.com/nephio-project/nephio/controllers/pkg/mocks/external/client"
	gitclientmocks "github.com/nephio-project/nephio/controllers/pkg/mocks/external/gitclient"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"github.com/nephio-project/nephio/testing/mockeryutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fields struct {
	APIPatchingApplicator resource.APIPatchingApplicator
	gitClients            map[git.ProviderType]git.Client
	finalizer             *resource.APIFinalizer
}
type args struct {
	ctx       context.Context
	gitClient git.Client
	cr        *infrav1alpha1.Token
}
type tokenTests struct {
	name    string
	fields  fields
	args    args
	mocks   []mockeryutils.MockHelper
	wantErr bool
}

func TestDeleteToken(t *testing.T) {
	named := func(name string) *infrav1alpha1.Token {
		return &infrav1alpha1.Token{ObjectMeta: metav1.ObjectMeta{Name: name}}
	}
	status := func(code int) *types.Response {
		return &types.Response{Response: &http.Response{StatusCode: code}}
	}

	tests := []tokenTests{
		{
			// The name, not an id: Gitea resolves either from the same path
			// segment, and the name is what the reconciler has.
			name:   "the token is deleted by name",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil},
			args:   args{nil, nil, named("a-token")},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "DeleteAccessToken", ArgType: []string{"string"},
					RetArgList: []interface{}{status(http.StatusNoContent), nil}},
			},
			wantErr: false,
		},
		{
			// Deleting the same Token twice, or one removed on the server,
			// has already reached the state deletion is asking for.
			name:   "a 404 is the state deletion wanted",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil},
			args:   args{nil, nil, named("a-token")},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "DeleteAccessToken", ArgType: []string{"string"},
					RetArgList: []interface{}{status(http.StatusNotFound), fmt.Errorf("The target couldn't be found.")}},
			},
			wantErr: false,
		},
		{
			name:   "any other failure is reported",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil},
			args:   args{nil, nil, named("a-token")},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "DeleteAccessToken", ArgType: []string{"string"},
					RetArgList: []interface{}{status(http.StatusForbidden), fmt.Errorf("forbidden")}},
			},
			wantErr: true,
		},
		{
			// The SDK returns no response when it refuses the argument, and
			// none on a transport error either.
			name:   "a failure with no response is reported rather than dereferenced",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil},
			args:   args{nil, nil, named("a-token")},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "DeleteAccessToken", ArgType: []string{"string"},
					RetArgList: []interface{}{nil, fmt.Errorf("dial tcp: connection refused")}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &reconciler{
				APIPatchingApplicator: tt.fields.APIPatchingApplicator,
				gitClients:            tt.fields.gitClients,
				finalizer:             tt.fields.finalizer,
			}

			initMockeryMocks(&tt)

			require.NotPanics(t, func() {
				if err := r.deleteToken(tt.args.ctx, tt.args.gitClient, tt.args.cr); (err != nil) != tt.wantErr {
					t.Errorf("deleteToken() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
			tt.args.gitClient.(*gitclientmocks.MockClient).AssertCalled(t, "DeleteAccessToken", "a-token")
		})
	}
}

func TestCreateToken(t *testing.T) {

	clientMock := new(mocks.MockClient)
	clientMock.On("Get", nil, mock.AnythingOfType("types.NamespacedName"), mock.AnythingOfType("*v1.Secret")).Return(nil).Run(func(args mock.Arguments) {})
	clientMock.On("Patch", nil, mock.AnythingOfType("*v1.Secret"), mock.AnythingOfType("*resource.patch")).Return(nil).Run(func(args mock.Arguments) {})

	tests := []tokenTests{
		{
			name:   "Create Access token reports user auth error",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil},
			args:   args{nil, nil, &infrav1alpha1.Token{}},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "ListAccessTokens",
					ArgType:    []string{"types.ListAccessTokensOptions"},
					RetArgList: []interface{}{nil, nil, fmt.Errorf("\"username\" not set: only BasicAuth allowed")}},
			},
			wantErr: true,
		},
		{
			name:   "Create Access token already exists",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil},
			args: args{nil, nil, &infrav1alpha1.Token{
				TypeMeta: metav1.TypeMeta{
					APIVersion: corev1.SchemeGroupVersion.Identifier(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "test-token",
				}}},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "ListAccessTokens",
					ArgType: []string{"types.ListAccessTokensOptions"},
					RetArgList: []interface{}{[]*types.AccessToken{
						{ID: 123,
							Name: "test-token-test-ns"},
					}, nil, nil}},
			},
			wantErr: false,
		},
		{
			name:   "Create Access token reports user info not found",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil},
			args:   args{nil, nil, &infrav1alpha1.Token{}},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "ListAccessTokens",
					ArgType: []string{"types.ListAccessTokensOptions"},
					RetArgList: []interface{}{[]*types.AccessToken{
						{ID: 123,
							Name: "test-token-test-ns"},
					}, nil, nil}},
				{MethodName: "GetMyUserInfo", ArgType: []string{}, RetArgList: []interface{}{nil, nil, fmt.Errorf("error getting User Information")}},
			},
			wantErr: true,
		},
		{
			name:   "Create Access token reports failed to create",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil},
			args:   args{nil, nil, &infrav1alpha1.Token{}},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "ListAccessTokens",
					ArgType: []string{"types.ListAccessTokensOptions"},
					RetArgList: []interface{}{[]*types.AccessToken{
						{ID: 123,
							Name: "test-token-test-ns"},
					}, nil, nil}},
				{MethodName: "GetMyUserInfo", ArgType: []string{}, RetArgList: []interface{}{&types.User{UserName: "gitea"}, nil, nil}},
				{MethodName: "CreateAccessToken",
					ArgType:    []string{"types.CreateAccessTokenOption"},
					RetArgList: []interface{}{&types.AccessToken{}, nil, fmt.Errorf("failed to create token")}},
			},
			wantErr: true,
		},
		{
			name:   "Create Access token reports success",
			fields: fields{resource.NewAPIPatchingApplicator(clientMock), nil, nil},
			args: args{nil, nil, &infrav1alpha1.Token{
				TypeMeta: metav1.TypeMeta{
					APIVersion: corev1.SchemeGroupVersion.Identifier(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "test-token",
				}}},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "ListAccessTokens", ArgType: []string{"types.ListAccessTokensOptions"}, RetArgList: []interface{}{[]*types.AccessToken{}, nil, nil}},
				{MethodName: "GetMyUserInfo", ArgType: []string{}, RetArgList: []interface{}{&types.User{UserName: "gitea"}, nil, nil}},
				{MethodName: "CreateAccessToken",
					ArgType: []string{"types.CreateAccessTokenOption"},
					RetArgList: []interface{}{&types.AccessToken{ID: 123,
						Name: "test-token-test-ns"}, nil, nil}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &reconciler{
				APIPatchingApplicator: tt.fields.APIPatchingApplicator,
				gitClients:            tt.fields.gitClients,
				finalizer:             tt.fields.finalizer,
			}

			initMockeryMocks(&tt)

			if err := r.createToken(tt.args.ctx, tt.args.gitClient, tt.args.cr); (err != nil) != tt.wantErr {
				t.Errorf("createToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func initMockeryMocks(tt *tokenTests) {
	mockGitClient := new(gitclientmocks.MockClient)
	tt.args.gitClient = mockGitClient
	tt.fields.gitClients = make(map[git.ProviderType]git.Client)
	tt.fields.gitClients[git.ProviderGitea] = mockGitClient
	mockeryutils.InitMocks(&mockGitClient.Mock, tt.mocks)
}
