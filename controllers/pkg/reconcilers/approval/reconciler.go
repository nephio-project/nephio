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
	porchconds "github.com/nephio-project/nephio/controllers/pkg/porch/condition"
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

// +kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions/status,verbs=get
// +kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions/approval,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	cfg, ok := c.(*ctrlconfig.ControllerConfig)
	if !ok {
		return nil, fmt.Errorf("cannot initialize, expecting controllerConfig, got: %s", reflect.TypeOf(c).Name())
	}

	r.Client = mgr.GetClient()
	r.porchRESTClient = cfg.PorchRESTClient
	r.recorder = mgr.GetEventRecorderFor("approval-controller")

	return nil, ctrl.NewControllerManagedBy(mgr).
		For(&porchv1alpha1.PackageRevision{}).
		Complete(r)
}

// reconciler reconciles a NetworkInstance object
type reconciler struct {
	client.Client
	porchRESTClient rest.Interface
	recorder        record.EventRecorder

	l logr.Logger
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.l = log.FromContext(ctx).WithValues("req", req)

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

	// If it is published, ignore it
	if porchv1alpha1.LifecycleIsPublished(pr.Spec.Lifecycle) {
		return ctrl.Result{}, nil
	}

	// Delay if needed
	// This is a workaround for some "settling" that seems to be needed
	// in Porch and/or PackageVariant. We should be able to remove it if
	// we can fix that.
	requeue, err := r.manageDelay(ctx, pr)
	if err != nil {
		r.recorder.Eventf(pr, corev1.EventTypeWarning,
			"Error", "error processing %q: %s", DelayAnnotationName, err.Error())

		return ctrl.Result{}, err
	}

	// if requeue is > 0, then we should do nothing more with this PackageRevision
	// for at least that long
	if requeue > 0 {
		return ctrl.Result{RequeueAfter: requeue}, nil
	}

	// Check for the approval policy annotation
	policy, ok := pr.GetAnnotations()[PolicyAnnotationName]
	if !ok {
		// no policy set, so just return, we are done
		return ctrl.Result{}, nil
	}

	// All policies require readiness gates to be met, so if they
	// are not, we are done for now.
	if !porchconds.PackageRevisionIsReady(pr.Spec.ReadinessGates, pr.Status.Conditions) {
		r.recorder.Event(pr, corev1.EventTypeNormal,
			"NotApproved", "readiness gates not met")

		return ctrl.Result{}, nil
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

		return ctrl.Result{}, nil
	}

	// policy met
	if pr.Spec.Lifecycle == porchv1alpha1.PackageRevisionLifecycleDraft {
		pr.Spec.Lifecycle = porchv1alpha1.PackageRevisionLifecycleProposed
		err = r.Update(ctx, pr)
	} else {
		err = porchclient.UpdatePackageRevisionApproval(ctx, r.porchRESTClient, client.ObjectKey{
			Namespace: pr.Namespace,
			Name:      pr.Name,
		}, porchv1alpha1.PackageRevisionLifecyclePublished)
	}

	if err != nil {
		r.recorder.Eventf(pr, corev1.EventTypeWarning,
			"Error", "error approving: %s", err.Error())
	}

	return ctrl.Result{}, err
}

func (r *reconciler) manageDelay(ctx context.Context, pr *porchv1alpha1.PackageRevision) (time.Duration, error) {
	delay, ok := pr.GetAnnotations()[DelayAnnotationName]
	if !ok {
		delay = "30s"
	}
	d, err := time.ParseDuration(delay)
	if err != nil {
		return 0, fmt.Errorf("error parsing delay duration: %w", err)
	}

	// force at least a 30 second delay
	if d < 30*time.Second {
		d = 30 * time.Second
	}

	if time.Since(pr.CreationTimestamp.Time) > d {
		return 0, nil
	}

	return d, nil
}

func (r *reconciler) policyInitial(ctx context.Context, pr *porchv1alpha1.PackageRevision) (bool, error) {
	var prList porchv1alpha1.PackageRevisionList
	if err := r.Client.List(ctx, &prList); err != nil {
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
