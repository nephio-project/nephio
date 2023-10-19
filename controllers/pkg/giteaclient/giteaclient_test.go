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
