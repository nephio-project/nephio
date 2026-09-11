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
	"net/http"
	"reflect"
	"time"

	commonv1alpha1 "github.com/nephio-project/api/common/v1alpha1"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	git "github.com/nephio-project/nephio/controllers/pkg/git"
	giteaclient "github.com/nephio-project/nephio/controllers/pkg/git/gitea"
	githubclient "github.com/nephio-project/nephio/controllers/pkg/git/github"
	"github.com/nephio-project/nephio/controllers/pkg/git/types"
	ctrlconfig "github.com/nephio-project/nephio/controllers/pkg/reconcilers/config"
	reconcilerinterface "github.com/nephio-project/nephio/controllers/pkg/reconcilers/reconciler-interface"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
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
	errUpdateStatus = "cannot update status"
	// GitHub installation tokens expire after 1 hour
	githubTokenRefreshInterval = 55 * time.Minute
)

//+kubebuilder:rbac:groups=infra.nephio.org,resources=tokens,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.nephio.org,resources=tokens/status,verbs=get;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	cfg, ok := c.(*ctrlconfig.ControllerConfig)
	if !ok {
		return nil, fmt.Errorf("cannot initialize, expecting controllerConfig, got: %s", reflect.TypeOf(c).Name())
	}

	// Sending the porchclient to git, this will be used to get
	// the secret objects for git client authentication. The client
	// of the manager of this controller cannot be used at this point.
	porchClient := resource.NewAPIPatchingApplicator(cfg.PorchClient)

	// Initialize git clients for all supported providers
	r.gitClients = make(map[git.ProviderType]git.Client)

	// Initialize Gitea client
	if giteaClient, err := giteaclient.GetClient(ctx, porchClient); err == nil {
		r.gitClients[git.ProviderGitea] = giteaClient
	} else {
		// Gitea client initialization failed, but continue - it might not be needed
		log.FromContext(ctx).Info("failed to initialize gitea client", "error", err)
	}

	// Future: Initialize GitHub client when supported
	if githubClient, err := githubclient.GetClient(ctx, porchClient); err == nil {
		r.gitClients[git.ProviderGitHub] = githubClient
	}

	// Future: Initialize GitLab client when supported
	// if gitlabClient, err := gitlabclient.GetClient(ctx, porchClient); err == nil {
	//     r.gitClients[git.ProviderGitLab] = gitlabClient
	// }

	if err := infrav1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	r.APIPatchingApplicator = resource.NewAPIPatchingApplicator(mgr.GetClient())
	r.finalizer = resource.NewAPIFinalizer(mgr.GetClient(), finalizer)

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named("TokenController").
		For(&infrav1alpha1.Token{}).
		Complete(r)
}

type reconciler struct {
	resource.APIPatchingApplicator
	gitClients map[git.ProviderType]git.Client
	finalizer  *resource.APIFinalizer
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("reconcile", "req", req)

	cr := &infrav1alpha1.Token{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		// if the resource no longer exists the reconcile loop is done
		if resource.IgnoreNotFound(err) != nil {
			log.Error(err, "cannot get resource")
			return ctrl.Result{Requeue: true}, errors.Wrap(resource.IgnoreNotFound(err), "cannot get resource")
		}
		return ctrl.Result{}, nil
	}

	// Detect provider from annotation (defaults to gitea for backward compatibility)
	provider := git.ProviderGitea
	if cr.Annotations != nil {
		if p, ok := cr.Annotations["nephio.org/git-provider"]; ok {
			provider = git.ProviderType(p)
			log.Info("detected git provider from annotation", "provider", provider)
		}
	}

	// Get the pre-initialized git client for the provider
	gitClient, exists := r.gitClients[provider]
	if !exists {
		// Provider not supported or client not initialized
		log.Info("git provider not supported or client not initialized", "provider", provider)
		cr.SetConditions(infrav1alpha1.Ready())
		return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	// check if client exists otherwise retry
	if !gitClient.IsInitialized() {
		err := fmt.Errorf("git server unreachable")
		log.Error(err, "cannot connect to git server")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	if resource.WasDeleted(cr) {
		// TODO DELETION POLICY: "orphan" deletion policy
		// token being deleted
		// Delete the token from the git server
		// when successful remove the finalizer
		if cr.Spec.Lifecycle.DeletionPolicy == commonv1alpha1.DeletionDelete {
			if err := r.deleteToken(ctx, gitClient, cr); err != nil {
				log.Error(err, "cannot delete token in git server")
				return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
		}

		if err := r.finalizer.RemoveFinalizer(ctx, cr); err != nil {
			log.Error(err, "cannot remove finalizer")
			cr.SetConditions(infrav1alpha1.Failed(err.Error()))
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}

		log.Info("Successfully deleted resource")
		return ctrl.Result{Requeue: false}, nil
	}

	// add finalizer to avoid deleting the token w/o it being deleted from the git server
	if err := r.finalizer.AddFinalizer(ctx, cr); err != nil {
		log.Error(err, "cannot add finalizer")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	// create token and secret
	if err := r.createToken(ctx, gitClient, cr); err != nil {
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}
	cr.SetConditions(infrav1alpha1.Ready())

	// For GitHub, requeue after 55 minutes to regenerate installation token before it expires
	if provider == git.ProviderGitHub {
		log.Info("scheduling token refresh for github", "interval", githubTokenRefreshInterval)
		return ctrl.Result{RequeueAfter: githubTokenRefreshInterval}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
}

func (r *reconciler) createToken(ctx context.Context, gitClient git.Client, cr *infrav1alpha1.Token) error {
	log := log.FromContext(ctx)

	// Detect provider from annotation
	provider := git.ProviderGitea
	if cr.Annotations != nil {
		if p, ok := cr.Annotations["nephio.org/git-provider"]; ok {
			provider = git.ProviderType(p)
		}
	}

	// For GitHub, always regenerate token (installation tokens expire after 1 hour)
	// For other providers, check if token already exists
	tokenFound := false
	if provider != git.ProviderGitHub {
		tokens, _, err := gitClient.ListAccessTokens(types.ListAccessTokensOptions{})
		if err != nil {
			log.Error(err, "cannot list tokens")
			cr.SetConditions(infrav1alpha1.Failed(err.Error()))
			return err
		}
		for _, token := range tokens {
			if token.Name == cr.GetTokenName() {
				tokenFound = true
				break
			}
		}
	} else {
		log.Info("regenerating github installation token")
	}

	if !tokenFound {
		u, _, err := gitClient.GetMyUserInfo()
		if err != nil {
			log.Error(err, "cannot get user info")
			cr.SetConditions(infrav1alpha1.Failed(err.Error()))
			return err
		}

		token, _, err := gitClient.CreateAccessToken(types.CreateAccessTokenOption{
			Name: cr.GetTokenName(),
			Scopes: []types.AccessTokenScope{
				types.AccessTokenScopeRepo,
			},
		})
		if err != nil {
			log.Error(err, "cannot create token")
			cr.SetConditions(infrav1alpha1.Failed(err.Error()))
			return err
		}
		if provider == git.ProviderGitHub {
			log.Info("github installation token refreshed", "name", cr.GetName())
		} else {
			log.Info("token created", "name", cr.GetName())
		}
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.Identifier(),
				Kind:       reflect.TypeFor[corev1.Secret]().Name(),
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
						Controller: ptr.To(true),
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
			log.Error(err, "cannot create secret")
			return err
		}
		log.Info("secret for token created", "name", cr.GetName())
	}
	return nil
}

// deleteToken removes the token from the git server. A 404 means the token is
// not there, which is the state deletion asks for, so the finalizer comes off
// rather than holding the Token open on one somebody already removed.
func (r *reconciler) deleteToken(ctx context.Context, gitClient git.Client, cr *infrav1alpha1.Token) error {
	log := log.FromContext(ctx)

	resp, err := gitClient.DeleteAccessToken(cr.GetTokenName())
	if err == nil {
		log.Info("token deleted", "name", cr.GetTokenName())
		return nil
	}

	if resp != nil && resp.Response != nil && resp.StatusCode == http.StatusNotFound {
		log.Info("token already absent on the git server", "name", cr.GetTokenName())
		return nil
	}

	log.Error(err, "cannot delete token")
	cr.SetConditions(infrav1alpha1.Failed(err.Error()))
	return err
}
