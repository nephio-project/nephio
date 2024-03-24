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

package openshift

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ocmv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type OpenShift struct {
	client.Client
	Secret *corev1.Secret
	l      logr.Logger
}

func (r *OpenShift) GetClusterName() string {
	if r.Secret == nil {
		return ""
	}

	// The secret will have the following label hive.openshift.io/cluster-deployment-name: ca-montreal
	name, ok := r.Secret.Labels["hive.openshift.io/cluster-deployment-name"]
	if !ok {
		r.l.Error(nil, "fail to get cluster name from secret %s", r.Secret.Name)
	}
	return name
}

func (r *OpenShift) GetClusterClient(ctx context.Context) (resource.APIPatchingApplicator, bool, error) {
	if !r.isOpenShiftClusterReady(ctx) {
		return resource.APIPatchingApplicator{}, false, nil
	}
	return getClusterClient(r.Secret, "kubeconfig")
}

func (r *OpenShift) isOpenShiftClusterReady(ctx context.Context) bool {
	r.l = log.FromContext(ctx)
	name := r.GetClusterName()

	cl := resource.GetUnstructuredFromGVK(&schema.GroupVersionKind{Group: ocmv1.GroupName, Version: ocmv1.GroupVersion.Version, Kind: reflect.TypeOf(ocmv1.ManagedCluster{}).Name()})
	if err := r.Get(ctx, types.NamespacedName{Namespace: r.Secret.GetNamespace(), Name: name}, cl); err != nil {
		r.l.Error(err, "cannot get cluster")
		return false
	}
	b, err := json.Marshal(cl)
	if err != nil {
		r.l.Error(err, "cannot marshal cluster")
		return false
	}
	cluster := &ocmv1.ManagedCluster{}
	if err := json.Unmarshal(b, cluster); err != nil {
		r.l.Error(err, "cannot unmarshal cluster")
		return false
	}
	return isReady(cluster.Status.Conditions)
}

func isReady(cs []metav1.Condition) bool {
	for _, c := range cs {
		if c.Type == ocmv1.ManagedClusterConditionAvailable {
			if c.Status == metav1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

func getClusterClient(secret *corev1.Secret, secretKey string) (resource.APIPatchingApplicator, bool, error) {
	//provide a restconfig from the secret value
	config, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[secretKey])
	if err != nil {
		return resource.APIPatchingApplicator{}, false, err
	}
	// build a cluster client from the kube rest config
	clClient, err := client.New(config, client.Options{})
	if err != nil {
		return resource.APIPatchingApplicator{}, false, err
	}
	return resource.NewAPIPatchingApplicator(clClient), true, nil
}
