/*
Copyright 2022-2023 The Nephio Authors.

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

package controllers

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nfdeployv1alpha1 "github.com/nephio-project/edge-status-aggregator/api/v1alpha1"
	"github.com/nephio-project/edge-status-aggregator/deployment"
	ps "github.com/nephio-project/edge-status-aggregator/packageservice"
	"github.com/nephio-project/edge-status-aggregator/util"
)

var (
	nfDeployFinalizerName = "nfdeploy.nephio.org/nfdeployfinalizer"
)

type Options struct{}

func (o *Options) InitDefaults()                       {}
func (o *Options) BindFlags(_ string, _ *flag.FlagSet) {}

// NfDeployReconciler reconciles a NfDeploy object
type NfDeployReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	DeploymentManager deployment.DeploymentManager
	Log               logr.Logger
	PS                ps.PackageServiceInterface
	Options
}

//+kubebuilder:rbac:groups=nfdeploy.nephio.org,resources=nfdeploys,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nfdeploy.nephio.org,resources=nfdeploys/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nfdeploy.nephio.org,resources=nfdeploys/finalizers,verbs=update
//+kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions;packagerevisionresources,verbs=get;list;create;update;delete
//+kubebuilder:rbac:groups=cloud.nephio.org,resources=edgeclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions/approval,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO: Modify the Reconcile function to compare the state specified by
// the NfDeploy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *NfDeployReconciler) Reconcile(
	ctx context.Context, req ctrl.Request,
) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var nfDeploy nfdeployv1alpha1.NfDeploy
	if err := r.Get(ctx, req.NamespacedName, &nfDeploy); err != nil {
		r.Log.Error(err, "unable to fetch nfDeploy")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	isDeleted, err := r.manageNfDeployFinalizer(ctx, req)
	if err != nil {
		r.Log.Error(err, "error managing finalizer")
		return ctrl.Result{}, err
	}
	if isDeleted {
		return ctrl.Result{}, nil
	}

	if err := util.ValidateNFDeploy(nfDeploy); err != nil {
		r.Log.Error(err, "nfDeploy validation failed")
		return ctrl.Result{}, err
	}
	r.Log.Info("Started to process NfDeploy", "nfDeploy", nfDeploy.Name)

	if err := r.setInitialStatus(ctx, req, nfDeploy.Generation); err != nil {
		r.Log.Error(err, "error updating NfDeploy status", "nfDeployName", nfDeploy.Name)
		return ctrl.Result{}, err
	}

	go r.DeploymentManager.ReportNFDeployEvent(nfDeploy, req.NamespacedName)
	r.Log.Info("Reconciled successfully!", "nfDeploy", nfDeploy.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NfDeployReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, i interface{}) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nfdeployv1alpha1.NfDeploy{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *NfDeployReconciler) setInitialStatus(ctx context.Context,
	req ctrl.Request, generation int64) error {
	return r.setNfDeployStatus(ctx, req, generation,
		map[nfdeployv1alpha1.NFDeployConditionType]nfdeployv1alpha1.NFDeployCondition{
			nfdeployv1alpha1.DeploymentReconciling: {
				Type:    nfdeployv1alpha1.DeploymentReconciling,
				Status:  corev1.ConditionTrue,
				Reason:  "NewVersionAvailable",
				Message: "Reconciling NfDeploy",
			},
			nfdeployv1alpha1.DeploymentStalled: {
				Type:   nfdeployv1alpha1.DeploymentStalled,
				Status: corev1.ConditionFalse,
			},
			nfdeployv1alpha1.DeploymentPeering: {
				Type:   nfdeployv1alpha1.DeploymentPeering,
				Status: corev1.ConditionUnknown,
			},
			nfdeployv1alpha1.DeploymentReady: {
				Type:   nfdeployv1alpha1.DeploymentReady,
				Status: corev1.ConditionUnknown,
			},
		})
}

func (r *NfDeployReconciler) setNfDeployStatus(ctx context.Context,
	req ctrl.Request, generation int64,
	condMap map[nfdeployv1alpha1.NFDeployConditionType]nfdeployv1alpha1.NFDeployCondition) error {

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fetching latest nfDeploy
		var nfDeploy nfdeployv1alpha1.NfDeploy
		if err := r.Get(ctx, req.NamespacedName, &nfDeploy); err != nil {
			return err
		}

		currConditions := make(map[nfdeployv1alpha1.NFDeployConditionType]nfdeployv1alpha1.NFDeployCondition)
		now := metav1.NewTime(time.Now())
		for _, c := range nfDeploy.Status.Conditions {
			currConditions[c.Type] = c
			cond := condMap[c.Type]
			if c.Status != cond.Status {
				cond.LastTransitionTime = now
			}
			if c.Status != cond.Status || c.Reason != cond.Reason || c.Message != cond.Message {
				cond.LastUpdateTime = now
			}
			condMap[c.Type] = cond
		}
		conditions := []nfdeployv1alpha1.NFDeployCondition{}
		for _, v := range condMap {
			conditions = append(conditions, v)
		}
		nfDeploy.Status.ObservedGeneration = int32(generation)
		nfDeploy.Status.Conditions = conditions
		if err := r.Status().Update(ctx, &nfDeploy); err != nil {
			return fmt.Errorf("error updating NfDeploy status: %w", err)
		}
		return nil
	})
	return err
}

func (r *NfDeployReconciler) manageNfDeployFinalizer(ctx context.Context, req ctrl.Request) (bool, error) {
	isDeleted := false
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fetching latest nfDeploy
		var nfDeploy nfdeployv1alpha1.NfDeploy
		if err := r.Get(ctx, req.NamespacedName, &nfDeploy); err != nil {
			r.Log.Info("Exit - 1")
			return err
		}

		// examine DeletionTimestamp to determine if object is under deletion
		if nfDeploy.ObjectMeta.DeletionTimestamp.IsZero() {
			// The object is not being deleted, so if it does not have our finalizer,
			// then adding the finalizer and updating the object
			if !controllerutil.ContainsFinalizer(&nfDeploy, nfDeployFinalizerName) {
				controllerutil.AddFinalizer(&nfDeploy, nfDeployFinalizerName)
				if err := r.Update(ctx, &nfDeploy); err != nil {
					return err
				}
				r.Log.Info("Successfully added finalizer", "nfDeployName", nfDeploy.Name,
					"finalizerName", nfDeployFinalizerName)
			}
		} else {
			// The object is being deleted
			if controllerutil.ContainsFinalizer(&nfDeploy, nfDeployFinalizerName) {
				r.Log.Info("ContainsFinalizer")
				if err := r.handleResourceDeletion(ctx, &nfDeploy); err != nil {
					return err
				}
				r.Log.Info("Successfully deleted resources")

				// remove our finalizer from the list and update it.
				controllerutil.RemoveFinalizer(&nfDeploy, nfDeployFinalizerName)
				if err := r.Update(ctx, &nfDeploy); err != nil {
					return err
				}
				r.Log.Info("Successfully removed finalizer", "nfDeployName", nfDeploy.Name,
					"finalizerName", nfDeployFinalizerName)
			}
			isDeleted = true
		}
		return nil
	})
	return isDeleted, err
}

func (r *NfDeployReconciler) handleResourceDeletion(ctx context.Context, nfDeploy *nfdeployv1alpha1.NfDeploy) error {
	r.DeploymentManager.ReportNFDeployDeleteEvent(*nfDeploy)
	return nil
}
