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

package token

import (
	"context"
	"fmt"
	"reflect"

	"code.gitea.io/sdk/gitea"
	"github.com/go-logr/logr"
	commonv1alpha1 "github.com/nephio-project/api/common/v1alpha1"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	"github.com/nephio-project/nephio/controllers/pkg/giteaclient"
	ctrlconfig "github.com/nephio-project/nephio/controllers/pkg/reconcilers/config"
	reconcilerinterface "github.com/nephio-project/nephio/controllers/pkg/reconcilers/reconciler-interface"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	reconcilerinterface.Register("tokens", &reconciler{})
}

const (
	finalizer = "infra.nephio.org/finalizer"
	// errors
	errGetCr        = "cannot get cr"
	errUpdateStatus = "cannot update status"
)

//+kubebuilder:rbac:groups=infra.nephio.org,resources=tokens,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.nephio.org,resources=tokens/status,verbs=get;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(mgr ctrl.Manager, c interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	cfg, ok := c.(*ctrlconfig.ControllerConfig)
	if !ok {
		return nil, fmt.Errorf("cannot initialize, expecting controllerConfig, got: %s", reflect.TypeOf(c).Name())
	}

	if err := infrav1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	r.APIPatchingApplicator = resource.NewAPIPatchingApplicator(mgr.GetClient())
	r.giteaClient = cfg.GiteaClient
	r.finalizer = resource.NewAPIFinalizer(mgr.GetClient(), finalizer)

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named("TokenController").
		For(&infrav1alpha1.Token{}).
		Complete(r)
}

type reconciler struct {
	resource.APIPatchingApplicator
	giteaClient giteaclient.GiteaClient
	finalizer   *resource.APIFinalizer

	l logr.Logger
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.l = log.FromContext(ctx)
	r.l.Info("reconcile", "req", req)

	cr := &infrav1alpha1.Token{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		// if the resource no longer exists the reconcile loop is done
		if resource.IgnoreNotFound(err) != nil {
			r.l.Error(err, "cannot get resource")
			return ctrl.Result{Requeue: true}, errors.Wrap(resource.IgnoreNotFound(err), "cannot get resource")
		}
		return ctrl.Result{}, nil
	}

	// check if client exists otherwise retry
	giteaClient := r.giteaClient.Get()
	if giteaClient == nil {
		err := fmt.Errorf("gitea server unreachable")
		r.l.Error(err, "cannot connect to gitea server")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	if resource.WasDeleted(cr) {
		// TODO DELETION POLICY: "orphan" deletion policy
		// token being deleted
		// Delete the token from the git server
		// when successfull remove the finalizer
		if cr.Spec.Lifecycle.DeletionPolicy == commonv1alpha1.DeletionDelete {
			if err := r.deleteToken(ctx, giteaClient, cr); err != nil {
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
		}

		if err := r.finalizer.RemoveFinalizer(ctx, cr); err != nil {
			r.l.Error(err, "cannot remove finalizer")
			cr.SetConditions(infrav1alpha1.Failed(err.Error()))
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}

		r.l.Info("Successfully deleted resource")
		return ctrl.Result{Requeue: false}, nil
	}

	// add finalizer to avoid deleting the token w/o it being deleted from the git server
	if err := r.finalizer.AddFinalizer(ctx, cr); err != nil {
		r.l.Error(err, "cannot add finalizer")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	// create token and secret
	if err := r.createToken(ctx, giteaClient, cr); err != nil {
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}
	cr.SetConditions(infrav1alpha1.Ready())
	return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
}

func (r *reconciler) createToken(ctx context.Context, giteaClient *gitea.Client, cr *infrav1alpha1.Token) error {
	tokens, _, err := giteaClient.ListAccessTokens(gitea.ListAccessTokensOptions{})
	if err != nil {
		r.l.Error(err, "cannot list repo")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return err
	}
	tokenFound := false
	for _, repo := range tokens {
		if repo.Name == cr.GetTokenName() {
			tokenFound = true
			break
		}
	}
	if !tokenFound {
		u, _, err := giteaClient.GetMyUserInfo()
		if err != nil {
			r.l.Error(err, "cannot get user info")
			cr.SetConditions(infrav1alpha1.Failed(err.Error()))
			return err
		}

		token, _, err := giteaClient.CreateAccessToken(gitea.CreateAccessTokenOption{
			Name: cr.GetTokenName(),
			Scopes: []gitea.AccessTokenScope{
				gitea.AccessTokenScopeRepo,
			},
		})
		if err != nil {
			r.l.Error(err, "cannot create token")
			cr.SetConditions(infrav1alpha1.Failed(err.Error()))
			return err
		}
		r.l.Info("token created", "name", cr.GetName())
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.Identifier(),
				Kind:       reflect.TypeOf(corev1.Secret{}).Name(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   cr.GetNamespace(),
				Name:        cr.GetName(),
				Annotations: cr.GetAnnotations(),
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: cr.APIVersion,
						Kind:       cr.Kind,
						Name:       cr.Name,
						UID:        cr.UID,
						Controller: pointer.Bool(true),
					},
				},
			},
			Data: map[string][]byte{
				"username": []byte(u.UserName),
				"password": []byte(token.Token), // needed for porch
				"token":    []byte(token.Token), // needed for configsync
			},
			Type: corev1.SecretTypeBasicAuth,
		}
		if err := r.Apply(ctx, secret); err != nil {
			cr.SetConditions(infrav1alpha1.Failed(err.Error()))
			r.l.Error(err, "cannot create secret")
			return err
		}
		r.l.Info("secret for token created", "name", cr.GetName())
	}
	return nil
}

func (r *reconciler) deleteToken(ctx context.Context, giteaClient *gitea.Client, cr *infrav1alpha1.Token) error {
	_, err := giteaClient.DeleteAccessToken(cr.GetTokenName())
	if err != nil {
		r.l.Error(err, "cannot delete token")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return err
	}
	r.l.Info("token deleted", "name", cr.GetTokenName())
	return nil
}
