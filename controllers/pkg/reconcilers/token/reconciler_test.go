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
	"github.com/go-logr/logr"
	"github.com/nephio-project/nephio/controllers/pkg/giteaclient"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
)

func TestDeleteToken(t *testing.T) {
	type mockHelper struct {
		methodName string
		argType    []string
		retArgList []interface{}
	}
	type fields struct {
		APIPatchingApplicator resource.APIPatchingApplicator
		giteaClient           giteaclient.GiteaClient
		finalizer             *resource.APIFinalizer
		l                     logr.Logger
	}
	type args struct {
		ctx         context.Context
		giteaClient giteaclient.GiteaClient
		cr          *infrav1alpha1.Token
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		mocks   []mockHelper
		wantErr bool
	}{
		{
			name:   "Delete Access token reports error",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil, log.FromContext(nil)},
			args:   args{nil, nil, &infrav1alpha1.Token{}},
			mocks: []mockHelper{
				{"DeleteAccessToken", []string{"string"}, []interface{}{nil, fmt.Errorf("\"username\" not set: only BasicAuth allowed")}},
			},
			wantErr: true,
		},
		{
			name:   "Delete Access token success",
			fields: fields{resource.NewAPIPatchingApplicator(nil), nil, nil, log.FromContext(nil)},
			args:   args{nil, nil, &infrav1alpha1.Token{}},
			mocks: []mockHelper{
				{"DeleteAccessToken", []string{"string"}, []interface{}{nil, nil}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &reconciler{
				APIPatchingApplicator: tt.fields.APIPatchingApplicator,
				giteaClient:           tt.fields.giteaClient,
				finalizer:             tt.fields.finalizer,
			}
			// The below block being setup and processing of mocks before invoking the function to be tested
			mockGClient := new(giteaclient.MockGiteaClient)
			tt.args.giteaClient = mockGClient
			tt.fields.giteaClient = mockGClient
			for counter := range tt.mocks {
				call := mockGClient.Mock.On(tt.mocks[counter].methodName)
				for _, arg := range tt.mocks[counter].argType {
					call.Arguments = append(call.Arguments, mock.AnythingOfType(arg))
				}
				for _, ret := range tt.mocks[counter].retArgList {
					call.ReturnArguments = append(call.ReturnArguments, ret)
				}
			}

			if err := r.deleteToken(tt.args.ctx, tt.args.giteaClient, tt.args.cr); (err != nil) != tt.wantErr {
				t.Errorf("deleteToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
