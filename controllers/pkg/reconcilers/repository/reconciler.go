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
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/go-logr/logr"
	ctrlconfig "github.com/henderiw-nephio/nephio-controllers/controllers/config"
	"github.com/henderiw-nephio/nephio-controllers/pkg/applicator"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	"github.com/nephio-project/nephio/controllers/pkg/giteaclient"
	"github.com/nokia/k8s-ipam/pkg/meta"
	"github.com/nokia/k8s-ipam/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

/*
func init() {
	controllers.Register("repositories", &reconciler{})
}
*/

const (
	finalizer = "infra.nephio.org/finalizer"
	// errors
	errGetCr        = "cannot get cr"
	errUpdateStatus = "cannot update status"
)

//+kubebuilder:rbac:groups=infra.nephio.org,resources=repositories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.nephio.org,resources=repositories/status,verbs=get;update;patch

// Setup sets up the controller with the Manager.
func (r *Reconciler) Setup(mgr ctrl.Manager, cfg *ctrlconfig.ControllerConfig) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	if err := infrav1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	r.APIPatchingApplicator = applicator.NewAPIPatchingApplicator(mgr.GetClient())
	r.giteaClient = cfg.GiteaClient
	r.finalizer = resource.NewAPIFinalizer(mgr.GetClient(), finalizer)

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named("RepositoryController").
		For(&infrav1alpha1.Repository{}).
		Complete(r)
}

type Reconciler struct {
	applicator.APIPatchingApplicator
	giteaClient giteaclient.GiteaClient
	finalizer   *resource.APIFinalizer

	l logr.Logger
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.l = log.FromContext(ctx)
	r.l.Info("reconcile", "req", req)

	cr := &infrav1alpha1.Repository{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		// if the resource no longer exists the reconcile loop is done
		if resource.IgnoreNotFound(err) != nil {
			r.l.Error(err, "cannot get resource")
			return ctrl.Result{}, errors.Wrap(resource.IgnoreNotFound(err), "cannot get resource")
		}
		return ctrl.Result{}, nil
	}

	// check if client exists otherwise retry
	giteaClient := r.giteaClient.Get()
	if giteaClient == nil {
		err := fmt.Errorf("gitea server unreachable")
		r.l.Error(err, "cannot connect to git server")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	if meta.WasDeleted(cr) {
		// repo being deleted
		// Delete the repo from the git server
		// when successfull remove the finalizer
		if err := r.deleteRepo(ctx, giteaClient, cr); err != nil {
			r.l.Error(err, "cannot delete repo in git server")
			return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}

		if err := r.finalizer.RemoveFinalizer(ctx, cr); err != nil {
			r.l.Error(err, "cannot remove finalizer")
			cr.SetConditions(infrav1alpha1.Failed(err.Error()))
			return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}

		r.l.Info("Successfully deleted resource")
		return ctrl.Result{Requeue: false}, nil
	}

	// add finalizer to avoid deleting the repo w/o it being deleted from the git server
	if err := r.finalizer.AddFinalizer(ctx, cr); err != nil {
		r.l.Error(err, "cannot add finalizer")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	// upsert repo in git server
	if err := r.upsertRepo(ctx, giteaClient, cr); err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}
	cr.SetConditions(infrav1alpha1.Ready())
	return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
}

func (r *Reconciler) upsertRepo(ctx context.Context, giteaClient *gitea.Client, cr *infrav1alpha1.Repository) error {
	u, _, err := giteaClient.GetMyUserInfo()
	if err != nil {
		r.l.Error(err, "cannot get user info")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return err
	}

	_, _, err = giteaClient.GetRepo(u.UserName, cr.GetName())
	if err != nil {
		// create repo
		createRepo := gitea.CreateRepoOption{Name: cr.GetName()}
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
			createRepo.TrustModel = gitea.TrustModel(*cr.Spec.TrustModel)
		}
		createRepo.AutoInit = true
		r.l.Info("repository", "config", createRepo)

		repo, _, err := giteaClient.CreateRepo(createRepo)
		if err != nil {
			r.l.Error(err, "cannot create repo")
			// Here we dont provide the full error sicne the message change every time and this will retrigger
			// a new reconcile loop
			cr.SetConditions(infrav1alpha1.Failed("cannot create repo"))
			return err
		}
		r.l.Info("repo created", "name", cr.GetName())
		cr.Status.URL = &repo.CloneURL
	} else {
		editRepo := gitea.EditRepoOption{Name: pointer.String(cr.GetName())}
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
		repo, _, err := giteaClient.EditRepo(u.UserName, cr.GetName(), editRepo)
		if err != nil {
			r.l.Error(err, "cannot update repo")
			// Here we dont provide the full error sicne the message change every time and this will retrigger
			// a new reconcile loop
			cr.SetConditions(infrav1alpha1.Failed("cannot update repo"))
			return err
		}
		r.l.Info("repo updated", "name", cr.GetName())
		cr.Status.URL = &repo.CloneURL
	}

	return nil
}

func (r *Reconciler) deleteRepo(ctx context.Context, giteaClient *gitea.Client, cr *infrav1alpha1.Repository) error {
	u, _, err := giteaClient.GetMyUserInfo()
	if err != nil {
		r.l.Error(err, "cannot get user info")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return err
	}

	_, err = giteaClient.DeleteRepo(u.UserName, cr.GetName())
	if err != nil {
		r.l.Error(err, "cannot delete repo")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return err
	}
	r.l.Info("repo deleted", "name", cr.GetName())
	return nil
}
