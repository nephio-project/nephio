/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"
	focomv1alpha1 "github.com/dekstroza/focom-operator/api/focom/v1alpha1"
	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FocomProvisioningRequestReconciler reconciles a FocomProvisioningRequest object
type FocomProvisioningRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Finalizer used for deleting remote CR
const focomFinalizer = "focom.nephio.org/finalizer"

// +kubebuilder:rbac:groups=focom.nephio.org,resources=focomprovisioningrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=provisioning.oran.org,resources=templateinfoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=focom.nephio.org,resources=focomprovisioningrequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=focom.nephio.org,resources=focomprovisioningrequests/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *FocomProvisioningRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("FocomProvisioningRequest", req.NamespacedName)

	// 1. Fetch the local CR
	var fpr focomv1alpha1.FocomProvisioningRequest
	if err := r.Get(ctx, req.NamespacedName, &fpr); err != nil {
		if k8serrors.IsNotFound(err) {
			// CR was deleted before we got here
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Check if being deleted
	if !fpr.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &fpr, logger)
	}

	// 3. Ensure finalizer
	if requeue, err := r.ensureFinalizer(ctx, &fpr); err != nil {
		return ctrl.Result{}, err
	} else if requeue {
		// we updated the CR (added finalizer), requeue
		return ctrl.Result{Requeue: true}, nil
	}

	// 4. Validate CR
	if err := r.validateTemplateAlignment(ctx, &fpr); err != nil {
		logger.Error(err, "Template alignment check failed")
		r.updateStatus(&fpr, "failed", err.Error())
		_ = r.Status().Update(ctx, &fpr)
		return ctrl.Result{}, nil
	}

	// 5. Build remote client from OCloud
	remoteCl, err := r.buildRemoteClient(ctx, &fpr)
	if err != nil {
		logger.Error(err, "Failed to build remote cluster client")
		r.updateStatus(&fpr, "failed", err.Error())
		_ = r.Status().Update(ctx, &fpr)
		return ctrl.Result{}, nil
	}

	// 6. Ensure remote resource
	requeueAfter, err := r.ensureRemoteResource(ctx, remoteCl, &fpr, logger)
	if err != nil {
		// We already set status inside ensureRemoteResource
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// If the CR is being deleted, handle finalizer cleanup
func (r *FocomProvisioningRequestReconciler) handleDeletion(
	ctx context.Context,
	fpr *focomv1alpha1.FocomProvisioningRequest,
	logger logr.Logger,
) (ctrl.Result, error) {

	if controllerutil.ContainsFinalizer(fpr, focomFinalizer) {
		// Attempt to delete remote
		if err := r.deleteRemoteResource(ctx, fpr); err != nil {
			logger.Error(err, "Failed to delete remote ProvisioningRequest, will requeue")
			return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
		}
		// remove finalizer
		controllerutil.RemoveFinalizer(fpr, focomFinalizer)
		if err := r.Update(ctx, fpr); err != nil {
			return ctrl.Result{}, err
		}
	}
	// done
	return ctrl.Result{}, nil
}

// ensureFinalizer checks/sets the finalizer. Returns (requeueNeeded, error).
func (r *FocomProvisioningRequestReconciler) ensureFinalizer(
	ctx context.Context,
	fpr *focomv1alpha1.FocomProvisioningRequest,
) (bool, error) {
	if !controllerutil.ContainsFinalizer(fpr, focomFinalizer) {
		controllerutil.AddFinalizer(fpr, focomFinalizer)
		if err := r.Update(ctx, fpr); err != nil {
			return false, err
		}
		return true, nil // we updated the CR, so we should requeue
	}
	return false, nil // no update needed
}

func (r *FocomProvisioningRequestReconciler) ensureRemoteResource(
	ctx context.Context,
	remoteCl client.Client,
	fpr *focomv1alpha1.FocomProvisioningRequest,
	logger logr.Logger,
) (time.Duration, error) {

	// If no remote resource name, create it
	if fpr.Status.RemoteName == "" {
		remoteName, err := r.createRemoteProvisioningRequest(ctx, remoteCl, fpr)
		if err != nil {
			logger.Error(err, "Failed to create remote ProvisioningRequest")
			r.updateStatus(fpr, "failed", err.Error())
			_ = r.Status().Update(ctx, fpr)
			return 0, err
		}
		fpr.Status.RemoteName = remoteName
		r.updateStatus(fpr, "provisioning", "Remote CR created, waiting for fulfillment")
		if uerr := r.Status().Update(ctx, fpr); uerr != nil {
			return 0, uerr
		}
		// requeue to poll
		return 20 * time.Second, nil
	}

	// Else poll
	done, phase, msg, err := r.pollRemoteProvisioningRequest(ctx, remoteCl, fpr)
	if err != nil {
		logger.Error(err, "Failed to poll remote ProvisioningRequest")
		r.updateStatus(fpr, "failed", fmt.Sprintf("poll error: %v", err))
		_ = r.Status().Update(ctx, fpr)
		// requeue to keep trying
		return 30 * time.Second, err
	}

	// Update local status
	r.updateStatus(fpr, phase, msg)
	if uerr := r.Status().Update(ctx, fpr); uerr != nil {
		return 0, uerr
	}

	if done {
		// no further requeues needed
		return 0, nil
	}
	return 30 * time.Second, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FocomProvisioningRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&focomv1alpha1.FocomProvisioningRequest{}).
		Complete(r)
}
