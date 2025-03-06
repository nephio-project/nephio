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

package giteaclient

import (
	"context"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

func TestClient(t *testing.T) {
	ctx := ctrl.SetupSignalHandler()

	type args struct {
		ctx    context.Context
		client resource.APIPatchingApplicator
	}
	tests := []struct {
		name    string
		args    args
		want    GiteaClient
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
