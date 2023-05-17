/*
Copyright 2023 The Nephio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"context"
	"strings"

	"github.com/nephio-project/nephio/controllers/pkg/cluster/capi"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Cluster struct {
	client.Client
}

func (r Cluster) GetClusterClient(secret *corev1.Secret) (ClusterClient, bool) {
	switch string(secret.Type) {
	case "cluster.x-k8s.io/secret":
		if strings.Contains(secret.GetName(), "kubeconfig") {
			return &capi.Capi{Client: r.Client, Secret: secret}, true
		}
	}
	return nil, false
}

type ClusterClient interface {
	GetClusterClient(context.Context) (resource.APIPatchingApplicator, bool, error)
	GetClusterName() string
}
