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

package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	porchconfigv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/nephio-project/nephio/controllers/pkg/cluster"
	ctrlconfig "github.com/nephio-project/nephio/controllers/pkg/reconcilers/config"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

/*
func init() {
	controllers.Register("bootstrap", &reconciler{})
}
*/

//+kubebuilder:rbac:groups="*",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters/status,verbs=get
//+kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions,verbs=get;list;watch
//+kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions/status,verbs=get

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) Setup(mgr ctrl.Manager, cfg *ctrlconfig.ControllerConfig) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	//if err := capiv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
	//	return nil, err
	//}

	r.Client = mgr.GetClient()
	r.porchClient = cfg.PorchClient

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named("BootstrapController").
		For(&corev1.Secret{}).
		Complete(r)
}

type reconciler struct {
	client.Client
	porchClient client.Client

	l logr.Logger
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.l = log.FromContext(ctx)

	cr := &corev1.Secret{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		// if the resource no longer exists the reconcile loop is done
		if resource.IgnoreNotFound(err) != nil {
			msg := "cannot get resource"
			r.l.Error(err, msg)
			return ctrl.Result{}, errors.Wrap(resource.IgnoreNotFound(err), msg)
		}
		return reconcile.Result{}, nil
	}

	// if the secret is being deleted dont do anything for now
	if cr.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	// this branch handles installing the secrets to the remote cluster
	if cr.GetNamespace() == "config-management-system" {
		r.l.Info("reconcile")
		clusterName, ok := cr.GetAnnotations()["nephio.org/site"]
		if !ok {
			return reconcile.Result{}, nil
		}
		if clusterName != "mgmt" {
			secrets := &corev1.SecretList{}
			if err := r.List(ctx, secrets); err != nil {
				msg := "cannot list secrets"
				r.l.Error(err, msg)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, errors.Wrap(err, msg)
			}
			found := false
			for _, secret := range secrets.Items {
				if strings.Contains(secret.GetName(), clusterName) {
					clusterClient, ok := cluster.Cluster{Client: r.Client}.GetClusterClient(&secret)
					if ok {
						found = true
						clusterClient, ready, err := clusterClient.GetClusterClient(ctx)
						if err != nil {
							msg := "cannot get clusterClient"
							r.l.Error(err, msg)
							return ctrl.Result{RequeueAfter: 30 * time.Second}, errors.Wrap(err, msg)
						}
						if !ready {
							r.l.Info("cluster not ready")
							return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
						}
						ns := &corev1.Namespace{}
						if err = clusterClient.Get(ctx, types.NamespacedName{Name: cr.GetNamespace()}, ns); err != nil {
							if resource.IgnoreNotFound(err) != nil {
								msg := fmt.Sprintf("cannot get namespace: %s", secret.GetNamespace())
								r.l.Error(err, msg)
								return ctrl.Result{RequeueAfter: 30 * time.Second}, errors.Wrap(err, msg)
							}
							msg := fmt.Sprintf("namespace: %s, does not exist, retry...", cr.GetNamespace())
							r.l.Info(msg)
							return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
						}

						if err := clusterClient.Apply(ctx, cr); err != nil {
							msg := fmt.Sprintf("cannot apply secret to cluster %s", clusterName)
							r.l.Error(err, msg)
							return ctrl.Result{RequeueAfter: 10 * time.Second}, errors.Wrap(err, msg)
						}
					}
				}
			}
			if !found {
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}
		}
	} else {
		// this branch handles manifest installation
		cl, ok := cluster.Cluster{Client: r.Client}.GetClusterClient(cr)
		if ok {
			r.l.Info("reconcile")
			clusterClient, ready, err := cl.GetClusterClient(ctx)
			if err != nil {
				msg := "cannot get clusterClient"
				r.l.Error(err, msg)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, errors.Wrap(err, msg)
			}
			if !ready {
				r.l.Info("cluster not ready")
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}

			// install resources
			resources, err := r.getResources(ctx, cl.GetClusterName())
			if err != nil {
				msg := "cannot get resources"
				r.l.Error(err, msg)
				return ctrl.Result{RequeueAfter: 10 * time.Second}, errors.Wrap(err, msg)
			}
			for _, resource := range resources {
				r.l.Info("install manifest", "resources",
					fmt.Sprintf("%s.%s.%s", resource.GetAPIVersion(), resource.GetKind(), resource.GetName()))
				if err := clusterClient.Apply(ctx, &resource); err != nil {
					r.l.Error(err, "cannot apply resource to cluster", "name", resource.GetName())
				}
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *reconciler) getResources(ctx context.Context, clusterName string) ([]unstructured.Unstructured, error) {
	repos := &porchconfigv1alpha1.RepositoryList{}
	if err := r.porchClient.List(ctx, repos); err != nil {
		return nil, err
	}

	stagingRepoName := ""
	for _, repo := range repos.Items {
		if _, ok := repo.Annotations["nephio.org/staging"]; ok {
			stagingRepoName = repo.GetName()
		}
	}

	prList := &porchv1alpha1.PackageRevisionList{}
	if err := r.porchClient.List(ctx, prList); err != nil {
		return nil, err
	}

	prKeys := []types.NamespacedName{}
	for _, pr := range prList.Items {
		if pr.Spec.RepositoryName == stagingRepoName && pr.Annotations["test"] == clusterName {
			prKeys = append(prKeys, types.NamespacedName{Name: pr.GetName(), Namespace: pr.GetNamespace()})
		}
	}
	resources := []unstructured.Unstructured{}
	for _, prKey := range prKeys {
		prr := &porchv1alpha1.PackageRevisionResources{}
		if err := r.porchClient.Get(ctx, prKey, prr); err != nil {
			r.l.Error(err, "cannot get package resvision resourcelist", "key", prKey)
			return nil, err
		}

		res, err := r.getResourcesPRR(prr.Spec.Resources)
		if err != nil {
			r.l.Error(err, "cannot get resources", "key", prKey)
			return nil, err
		}
		resources = append(resources, res...)
	}

	return resources, nil
}

func includeFile(path string, match []string) bool {
	for _, m := range match {
		file := filepath.Base(path)
		if matched, err := filepath.Match(m, file); err == nil && matched {
			return true
		}
	}
	return false
}

func (r *reconciler) getResourcesPRR(resources map[string]string) ([]unstructured.Unstructured, error) {
	inputs := []kio.Reader{}
	for path, data := range resources {
		if includeFile(path, []string{"*.yaml", "*.yml", "Kptfile"}) {
			inputs = append(inputs, &kio.ByteReader{
				Reader: strings.NewReader(data),
				SetAnnotations: map[string]string{
					kioutil.PathAnnotation: path,
				},
				DisableUnwrapping: true,
			})
		}
	}
	var pb kio.PackageBuffer
	err := kio.Pipeline{
		Inputs:  inputs,
		Filters: []kio.Filter{},
		Outputs: []kio.Writer{&pb},
	}.Execute()
	if err != nil {
		return nil, err
	}

	ul := []unstructured.Unstructured{}
	for _, n := range pb.Nodes {
		if v, ok := n.GetAnnotations()[filters.LocalConfigAnnotation]; ok && v == "true" {
			continue
		}
		u := unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(n.MustString()), &u); err != nil {
			r.l.Error(err, "cannot unmarshal data", "data", n.MustString())
			// we dont fail
			continue
		}
		ul = append(ul, u)
	}
	return ul, nil
}
