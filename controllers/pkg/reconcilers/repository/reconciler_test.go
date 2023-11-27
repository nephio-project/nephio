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

package repository

import (
	"context"
	"errors"
	"testing"

	"code.gitea.io/sdk/gitea"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	"github.com/stretchr/testify/require"
)

type GiteaClientMock struct {
}

type NephioGiteaClientMock struct {
	myUserInfoError error
	deleteRepoError error
	getRepoError    error
	createRepoError error
	editRepoError   error
}

func (gc NephioGiteaClientMock) Get() *gitea.Client {
	return nil
}

func (gc NephioGiteaClientMock) Start(ctx context.Context) {
}

func (gc NephioGiteaClientMock) GetMyUserInfo() (*gitea.User, *gitea.Response, error) {
	return &gitea.User{}, &gitea.Response{}, gc.myUserInfoError
}

func (gc NephioGiteaClientMock) DeleteRepo(owner string, repo string) (*gitea.Response, error) {
	return nil, gc.deleteRepoError
}

func (gc NephioGiteaClientMock) GetRepo(userName string, repoCRName string) (*gitea.Repository, *gitea.Response, error) {
	return nil, nil, gc.getRepoError
}

func (gc NephioGiteaClientMock) CreateRepo(createRepoOption gitea.CreateRepoOption) (*gitea.Repository, *gitea.Response, error) {
	return &gitea.Repository{}, nil, gc.createRepoError
}

func (gc NephioGiteaClientMock) EditRepo(userName string, repoCRName string, editRepoOption gitea.EditRepoOption) (*gitea.Repository, *gitea.Response, error) {
	return &gitea.Repository{}, nil, gc.editRepoError
}

func (gc NephioGiteaClientMock) DeleteAccessToken(value interface{}) (*gitea.Response, error) {
	return nil, nil
}

func TestUpsertRepo(t *testing.T) {
	dummyString := "Dummy String"
	dummyBool := true
	dummyTrustModel := infrav1alpha1.TrustModel("Trust Model")

	testCases := map[string]struct {
		ctx         context.Context
		giteaClient NephioGiteaClientMock
		cr          infrav1alpha1.Repository
		expectedErr error
	}{
		"User Info reports error": {
			ctx: nil,
			giteaClient: NephioGiteaClientMock{
				myUserInfoError: errors.New("Error getting User Information"),
			},
			cr:          infrav1alpha1.Repository{},
			expectedErr: errors.New("Error getting User Information"),
		},
		"Repo exists, cr fields blank": {
			ctx:         nil,
			giteaClient: NephioGiteaClientMock{},
			cr: infrav1alpha1.Repository{
				Status: infrav1alpha1.RepositoryStatus{},
			},
			expectedErr: nil,
		},
		"Repo exists, cr fields not blank": {
			ctx: nil,
			giteaClient: NephioGiteaClientMock{
				editRepoError: errors.New("Repo update failed"),
			},
			cr: infrav1alpha1.Repository{
				Spec: infrav1alpha1.RepositorySpec{
					Description: &dummyString,
					Private:     &dummyBool,
				},
				Status: infrav1alpha1.RepositoryStatus{},
			},
			expectedErr: errors.New("Repo update failed"),
		},
		"Repo exists, update fails": {
			ctx:         nil,
			giteaClient: NephioGiteaClientMock{},
			cr: infrav1alpha1.Repository{
				Status: infrav1alpha1.RepositoryStatus{},
			},
			expectedErr: nil,
		},
		"Create repo: cr fields not blank": {
			ctx: nil,
			giteaClient: NephioGiteaClientMock{
				getRepoError: errors.New("Repo does not exist"),
			},
			cr: infrav1alpha1.Repository{
				Spec: infrav1alpha1.RepositorySpec{
					Description:   &dummyString,
					Private:       &dummyBool,
					IssueLabels:   &dummyString,
					Gitignores:    &dummyString,
					License:       &dummyString,
					Readme:        &dummyString,
					DefaultBranch: &dummyString,
					TrustModel:    &dummyTrustModel,
				},
			},
			expectedErr: nil,
		},
		"Create repo: cr fields blank": {
			ctx: nil,
			giteaClient: NephioGiteaClientMock{
				getRepoError: errors.New("Repo does not exist"),
			},
			cr:          infrav1alpha1.Repository{},
			expectedErr: nil,
		},
		"Create repo: fails": {
			ctx: nil,
			giteaClient: NephioGiteaClientMock{
				getRepoError:    errors.New("Repo does not exist"),
				createRepoError: errors.New("Error creating repo"),
			},
			cr:          infrav1alpha1.Repository{},
			expectedErr: errors.New("Error creating repo"),
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			r := reconciler{}
			err := r.upsertRepo(tc.ctx, tc.giteaClient, &tc.cr)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestDeleteRepo(t *testing.T) {

	testCases := map[string]struct {
		ctx         context.Context
		giteaClient NephioGiteaClientMock
		cr          infrav1alpha1.Repository
		expectedErr error
	}{
		"User Info and Delete repo both work": {
			ctx:         nil,
			giteaClient: NephioGiteaClientMock{},
			cr:          infrav1alpha1.Repository{},
			expectedErr: nil,
		},
		"User Info reports error": {
			ctx: nil,
			giteaClient: NephioGiteaClientMock{
				myUserInfoError: errors.New("Error getting User Information"),
			},
			cr:          infrav1alpha1.Repository{},
			expectedErr: errors.New("Error getting User Information"),
		},
		"Delete repo reports error": {
			ctx: nil,
			giteaClient: NephioGiteaClientMock{
				deleteRepoError: errors.New("Error deleting repo"),
			},
			cr:          infrav1alpha1.Repository{},
			expectedErr: errors.New("Error deleting repo"),
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			r := reconciler{}
			err := r.deleteRepo(tc.ctx, tc.giteaClient, &tc.cr)
			require.Equal(t, tc.expectedErr, err)
		})
	}
}
