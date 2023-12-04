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

package bootstrapsecret

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nephio-project/nephio/controllers/pkg/cluster"
	reconcilerinterface "github.com/nephio-project/nephio/controllers/pkg/reconcilers/reconciler-interface"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	reconcilerinterface.Register("bootstrapsecrets", &reconciler{})
}

const (
	clusterNameKey      = "nephio.org/cluster-name"
	nephioAppKey        = "nephio.org/app"
	configsyncApp       = "configsync"
	bootstrapApp        = "bootstrap"
	//configsyncNamespace = "config-management-system"
)

//+kubebuilder:rbac:groups="*",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters/status,verbs=get

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c any) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	r.Client = mgr.GetClient()

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named("BootstrapSecretController").
		For(&corev1.Secret{}).
		Complete(r)
}

type reconciler struct {
	client.Client
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	cr := &corev1.Secret{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		// if the resource no longer exists the reconcile loop is done
		if resource.IgnoreNotFound(err) != nil {
			msg := "cannot get resource"
			log.Error(err, msg)
			return ctrl.Result{}, errors.Wrap(resource.IgnoreNotFound(err), msg)
		}
		return reconcile.Result{}, nil
	}

	// if the secret is being deleted don't do anything for now
	if resource.WasDeleted(cr) {
		return reconcile.Result{}, nil
	}

	// this branch handles installing the secrets to the remote cluster
	// the secret is relevant to be installed in the workload cluster if:
	// annotation key "nephio.org/app" == configsync
	// annotation key "nephio.org/cluster-name" different then "" and different then management
	if cr.GetAnnotations()[nephioAppKey] == configsyncApp &&
		cr.GetAnnotations()[clusterNameKey] != "" &&
		cr.GetAnnotations()[clusterNameKey] != "mgmt" {
		log.Info("reconcile secret")
		clusterName := cr.GetAnnotations()[clusterNameKey]

		secrets := &corev1.SecretList{}
		if err := r.List(ctx, secrets); err != nil {
			msg := "cannot list secrets"
			log.Error(err, msg)
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
						log.Error(err, msg)
						return ctrl.Result{RequeueAfter: 30 * time.Second}, errors.Wrap(err, msg)
					}
					if !ready {
						log.Info("cluster not ready")
						return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
					}
					// check if namespace exists, if not retry
					ns := &corev1.Namespace{}
					if err = clusterClient.Get(ctx, types.NamespacedName{Name: cr.Namespace}, ns); err != nil {
						if resource.IgnoreNotFound(err) != nil {
							msg := fmt.Sprintf("cannot get namespace: %s", cr.Namespace)
							log.Error(err, msg)
							return ctrl.Result{RequeueAfter: 30 * time.Second}, errors.Wrap(err, msg)
						}
						msg := fmt.Sprintf("namespace: %s, does not exist, retry...", cr.Namespace)
						log.Info(msg)
						return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
					}

					newcr := cr.DeepCopy()
					// since the original annotations are set by configsync we need to reset them
					// so apply 2 annotations to the secret: app = bootstrap +  cluster-name = clusterName
					newcr.SetAnnotations(map[string]string{
						nephioAppKey:   bootstrapApp,
						clusterNameKey: clusterName,
					})
					newcr.ResourceVersion = ""
					newcr.UID = ""
					log.Info("secret info", "secret", newcr.Annotations)
					if err := clusterClient.Apply(ctx, newcr); err != nil {
						msg := fmt.Sprintf("cannot apply secret to cluster %s", clusterName)
						log.Error(err, msg)
						return ctrl.Result{}, errors.Wrap(err, msg)
					}
				}
			}
		}
		if !found {
			// the cluster client was not found, we retry
			log.Info("cluster client not found, retry...")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}
	return ctrl.Result{}, nil
}
