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

package repository

import (
	"context"
	"fmt"
	"reflect"

	commonv1alpha1 "github.com/nephio-project/api/common/v1alpha1"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	git "github.com/nephio-project/nephio/controllers/pkg/git"
	giteaclient "github.com/nephio-project/nephio/controllers/pkg/git/gitea"
	"github.com/nephio-project/nephio/controllers/pkg/git/types"
	ctrlconfig "github.com/nephio-project/nephio/controllers/pkg/reconcilers/config"
	reconcilerinterface "github.com/nephio-project/nephio/controllers/pkg/reconcilers/reconciler-interface"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	reconcilerinterface.Register("repositories", &reconciler{})
}

const (
	finalizer       = "infra.nephio.org/finalizer"
	errUpdateStatus = "cannot update status"
)

//+kubebuilder:rbac:groups=infra.nephio.org,resources=repositories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.nephio.org,resources=repositories/status,verbs=get;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	cfg, ok := c.(*ctrlconfig.ControllerConfig)
	if !ok {
		return nil, fmt.Errorf("cannot initialize, expecting controllerConfig, got: %s", reflect.TypeOf(c).Name())
	}

	// Sending the porchclient to git server, this will be used to get
	// the secret objects for git server client authentication. The client
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
	// if githubClient, err := githubclient.GetClient(ctx, porchClient); err == nil {
	//     r.gitClients[git.ProviderGitHub] = githubClient
	// }

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
		Named("RepositoryController").
		For(&infrav1alpha1.Repository{}).
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

	cr := &infrav1alpha1.Repository{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		// if the resource no longer exists the reconcile loop is done
		if resource.IgnoreNotFound(err) != nil {
			log.Error(err, "cannot get resource")
			return ctrl.Result{}, errors.Wrap(resource.IgnoreNotFound(err), "cannot get resource")
		}
		return ctrl.Result{}, nil
	}

	// Detect provider from spec (defaults to gitea for backward compatibility)
	provider := git.ProviderGitea
	if cr.Spec.Provider != nil {
		provider = git.ProviderType(*cr.Spec.Provider)
		log.Info("detected git provider from spec", "provider", provider)
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
		// repo being deleted
		// Delete the repo from the git server
		// when successful remove the finalizer
		if cr.Spec.Lifecycle.DeletionPolicy == commonv1alpha1.DeletionDelete {
			if err := r.deleteRepo(ctx, gitClient, cr); err != nil {
				log.Error(err, "cannot delete repo in git server")
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

	// add finalizer to avoid deleting the repo w/o it being deleted from the git server
	if err := r.finalizer.AddFinalizer(ctx, cr); err != nil {
		log.Error(err, "cannot add finalizer")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	// upsert repo in git server
	if err := r.upsertRepo(ctx, gitClient, cr); err != nil {
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}
	cr.SetConditions(infrav1alpha1.Ready())
	return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
}

func (r *reconciler) upsertRepo(ctx context.Context, gitClient git.Client, cr *infrav1alpha1.Repository) error {
	log := log.FromContext(ctx)
	u, _, err := gitClient.GetMyUserInfo()
	if err != nil {
		log.Error(err, "cannot get user info")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return err
	}

	_, _, err = gitClient.GetRepo(u.UserName, cr.GetName())
	if err != nil {
		// create repo
		createRepo := types.CreateRepoOption{Name: cr.GetName()}
		if cr.Spec.Description != nil {
			createRepo.Description = *cr.Spec.Description
		}
		if cr.Spec.Private != nil {
			createRepo.Private = *cr.Spec.Private
		}
		if cr.Spec.IssueLabels != nil {
			createRepo.IssueLabels = *cr.Spec.IssueLabels
		}
		if cr.Spec.Gitignores != nil {
			createRepo.Gitignores = *cr.Spec.Gitignores
		}
		if cr.Spec.License != nil {
			createRepo.License = *cr.Spec.License
		}
		if cr.Spec.Readme != nil {
			createRepo.Readme = *cr.Spec.Readme
		}
		if cr.Spec.DefaultBranch != nil {
			createRepo.DefaultBranch = *cr.Spec.DefaultBranch
		}
		if cr.Spec.TrustModel != nil {
			createRepo.TrustModel = types.TrustModel(*cr.Spec.TrustModel)
		}
		createRepo.AutoInit = true
		log.Info("repository", "config", createRepo)

		repo, _, err := gitClient.CreateRepo(createRepo)
		if err != nil {
			log.Error(err, "cannot create repo")
			// Here we don't provide the full error since the message change every time and this will re-trigger
			// a new reconcile loop
			cr.SetConditions(infrav1alpha1.Failed("cannot create repo"))
			return err
		}
		log.Info("repo created", "name", cr.GetName())
		cr.Status.URL = &repo.CloneURL
		return nil
	}
	editRepo := types.EditRepoOption{Name: ptr.To(cr.GetName())}
	if cr.Spec.Description != nil {
		editRepo.Description = cr.Spec.Description
	} else {
		editRepo.Description = nil
	}
	if cr.Spec.Private != nil {
		editRepo.Private = cr.Spec.Private
	} else {
		editRepo.Private = nil
	}
	repo, _, err := gitClient.EditRepo(u.UserName, cr.GetName(), editRepo)
	if err != nil {
		log.Error(err, "cannot update repo")
		// Here we don't provide the full error since the message change every time and this will re-trigger
		// a new reconcile loop
		cr.SetConditions(infrav1alpha1.Failed("cannot update repo"))
		return err
	}
	log.Info("repo updated", "name", cr.GetName())
	cr.Status.URL = &repo.CloneURL

	return nil
}

func (r *reconciler) deleteRepo(ctx context.Context, gitClient git.Client, cr *infrav1alpha1.Repository) error {
	log := log.FromContext(ctx)
	u, _, err := gitClient.GetMyUserInfo()
	if err != nil {
		log.Error(err, "cannot get user info")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return err
	}

	_, err = gitClient.DeleteRepo(u.UserName, cr.GetName())
	if err != nil {
		log.Error(err, "cannot delete repo")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return err
	}
	log.Info("repo deleted", "name", cr.GetName())
	return nil
}
