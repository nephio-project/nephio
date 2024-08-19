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

	porchclient "github.com/nephio-project/nephio/controllers/pkg/porch/client"
	porchconds "github.com/nephio-project/nephio/controllers/pkg/porch/condition"
	porchutil "github.com/nephio-project/nephio/controllers/pkg/porch/util"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	porchv1alpha1 "github.com/nephio-project/porch/api/porch/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DelayAnnotationName          = "approval.nephio.org/delay"
	PolicyAnnotationName         = "approval.nephio.org/policy"
	InitialPolicyAnnotationValue = "initial"
)

func init() {
	reconcilerinterface.Register("approval", &reconciler{})
}

// +kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions/status,verbs=get
// +kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions/approval,verbs=get;update;patch
// +kubebuilder:rbac:groups=config.porch.kpt.dev,resources=packagevariants,verbs=get;list;watch
// +kubebuilder:rbac:groups=config.porch.kpt.dev,resources=packagevariants/status,verbs=get
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	cfg, ok := c.(*ctrlconfig.ControllerConfig)
	if !ok {
		return nil, fmt.Errorf("cannot initialize, expecting controllerConfig, got: %s", reflect.TypeOf(c).Name())
	}

	r.apiReader = mgr.GetAPIReader()
	r.baseClient = mgr.GetClient()
	r.porchClient = cfg.PorchClient
	r.porchRESTClient = cfg.PorchRESTClient
	r.recorder = mgr.GetEventRecorderFor("approval-controller")
	r.requeueDuration = time.Duration(cfg.ApprovalRequeueDuration) * time.Second

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named("ApprovalController").
		For(&porchv1alpha1.PackageRevision{}).
		Complete(r)
}

// reconciler reconciles a NetworkInstance object
type reconciler struct {
	apiReader       client.Reader
	baseClient      client.Client
	porchClient     client.Client
	porchRESTClient rest.Interface
	recorder        record.EventRecorder
	requeueDuration time.Duration
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("req", req)
	log.Info("reconcile approval")

	pr := &porchv1alpha1.PackageRevision{}
	if err := r.apiReader.Get(ctx, req.NamespacedName, pr); err != nil {
		// There's no need to requeue if we no longer exist. Otherwise we'll be
		// requeued implicitly because we return an error.
		if resource.IgnoreNotFound(err) != nil {
			log.Error(err, "cannot get resource")
			return ctrl.Result{}, errors.Wrap(resource.IgnoreNotFound(err), "cannot get resource")
		}
		return ctrl.Result{}, nil
	}

	// If we shouldn't process this at all, just return
	policy, ok := shouldProcess(pr)
	if !ok {
		return ctrl.Result{}, nil
	}

	// If the package revision is owned by a PackageVariant, check the Ready condition
	// of the package variant. If it is not Ready, then we should not approve yet. The
	// lack of readiness could indicate an error which even impacts whether or not the
	// readiness gates have been properly set.
	pvReady, err := porchutil.PackageVariantReady(ctx, pr, r.porchClient)
	if err != nil {
		r.recorder.Event(pr, corev1.EventTypeWarning,
			"Error", fmt.Sprintf("could not get owning PackageVariant: %s", err.Error()))

		return ctrl.Result{}, nil
	}

	if !pvReady {
		r.recorder.Event(pr, corev1.EventTypeNormal,
			"NotApproved", "owning PackageVariant not Ready")

		return ctrl.Result{RequeueAfter: r.requeueDuration}, nil
	}

	// All policies require readiness gates to be met, so if they
	// are not, we are done for now.
	if !porchconds.PackageRevisionIsReady(pr.Spec.ReadinessGates, pr.Status.Conditions) {
		r.recorder.Event(pr, corev1.EventTypeNormal,
			"NotApproved", "readiness gates not met")

		return ctrl.Result{RequeueAfter: r.requeueDuration}, nil
	}

	// Readiness is met, so check our other policies
	approve := false
	switch policy {
	case InitialPolicyAnnotationValue:
		approve, err = r.policyInitial(ctx, pr)
	default:
		r.recorder.Eventf(pr, corev1.EventTypeWarning,
			"InvalidPolicy", "invalid %q annotation value: %q", PolicyAnnotationName, policy)

		return ctrl.Result{}, nil
	}

	if err != nil {
		r.recorder.Eventf(pr, corev1.EventTypeWarning,
			"Error", "error evaluating approval policy %q: %s", policy, err.Error())

		return ctrl.Result{}, nil
	}

	if !approve {
		r.recorder.Eventf(pr, corev1.EventTypeNormal,
			"NotApproved", "approval policy %q not met", policy)

		return ctrl.Result{RequeueAfter: r.requeueDuration}, nil
	}

	// Delay if needed, and let the user know via an event
	// We should be able to get rid of this if we add a policy to check
	// the specializer condition. We need to check the *specific* condition,
	// because if the condition has not been added to the readiness gates yet,
	// we could pass all the gates even though that specific condition is missing.
	// That check shouldn't be needed if the initial clone creates the readiness gate
	// entry though (with the function pipeline run).
	requeue, err := manageDelay(pr)
	if err != nil {
		r.recorder.Eventf(pr, corev1.EventTypeWarning,
			"Error", "error processing %q: %s", DelayAnnotationName, err.Error())

		// Do not propagate the error; we do not want it to force an immediate requeue
		// If we could not parse the annotation, it is a user error
		return ctrl.Result{}, nil
	}

	// if requeue is > 0, then we should do nothing more with this PackageRevision
	// for at least that long
	if requeue > 0 {
		r.recorder.Event(pr, corev1.EventTypeNormal,
			"NotApproved", "delay time not met")
		return ctrl.Result{RequeueAfter: requeue}, nil
	}

	action := "approving"
	reason := "Approved"

	// All policies met
	if pr.Spec.Lifecycle == porchv1alpha1.PackageRevisionLifecycleDraft {
		action = "proposing"
		reason = "Proposed"
		pr.Spec.Lifecycle = porchv1alpha1.PackageRevisionLifecycleProposed
		err = r.baseClient.Update(ctx, pr)
	} else {
		err = porchclient.UpdatePackageRevisionApproval(ctx, r.porchRESTClient, client.ObjectKey{
			Namespace: pr.Namespace,
			Name:      pr.Name,
		}, porchv1alpha1.PackageRevisionLifecyclePublished)
	}

	if err != nil {
		r.recorder.Eventf(pr, corev1.EventTypeWarning,
			"Error", "error %s: %s", action, err.Error())
	} else {
		r.recorder.Eventf(pr, corev1.EventTypeNormal,
			reason, "all approval policies met")
	}

	return ctrl.Result{}, err
}

func shouldProcess(pr *porchv1alpha1.PackageRevision) (string, bool) {
	result := true

	// If it is published, ignore it
	result = result && !porchv1alpha1.LifecycleIsPublished(pr.Spec.Lifecycle)

	// Check for the approval policy annotation
	policy, ok := pr.GetAnnotations()[PolicyAnnotationName]
	result = result && ok

	return policy, result
}

func manageDelay(pr *porchv1alpha1.PackageRevision) (time.Duration, error) {
	delay, ok := pr.GetAnnotations()[DelayAnnotationName]
	if !ok {
		// only delay if there is a delay annotation
		return 0, nil
	}

	d, err := time.ParseDuration(delay)
	if err != nil {
		return 0, err
	}

	if d < 0 {
		return 0, fmt.Errorf("invalid delay %q; delay must be 0 or more", delay)
	}

	if time.Since(pr.CreationTimestamp.Time) > d {
		return 0, nil
	}

	return d, nil
}

func (r *reconciler) policyInitial(ctx context.Context, pr *porchv1alpha1.PackageRevision) (bool, error) {
	var prList porchv1alpha1.PackageRevisionList
	if err := r.baseClient.List(ctx, &prList); err != nil {
		return false, err
	}

	// do not approve if a published version exists already
	for _, pr2 := range prList.Items {
		if !porchv1alpha1.LifecycleIsPublished(pr2.Spec.Lifecycle) {
			continue
		}
		if pr2.Spec.RepositoryName == pr.Spec.RepositoryName &&
			pr2.Spec.PackageName == pr.Spec.PackageName {
			return false, nil
		}
	}

	// we did not find an already published revision of this package, so approve it
	return true, nil
}
