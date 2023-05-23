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

package bootstrappackages

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	porchconfigv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/nephio-project/nephio/controllers/pkg/cluster"
	ctrlconfig "github.com/nephio-project/nephio/controllers/pkg/reconcilers/config"
	reconcilerinterface "github.com/nephio-project/nephio/controllers/pkg/reconcilers/reconciler-interface"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

func init() {
	reconcilerinterface.Register("bootstrappackages", &reconciler{})
}

const (
	stagingNameKey      = "nephio.org/staging"
	clusterNameKey      = "nephio.org/cluster-name"
	configsyncNamespace = "config-management-system"
)

//+kubebuilder:rbac:groups="*",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters/status,verbs=get
//+kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions,verbs=get;list;watch
//+kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions/status,verbs=get
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=repositories,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(mgr ctrl.Manager, c any) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	cfg, ok := c.(*ctrlconfig.ControllerConfig)
	if !ok {
		return nil, fmt.Errorf("cannot initialize, expecting controllerConfig, got: %s", reflect.TypeOf(c).Name())
	}

	if err := porchv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}
	if err := porchconfigv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	r.Client = mgr.GetClient()
	r.porchClient = cfg.PorchClient

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named("BootstrapPackageController").
		For(&porchv1alpha1.PackageRevision{}).
		Complete(r)
}

type reconciler struct {
	client.Client
	porchClient client.Client

	l logr.Logger
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.l = log.FromContext(ctx)
	cr := &porchv1alpha1.PackageRevision{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		// There's no need to requeue if we no longer exist. Otherwise we'll be
		// requeued implicitly because we return an error.
		if resource.IgnoreNotFound(err) != nil {
			msg := "cannot get resource"
			r.l.Error(err, msg)
			return ctrl.Result{}, errors.Wrap(resource.IgnoreNotFound(err), msg)
		}
		return ctrl.Result{}, nil
	}

	// check if the packagerevision is part of a staging repository
	// if not we can ignore this package revision
	stagingPR, err := r.IsStagingPackageRevision(ctx, cr)
	if err != nil {
		msg := "cannot list repositories"
		r.l.Error(err, msg)
		return ctrl.Result{}, errors.Wrap(err, msg)
	}
	if stagingPR && porchv1alpha1.LifecycleIsPublished(cr.Spec.Lifecycle) {
		r.l.Info("reconcile package revision")
		resources, namespacePresent, err := r.getResources(ctx, req)
		if err != nil {
			msg := "cannot get resources"
			r.l.Error(err, msg)
			return ctrl.Result{}, errors.Wrap(err, msg)
		}
		// we expect the clusterName to be applied to all resources in the
		// package revision resources, so we find the clustername by looking at the
		// first resource in the resource list
		if len(resources) > 0 {
			clusterName, ok := resources[0].GetAnnotations()[clusterNameKey]
			if !ok {
				r.l.Info("clusterName not found",
					"resource", fmt.Sprintf("%s.%s.%s", resources[0].GetAPIVersion(), resources[0].GetKind(), resources[0].GetName()),
					"annotations", resources[0].GetAnnotations())
				return ctrl.Result{}, nil
			}
			// we need to find the cluster client
			secrets := &corev1.SecretList{}
			if err := r.List(ctx, secrets); err != nil {
				msg := "cannot list secrets"
				r.l.Error(err, msg)
				return ctrl.Result{}, errors.Wrap(err, msg)
			}
			found := false
			for _, secret := range secrets.Items {
				if strings.Contains(secret.GetName(), clusterName) {
					secret := secret // required to prevent gosec warning: G601 (CWE-118): Implicit memory aliasing in for loop
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
						if !namespacePresent {
							ns := &corev1.Namespace{}
							if err = clusterClient.Get(ctx, types.NamespacedName{Name: configsyncNamespace}, ns); err != nil {
								if resource.IgnoreNotFound(err) != nil {
									msg := fmt.Sprintf("cannot get namespace: %s", configsyncNamespace)
									r.l.Error(err, msg)
									return ctrl.Result{RequeueAfter: 30 * time.Second}, errors.Wrap(err, msg)
								}
								msg := fmt.Sprintf("namespace: %s, does not exist, retry...", configsyncNamespace)
								r.l.Info(msg)
								return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
							}
						}
						// install resources
						for _, resource := range resources {
							resource := resource // required to prevent gosec warning: G601 (CWE-118): Implicit memory aliasing in for loop
							r.l.Info("install manifest", "resource",
								fmt.Sprintf("%s.%s.%s", resource.GetAPIVersion(), resource.GetKind(), resource.GetName()))
							if err := clusterClient.Apply(ctx, &resource); err != nil {
								msg := fmt.Sprintf("cannot apply resource to cluster: resourceName: %s", resource.GetName())
								r.l.Error(err, msg)
								return ctrl.Result{}, errors.Wrap(err, msg)
							}
						}
					}
				}
			}
			if !found {
				// the clusterclient was not found, we retry
				r.l.Info("cluster client not found, retry...")
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *reconciler) IsStagingPackageRevision(ctx context.Context, cr *porchv1alpha1.PackageRevision) (bool, error) {
	repos := &porchconfigv1alpha1.RepositoryList{}
	if err := r.porchClient.List(ctx, repos); err != nil {

		return false, err
	}

	stagingRepoNames := []string{}
	for _, repo := range repos.Items {
		if _, ok := repo.Annotations[stagingNameKey]; ok {
			stagingRepoNames = append(stagingRepoNames, repo.GetName())
		}
	}
	for _, stagingRepoName := range stagingRepoNames {
		if cr.Spec.RepositoryName == stagingRepoName {
			return true, nil
		}
	}
	return false, nil
}

func (r *reconciler) getResources(ctx context.Context, req ctrl.Request) ([]unstructured.Unstructured, bool, error) {
	prr := &porchv1alpha1.PackageRevisionResources{}
	if err := r.porchClient.Get(ctx, req.NamespacedName, prr); err != nil {
		r.l.Error(err, "cannot get package resvision resourcelist", "key", req.NamespacedName)
		return nil, false, err
	}

	return r.getResourcesPRR(prr.Spec.Resources)
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

func (r *reconciler) getResourcesPRR(resources map[string]string) ([]unstructured.Unstructured, bool, error) {
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
		return nil, false, err
	}

	namespacepresent := false
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
		if u.GetKind() == reflect.TypeOf(corev1.Namespace{}).Name() {
			namespacepresent = true
		}
		ul = append(ul, u)
	}
	return ul, namespacepresent, nil
}
