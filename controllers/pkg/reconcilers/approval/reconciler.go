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

package approval

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"k8s.io/client-go/rest"

	ctrlconfig "github.com/nephio-project/nephio/controllers/pkg/reconcilers/config"
	reconcilerinterface "github.com/nephio-project/nephio/controllers/pkg/reconcilers/reconciler-interface"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"k8s.io/client-go/tools/record"

	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/go-logr/logr"
	porchclient "github.com/nephio-project/nephio/controllers/pkg/porch/client"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DelayAnnotationName          = "approval.nephio.org/delay"
	DelayConditionType           = "approval.nephio.org.DelayExpired"
	PolicyAnnotationName         = "approval.nephio.org/policy"
	InitialPolicyAnnotationValue = "initial"
)

func init() {
	reconcilerinterface.Register("approval", &reconciler{})
}

// +kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions,verbs=get;list;watch
// +kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions/approval,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	cfg, ok := c.(*ctrlconfig.ControllerConfig)
	if !ok {
		return nil, fmt.Errorf("cannot initialize, expecting controllerConfig, got: %s", reflect.TypeOf(c).Name())
	}

	r.Client = mgr.GetClient()
	r.porchClient = cfg.PorchClient
	r.porchRESTClient = cfg.PorchRESTClient
	r.recorder = mgr.GetEventRecorderFor("approval-controller")

	return nil, ctrl.NewControllerManagedBy(mgr).
		For(&porchv1alpha1.PackageRevision{}).
		Complete(r)
}

// reconciler reconciles a NetworkInstance object
type reconciler struct {
	client.Client
	porchClient     client.Client
	porchRESTClient rest.Interface
	recorder        record.EventRecorder

	l logr.Logger
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.l = log.FromContext(ctx).WithValues("req", req)
	r.l.Info("reconcile approval")

	pr := &porchv1alpha1.PackageRevision{}
	if err := r.Get(ctx, req.NamespacedName, pr); err != nil {
		// There's no need to requeue if we no longer exist. Otherwise we'll be
		// requeued implicitly because we return an error.
		if resource.IgnoreNotFound(err) != nil {
			r.l.Error(err, "cannot get resource")
			return ctrl.Result{}, errors.Wrap(resource.IgnoreNotFound(err), "cannot get resource")
		}
		return ctrl.Result{}, nil
	}

	// Check for the annotation to add a delay condition and gate
	if delay, ok := pr.GetAnnotations()[DelayAnnotationName]; ok {
		requeue, err := r.manageDelay(ctx, delay, pr)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error processing %q: %w", DelayAnnotationName, err)
		}

		// if requeue is > 0, then we should do nothing more with this PackageRevision
		// for at least that long
		if requeue > 0 {
			return ctrl.Result{RequeueAfter: requeue}, nil
		}
	}

	// Check for the approval policy annotation
	policy, ok := pr.GetAnnotations()[PolicyAnnotationName]
	if !ok {
		// no policy set, so just return, we are done
		return ctrl.Result{}, nil
	}

	// Index our conditions
	conds := make(map[string]porchv1alpha1.Condition)
	for _, c := range pr.Status.Conditions {
		conds[c.Type] = c
	}

	// Check if the readiness gates are met
	ready := true
	for _, g := range pr.Spec.ReadinessGates {
		if _, ok := conds[g.ConditionType]; !ok {
			continue
		}
		ready = ready && conds[g.ConditionType].Status == "True"
	}

	// All policies require readiness gates to be met, so if they
	// are not, we are done for now.
	if !ready {
		r.recorder.Event(pr, corev1.EventTypeNormal,
			"NotApproved", "ReadinessGates not met")

		return ctrl.Result{}, nil
	}

	// Readiness is met, so check our other policies
	approve := false
	var err error
	switch policy {
	case InitialPolicyAnnotationValue:
		approve, err = r.policyInitial(ctx, pr)
	default:
		r.recorder.Eventf(pr, corev1.EventTypeWarning,
			"InvalidPolicy", "Invalid %q annotation value: %q", PolicyAnnotationName, policy)

		return ctrl.Result{}, nil
	}

	if err != nil {
		r.recorder.Eventf(pr, corev1.EventTypeWarning,
			"Error", "Error evaluating approval policy %q: %s", policy, err.Error())

		return ctrl.Result{}, nil
	}

	if !approve {
		r.recorder.Eventf(pr, corev1.EventTypeNormal,
			"NotApproved", "Automated approval policy %q not met", policy)

		return ctrl.Result{}, nil
	}

	// policy met, do the approval
	pr.Spec.Lifecycle = porchv1alpha1.PackageRevisionLifecyclePublished

	err = porchclient.UpdatePackageRevisionApproval(ctx, r.porchRESTClient, client.ObjectKey{
		Namespace: pr.Namespace,
		Name:      pr.Name,
	}, porchv1alpha1.PackageRevisionLifecyclePublished)

	if err != nil {
		r.recorder.Eventf(pr, corev1.EventTypeWarning,
			"Error", "Error approving: %s", err.Error())
	}

	return ctrl.Result{}, nil
}

func (r *reconciler) manageDelay(ctx context.Context, delay string, pr *porchv1alpha1.PackageRevision) (time.Duration, error) {
	r.l.Info("Found delay annotation, but not yet implemented")
	return 0, nil
}

func (r *reconciler) policyInitial(ctx context.Context, pr *porchv1alpha1.PackageRevision) (bool, error) {
	return false, nil
}
