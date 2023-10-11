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

package vlanspecializer

import (
	"context"
	"fmt"
	"reflect"

	porchcondition "github.com/nephio-project/nephio/controllers/pkg/porch/condition"
	ctrlconfig "github.com/nephio-project/nephio/controllers/pkg/reconcilers/config"
	reconcilerinterface "github.com/nephio-project/nephio/controllers/pkg/reconcilers/reconciler-interface"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	"github.com/nephio-project/nephio/krm-functions/lib/kptrl"
	function "github.com/nephio-project/nephio/krm-functions/vlan-fn/fn"
	vlanv1alpha1 "github.com/nokia/k8s-ipam/apis/resource/vlan/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

func init() {
	reconcilerinterface.Register("vlanspecializer", &reconciler{})
}

type Config struct {
	For         corev1.ObjectReference
	PorchClient client.Client
	KRMfunction fn.ResourceListProcessor
}

// +kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions/status,verbs=get;update;patch
// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	cfg, ok := c.(*ctrlconfig.ControllerConfig)
	if !ok {
		return nil, fmt.Errorf("cannot initialize, expecting controllerConfig, got: %s", reflect.TypeOf(c).Name())
	}

	f := &function.FnR{ClientProxy: cfg.VlanClientProxy}

	r.For = corev1.ObjectReference{
		APIVersion: vlanv1alpha1.SchemeBuilder.GroupVersion.Identifier(),
		Kind:       vlanv1alpha1.VLANClaimKind,
	}
	r.Client = mgr.GetClient()
	r.porchClient = cfg.PorchClient
	r.krmfn = fn.ResourceListProcessorFunc(f.Run)

	// TBD how does the proxy cache work with the injector for updates
	return nil, ctrl.NewControllerManagedBy(mgr).
		Named("VlanSpecializer").
		For(&porchv1alpha1.PackageRevision{}).
		Complete(r)
}

// reconciler reconciles a NetworkInstance object
type reconciler struct {
	client.Client
	For         corev1.ObjectReference
	porchClient client.Client
	krmfn       fn.ResourceListProcessor

	l logr.Logger
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.l = log.FromContext(ctx).WithValues("req", req)
	r.l.Info("reconcile specializer")

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
	// we just check for forResource conditions and we don't care if it is satisfied already
	// this allows us to refresh the allocation.
	ct := kptfilelibv1.GetConditionType(&r.For)
	if porchcondition.HasSpecificTypeConditions(pr.Status.Conditions, ct) {
		// get package revision resourceList
		prr := &porchv1alpha1.PackageRevisionResources{}
		if err := r.porchClient.Get(ctx, req.NamespacedName, prr); err != nil {
			r.l.Error(err, "cannot get package revision resources")
			return ctrl.Result{}, errors.Wrap(err, "cannot get package revision resources")
		}
		// get resourceList from resources
		rl, err := kptrl.GetResourceList(prr.Spec.Resources)
		if err != nil {
			r.l.Error(err, "cannot get resourceList")
			return ctrl.Result{}, errors.Wrap(err, "cannot get resourceList")
		}

		// run the function SDK
		_, err = r.krmfn.Process(rl)
		if err != nil {
			r.l.Error(err, "function run failed")
			// TBD if we need to return here + check if kptfile is set
			//return ctrl.Result{}, errors.Wrap(err, "function run failed")
		}
		for _, o := range rl.Items {
			r.l.Info("resourceList", "data", o.String())
			// TBD what if we create new resources
			// update the resources with the latest info
			prr.Spec.Resources[o.GetAnnotation(kioutil.PathAnnotation)] = o.String()
		}
		kptfile := rl.Items.GetRootKptfile()
		if kptfile == nil {
			r.l.Error(fmt.Errorf("mandatory Kptfile is missing from the package"), "")
			return ctrl.Result{}, nil
		}

		kptf := kptfilelibv1.KptFile{Kptfile: rl.Items.GetRootKptfile()}
		pr.Status.Conditions = porchcondition.GetPorchConditions(kptf.GetConditions())
		if err = r.porchClient.Update(ctx, prr); err != nil {
			return ctrl.Result{}, err
		}

	}
	return ctrl.Result{}, nil
}
