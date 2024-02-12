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
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/nephio-project/nephio/controllers/pkg/giteaclient"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"github.com/nephio-project/nephio/testing/mockeryutils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"code.gitea.io/sdk/gitea"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
)

type fields struct {
	APIPatchingApplicator resource.APIPatchingApplicator
	giteaClient           giteaclient.GiteaClient
	finalizer             *resource.APIFinalizer
	l                     logr.Logger
}
type args struct {
	ctx         context.Context
	giteaClient giteaclient.GiteaClient
	cr          *infrav1alpha1.Repository
}
type repoTest struct {
	name    string
	fields  fields
	args    args
	mocks   []mockeryutils.MockHelper
	wantErr bool
}


func TestUpsertRepo(t *testing.T) {
	dummyString := "Dummy String"
	dummyBool := true
	dummyTrustModel := infrav1alpha1.TrustModel("Trust Model")

	tests := []repoTest{
		{
			name:   "User Info reports error",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil, log.FromContext(context.Background())},
			args:   args{nil, nil, &infrav1alpha1.Repository{}},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "GetMyUserInfo", ArgType: []string{}, RetArgList: []interface{}{nil, nil, fmt.Errorf("error getting User Information")}},
			},
			wantErr: true,
		},
		{
			name:   "Repo exists, cr spec fields blank",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil, log.FromContext(context.Background())},
			args:   args{nil, nil, &infrav1alpha1.Repository{Status: infrav1alpha1.RepositoryStatus{}}},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "GetMyUserInfo", ArgType: []string{}, RetArgList: []interface{}{&gitea.User{UserName: "gitea"}, nil, nil}},
				{MethodName: "GetRepo", ArgType: []string{"string", "string"}, RetArgList: []interface{}{&gitea.Repository{}, nil, nil}},
				{MethodName: "EditRepo", ArgType: []string{"string", "string", "gitea.EditRepoOption"}, RetArgList: []interface{}{&gitea.Repository{}, nil, nil}},
			},
			wantErr: false,
		},
		{
			name:   "Repo exists, cr spec fields not blank",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil, log.FromContext(context.Background())},
			args: args{
				nil,
				nil,
				&infrav1alpha1.Repository{
					Spec: infrav1alpha1.RepositorySpec{
						Description: &dummyString,
						Private:     &dummyBool,
					},
					Status: infrav1alpha1.RepositoryStatus{},
				},
			},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "GetMyUserInfo", ArgType: []string{}, RetArgList: []interface{}{&gitea.User{UserName: "gitea"}, nil, nil}},
				{MethodName: "GetRepo", ArgType: []string{"string", "string"}, RetArgList: []interface{}{&gitea.Repository{}, nil, nil}},
				{MethodName: "EditRepo", ArgType: []string{"string", "string", "gitea.EditRepoOption"}, RetArgList: []interface{}{&gitea.Repository{}, nil, nil}},
			},
			wantErr: false,
		},
		{
			name:   "Repo exists, update fails",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil, log.FromContext(context.Background())},
			args: args{
				nil,
				nil,
				&infrav1alpha1.Repository{},
			},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "GetMyUserInfo", ArgType: []string{}, RetArgList: []interface{}{&gitea.User{UserName: "gitea"}, nil, nil}},
				{MethodName: "GetRepo", ArgType: []string{"string", "string"}, RetArgList: []interface{}{&gitea.Repository{}, nil, nil}},
				{MethodName: "EditRepo", ArgType: []string{"string", "string",
					"gitea.EditRepoOption"}, RetArgList: []interface{}{&gitea.Repository{}, nil, fmt.Errorf("error updating repo")}},
			},
			wantErr: true,
		},
		{
			name:   "Create repo: cr fields not blank",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil, log.FromContext(context.Background())},
			args: args{
				nil,
				nil,
				&infrav1alpha1.Repository{
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
			},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "GetMyUserInfo", ArgType: []string{}, RetArgList: []interface{}{&gitea.User{UserName: "gitea"}, nil, nil}},
				{MethodName: "GetRepo", ArgType: []string{"string", "string"}, RetArgList: []interface{}{&gitea.Repository{}, nil, fmt.Errorf("repo does not exist")}},
				{MethodName: "CreateRepo", ArgType: []string{"gitea.CreateRepoOption"}, RetArgList: []interface{}{&gitea.Repository{}, nil, nil}},
			},
			wantErr: false,
		},
		{
			name:   "Create repo: cr fields blank",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil, log.FromContext(context.Background())},
			args: args{
				nil,
				nil,
				&infrav1alpha1.Repository{},
			},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "GetMyUserInfo", ArgType: []string{}, RetArgList: []interface{}{&gitea.User{UserName: "gitea"}, nil, nil}},
				{MethodName: "GetRepo", ArgType: []string{"string", "string"}, RetArgList: []interface{}{&gitea.Repository{}, nil, fmt.Errorf("repo does not exist")}},
				{MethodName: "CreateRepo", ArgType: []string{"gitea.CreateRepoOption"}, RetArgList: []interface{}{&gitea.Repository{}, nil, nil}},
			},
			wantErr: false,
		},
		{
			name:   "Create repo: fails",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil, log.FromContext(context.Background())},
			args: args{
				nil,
				nil,
				&infrav1alpha1.Repository{},
			},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "GetMyUserInfo", ArgType: []string{}, RetArgList: []interface{}{&gitea.User{UserName: "gitea"}, nil, nil}},
				{MethodName: "GetRepo", ArgType: []string{"string", "string"}, RetArgList: []interface{}{&gitea.Repository{}, nil, fmt.Errorf("repo does not exist")}},
				{MethodName: "CreateRepo", ArgType: []string{"gitea.CreateRepoOption"}, RetArgList: []interface{}{&gitea.Repository{}, nil, fmt.Errorf("repo creation fails")}},
			},
			wantErr: true,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &reconciler{
				APIPatchingApplicator: tt.fields.APIPatchingApplicator,
				giteaClient:           tt.fields.giteaClient,
				finalizer:             tt.fields.finalizer,
			}

			initMockeryMocks(&tt)

			if err := r.upsertRepo(tt.args.ctx, tt.args.giteaClient, tt.args.cr); (err != nil) != tt.wantErr {
				t.Errorf("upsertRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteRepo(t *testing.T) {
	tests := []repoTest{
		{
			name:   "User Info and Delete Repo both OK",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil, log.FromContext(context.Background())},
			args: args{
				nil,
				nil,
				&infrav1alpha1.Repository{
					ObjectMeta: v1.ObjectMeta{
						Name: "repo-name",
					},
				},
			},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "GetMyUserInfo", ArgType: []string{}, RetArgList: []interface{}{&gitea.User{UserName: "gitea"}, nil, nil}},
				{MethodName: "DeleteRepo", ArgType: []string{"string", "string"}, RetArgList: []interface{}{&gitea.Response{}, nil, nil}},
			},
			wantErr: false,
		}, {
			name:   "User Info reports error",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil, log.FromContext(context.Background())},
			args: args{
				nil,
				nil,
				&infrav1alpha1.Repository{
					ObjectMeta: v1.ObjectMeta{
						Name: "repo-name",
					},
				},
			},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "GetMyUserInfo", ArgType: []string{}, RetArgList: []interface{}{&gitea.User{UserName: "gitea"}, nil, fmt.Errorf("Error getting User Information")}},
			},
			wantErr: true,
		}, {
			name:   "Delete Repo reports error",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil, log.FromContext(context.Background())},
			args: args{
				nil,
				nil,
				&infrav1alpha1.Repository{
					ObjectMeta: v1.ObjectMeta{
						Name: "repo-name",
					},
				},
			},
			mocks: []mockeryutils.MockHelper{
				{MethodName: "GetMyUserInfo", ArgType: []string{}, RetArgList: []interface{}{&gitea.User{UserName: "gitea"}, nil, nil}},
				{MethodName: "DeleteRepo", ArgType: []string{"string", "string"}, RetArgList: []interface{}{&gitea.Response{}, fmt.Errorf("Error deleting repo")}},
			},
			wantErr: true,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &reconciler{
				APIPatchingApplicator: tt.fields.APIPatchingApplicator,
				giteaClient:           tt.fields.giteaClient,
				finalizer:             tt.fields.finalizer,
			}

			initMockeryMocks(&tt)

			if err := r.deleteRepo(tt.args.ctx, tt.args.giteaClient, tt.args.cr); (err != nil) != tt.wantErr {
				t.Errorf("deleteRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func initMockeryMocks(tt *repoTest) {
	mockGClient := new(giteaclient.MockGiteaClient)
	tt.args.giteaClient = mockGClient
	tt.fields.giteaClient = mockGClient
	mockeryutils.InitMocks(&mockGClient.Mock, tt.mocks)
}
