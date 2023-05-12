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
	"os"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/nokia/k8s-ipam/pkg/meta"
	"github.com/pkg/errors"

	"code.gitea.io/sdk/gitea"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	ctrlconfig "github.com/nephio-project/nephio/controllers/repo-operator/controllers/config"
	"github.com/nephio-project/nephio/controllers/repo-operator/pkg/giteaclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	finalizer = "infra.nephio.org/finalizer"
	// errors
	errGetCr        = "cannot get cr"
	errUpdateStatus = "cannot update status"
)

//+kubebuilder:rbac:groups=infra.nephio.org,resources=repositories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.nephio.org,resources=repositories/status,verbs=get;update;patch

// SetupWithManager sets up the controller with the Manager.
func Setup(mgr ctrl.Manager, options *ctrlconfig.ControllerConfig) error {
	r := &reconciler{
		APIPatchingApplicator: resource.NewAPIPatchingApplicator(mgr.GetClient()),
		giteaClient:           options.GiteaClient,
		finalizer:             resource.NewAPIFinalizer(mgr.GetClient(), finalizer),
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("repositoryController").
		For(&infrav1alpha1.Repository{}).
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

	cr := &infrav1alpha1.Repository{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		// if the resource no longer exists the reconcile loop is done
		if resource.IgnoreNotFound(err) != nil {
			r.l.Error(err, "cannot get resource")
			return ctrl.Result{}, errors.Wrap(resource.IgnoreNotFound(err), "cannot get resource")
		}
		return ctrl.Result{}, nil
	}

	giteaClient := r.giteaClient.Get()
	if giteaClient == nil {
		err := fmt.Errorf("gitea server unreachable")
		r.l.Error(err, "cannot connect to gitea server")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	if meta.WasDeleted(cr) {

		if err := r.deleteRepo(ctx, giteaClient, cr); err != nil {
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

	if err := r.finalizer.AddFinalizer(ctx, cr); err != nil {
		// If this is the first time we encounter this issue we'll be requeued
		// implicitly when we update our status with the new error condition. If
		// not, we requeue explicitly, which will trigger backoff.
		r.l.Error(err, "cannot add finalizer")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	// create repo
	if err := r.createRepo(ctx, giteaClient, cr); err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}
	// create token and secret
	if err := r.createAccessToken(ctx, giteaClient, cr); err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}
	cr.SetConditions(infrav1alpha1.Ready())
	return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
}

func (r *reconciler) createRepo(ctx context.Context, giteaClient *gitea.Client, cr *infrav1alpha1.Repository) error {
	repos, _, err := giteaClient.ListMyRepos(gitea.ListReposOptions{})
	if err != nil {
		r.l.Error(err, "cannot list repo")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return err
	}
	repoFound := false
	for _, repo := range repos {
		if repo.Name == cr.GetName() {
			repoFound = true
			cr.Status.URL = &repo.CloneURL
			break
		}
	}

	createRepo := gitea.CreateRepoOption{Name: cr.GetName()}
	if cr.Spec.DefaultBranch != nil {
		createRepo.DefaultBranch = *cr.Spec.DefaultBranch
	}
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

	if !repoFound {
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
	}
	return nil
}

func (r *reconciler) createAccessToken(ctx context.Context, giteaClient *gitea.Client, cr *infrav1alpha1.Repository) error {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: os.Getenv("GIT_NAMESPACE"),
		Name:      os.Getenv("GIT_SECRET_NAME"),
	},
		secret); err != nil {
			r.l.Error(err, "cannot get secret")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return errors.Wrap(err, "cannot get secret")
	}

	tokens, _, err := giteaClient.ListAccessTokens(gitea.ListAccessTokensOptions{})
	if err != nil {
		r.l.Error(err, "cannot list repo")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return err
	}
	tokenFound := false
	for _, repo := range tokens {
		if repo.Name == cr.GetName() {
			tokenFound = true
			break
		}
	}
	if !tokenFound {
		token, _, err := giteaClient.CreateAccessToken(gitea.CreateAccessTokenOption{
			Name: cr.GetName(),
		})
		if err != nil {
			r.l.Error(err, "cannot create token")
			cr.SetConditions(infrav1alpha1.Failed(err.Error()))
			return err
		}
		r.l.Info("token created", "name", cr.GetName())
		// owner reference dont work since this is a cross-namespace resource
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.Identifier(),
				Kind:       reflect.TypeOf(corev1.Secret{}).Name(),
			},
			Data: map[string][]byte{
				"username": secret.Data["username"],
				"password": []byte(token.Token),
			},
			Type: corev1.SecretTypeBasicAuth,
		}
		if err := r.Apply(ctx, secret); err != nil {
			cr.SetConditions(infrav1alpha1.Failed(err.Error()))
			r.l.Error(err, "cannot create secret")
			return err
		}
		r.l.Info("secret access token created", "name", cr.GetName())
	}
	return nil
}

func (r *reconciler) deleteRepo(ctx context.Context, giteaClient *gitea.Client, cr *infrav1alpha1.Repository) error {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.Identifier(),
			Kind:       reflect.TypeOf(corev1.Secret{}).Name(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: os.Getenv("POD_NAMESPACE"),
			Name:      fmt.Sprintf("%s-%s", cr.GetName(), "access-token"),
		},
	}
	err := r.Delete(ctx, secret)
	if resource.IgnoreNotFound(err) != nil {
		r.l.Error(err, "cannot delete access token secret")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return err
	}

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
	_, err = giteaClient.DeleteAccessToken(cr.GetName())
	if err != nil {
		r.l.Error(err, "cannot delete repo")
		cr.SetConditions(infrav1alpha1.Failed(err.Error()))
		return err
	}
	r.l.Info("access token deleted", "name", cr.GetName())
	return nil
}
