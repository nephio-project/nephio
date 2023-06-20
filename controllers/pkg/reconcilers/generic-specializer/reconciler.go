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

package genericspecializer

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/go-logr/logr"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	porchcondition "github.com/nephio-project/nephio/controllers/pkg/porch/condition"
	porchutil "github.com/nephio-project/nephio/controllers/pkg/porch/util"
	ctrlconfig "github.com/nephio-project/nephio/controllers/pkg/reconcilers/config"
	reconcilerinterface "github.com/nephio-project/nephio/controllers/pkg/reconcilers/reconciler-interface"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	configinjectfn "github.com/nephio-project/nephio/krm-functions/configinject-fn/fn"
	ipamfn "github.com/nephio-project/nephio/krm-functions/ipam-fn/fn"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	"github.com/nephio-project/nephio/krm-functions/lib/kptrl"
	"github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	vlanfn "github.com/nephio-project/nephio/krm-functions/vlan-fn/fn"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/resource/ipam/v1alpha1"
	vlanv1alpha1 "github.com/nokia/k8s-ipam/apis/resource/vlan/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

func init() {
	reconcilerinterface.Register("genericspecializer", &reconciler{})
}

// +kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions/status,verbs=get;update;patch
// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	cfg, ok := c.(*ctrlconfig.ControllerConfig)
	if !ok {
		return nil, fmt.Errorf("cannot initialize, expecting controllerConfig, got: %s", reflect.TypeOf(c).Name())
	}

	if err := porchv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	r.Client = mgr.GetClient()
	r.porchClient = cfg.PorchClient
	r.recorder = mgr.GetEventRecorderFor("generic-specializer")

	r.ipamf = ipamfn.New(cfg.IpamClientProxy)
	r.ipamkrmfn = fn.ResourceListProcessorFunc(r.ipamf.Run)

	r.vlanf = vlanfn.New(cfg.VlanClientProxy)
	r.vlankrmfn = fn.ResourceListProcessorFunc(r.vlanf.Run)

	r.configInjectf = configinjectfn.New(cfg.PorchClient)
	r.configInjectkrmfn = fn.ResourceListProcessorFunc(r.configInjectf.Run)

	// TBD how does the proxy cache work with the injector for updates
	return nil, ctrl.NewControllerManagedBy(mgr).
		Named("GenericSpecializer").
		For(&porchv1alpha1.PackageRevision{}).
		Complete(r)
}

// reconciler reconciles a NetworkInstance object
type reconciler struct {
	client.Client
	ipamf             *ipamfn.FnR
	ipamkrmfn         fn.ResourceListProcessor
	vlanf             *vlanfn.FnR
	vlankrmfn         fn.ResourceListProcessor
	configInjectf     *configinjectfn.FnR
	configInjectkrmfn fn.ResourceListProcessor
	porchClient       client.Client
	recorder          record.EventRecorder

	l logr.Logger
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.l = log.FromContext(ctx).WithValues("req", req)
	r.l.Info("reconcile genericspecializer")

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

	// check if the PackageVariant has done its work
	pvReady, err := porchutil.PackageVariantReady(ctx, pr, r.porchClient)
	if err != nil {
		r.recorder.Event(pr, corev1.EventTypeWarning,
			"Error", fmt.Sprintf("could not get owning PackageVariant: %s", err.Error()))

		return ctrl.Result{}, nil
	}

	if !pvReady {
		r.recorder.Event(pr, corev1.EventTypeNormal,
			"Waiting", "owning PackageVariant not Ready")

		return ctrl.Result{}, nil
	}

	ipamFor := r.ipamf.GetConfig().For
	vlanFor := r.vlanf.GetConfig().For
	configInjectFor := r.configInjectf.GetConfig().For

	// we just check for forResource conditions and we dont care if it is satisfied already
	// this allows us to refresh the allocation.
	if porchcondition.HasSpecificTypeConditions(pr.Status.Conditions, kptfilelibv1.GetConditionType(&ipamFor)) ||
		porchcondition.HasSpecificTypeConditions(pr.Status.Conditions, kptfilelibv1.GetConditionType(&vlanFor)) ||
		porchcondition.HasSpecificTypeConditions(pr.Status.Conditions, kptfilelibv1.GetConditionType(&configInjectFor)) {

		// get package revision resourceList
		prr := &porchv1alpha1.PackageRevisionResources{}
		if err := r.porchClient.Get(ctx, req.NamespacedName, prr); err != nil {
			r.recorder.Event(pr, corev1.EventTypeWarning, "ReconcileError", fmt.Sprintf("cannot get package revision resources: %s", err.Error()))
			r.l.Error(err, "cannot get package revision resources")
			return ctrl.Result{}, errors.Wrap(err, "cannot get package revision resources")
		}
		// get resourceList from resources
		rl, err := kptrl.GetResourceList(prr.Spec.Resources)
		if err != nil {
			r.recorder.Event(pr, corev1.EventTypeWarning, "ReconcileError", fmt.Sprintf("cannot get resourceList: %s", err.Error()))
			r.l.Error(err, "cannot get resourceList")
			return ctrl.Result{}, errors.Wrap(err, "cannot get resourceList")
		}

		if porchcondition.HasSpecificTypeConditions(pr.Status.Conditions, kptfilelibv1.GetConditionType(&ipamFor)) {
			// run the function SDK
			_, err = r.ipamkrmfn.Process(rl)
			if err != nil {
				r.recorder.Event(pr, corev1.EventTypeWarning, "ReconcileError", fmt.Sprintf("ipam function: %s", err.Error()))
				r.l.Error(err, "ipam function run failed")
				return ctrl.Result{}, nil
			}
			r.l.Info("ipam specializer fn run successfull")
		}
		if porchcondition.HasSpecificTypeConditions(pr.Status.Conditions, kptfilelibv1.GetConditionType(&vlanFor)) {
			// run the function SDK
			_, err = r.vlankrmfn.Process(rl)
			if err != nil {
				r.recorder.Event(pr, corev1.EventTypeWarning, "ReconcileError", fmt.Sprintf("vlan function: %s", err.Error()))
				r.l.Error(err, "vlan function run failed")
				return ctrl.Result{}, nil
			}
			r.l.Info("vlan specializer fn run successfull")
		}
		if porchcondition.HasSpecificTypeConditions(pr.Status.Conditions, kptfilelibv1.GetConditionType(&configInjectFor)) {
			// run the function SDK
			_, err = r.configInjectkrmfn.Process(rl)
			if err != nil {
				r.recorder.Event(pr, corev1.EventTypeWarning, "ReconcileError", fmt.Sprintf("configInject function: %s", err.Error()))
				r.l.Error(err, "configInject function run failed")
				return ctrl.Result{}, nil
			}
			r.l.Info("configInject specializer fn run successfull")
		}
		workloadClusterObjs := rl.Items.Where(fn.IsGroupVersionKind(infrav1alpha1.WorkloadClusterGroupVersionKind))
		clusterName := r.getClusterName(workloadClusterObjs)

		// We want to process the functions to refresh the claims
		// but if the package is in publish state the updates cannot be done
		// so we stop here
		if porchv1alpha1.LifecycleIsPublished(pr.Spec.Lifecycle) {
			r.recorder.Event(pr, corev1.EventTypeNormal, "CannotRefreshClaims", "package is published, no update possible")
			r.l.Info("package is published, no updates possible",
				"repo", pr.Spec.RepositoryName,
				"package", pr.Spec.PackageName,
				"rev", pr.Spec.Revision,
				"clusterName", clusterName,
			)
			return ctrl.Result{}, nil
		}

		for _, o := range rl.Items {
			// TBD what if we create new resources
			// update only the resource we act upon
			if o.GetAPIVersion() == ipamFor.APIVersion && o.GetKind() == ipamFor.Kind {
				prr.Spec.Resources[o.GetAnnotation(kioutil.PathAnnotation)] = o.String()
				// Debug
				alloc, err := kubeobject.NewFromKubeObject[ipamv1alpha1.IPClaim](o)
				if err != nil {
					r.l.Error(err, "cannot get extended kubeobject")
					continue
				}
				ipAlloc, err := alloc.GetGoStruct()
				if err != nil {
					r.l.Error(err, "cannot get gostruct from kubeobject")
					continue
				}
				r.l.Info("generic specializer ip allocation", "clusterName", clusterName, "status", ipAlloc.Status)
			}
			if o.GetAPIVersion() == vlanFor.APIVersion && o.GetKind() == vlanFor.Kind {
				prr.Spec.Resources[o.GetAnnotation(kioutil.PathAnnotation)] = o.String()
				// Debug
				alloc, err := kubeobject.NewFromKubeObject[vlanv1alpha1.VLANClaim](o)
				if err != nil {
					r.l.Error(err, "cannot get extended kubeobject")
					continue
				}
				vlanAlloc, err := alloc.GetGoStruct()
				if err != nil {
					r.l.Error(err, "cannot get gostruct from kubeobject")
					continue
				}
				r.l.Info("generic specializer vlan allocation", "cluserName", clusterName, "status", vlanAlloc.Status)
			}
			if o.GetAPIVersion() == configInjectFor.APIVersion && o.GetKind() == configInjectFor.Kind {
				prr.Spec.Resources[o.GetAnnotation(kioutil.PathAnnotation)] = o.String()
				r.l.Info("generic specializer config injector", "cluserName", clusterName, "resourceName", fmt.Sprintf("%s/%s", configInjectFor.Kind, o.GetName()))
			}
			for own := range r.configInjectf.GetConfig().Owns {
				if o.GetAPIVersion() == own.APIVersion && o.GetKind() == own.Kind {
					prr.Spec.Resources[o.GetAnnotation(kioutil.PathAnnotation)] = o.String()
					r.l.Info("generic specializer config injector", "cluserName", clusterName, "resourceName", fmt.Sprintf("%s/%s", own.Kind, o.GetName()))
				}
			}

			if o.GetAPIVersion() == "kpt.dev/v1" && o.GetKind() == "Kptfile" {
				prr.Spec.Resources[o.GetAnnotation(kioutil.PathAnnotation)] = o.String()
				kptf, err := kubeobject.NewFromKubeObject[kptv1.KptFile](o)
				if err != nil {
					r.l.Error(err, "cannot get extended kubeobject")
					continue
				}
				kptfile, err := kptf.GetGoStruct()
				if err != nil {
					r.l.Error(err, "cannot get gostruct from kubeobject")
					continue
				}
				for _, c := range kptfile.Status.Conditions {
					if strings.HasPrefix(c.Type, kptfilelibv1.GetConditionType(&vlanFor)+".") ||
						strings.HasPrefix(c.Type, kptfilelibv1.GetConditionType(&ipamFor)+".") ||
						strings.HasPrefix(c.Type, kptfilelibv1.GetConditionType(&configInjectFor)+".") {
						r.l.Info("generic specializer conditions", "cluserName", clusterName, "status", c.Status, "condition", c.Type)
					}
				}
			}
		}

		kptfile := rl.Items.GetRootKptfile()
		if kptfile == nil {
			r.recorder.Event(pr, corev1.EventTypeWarning, "ReconcileError", "mandatory Kptfile is missing")
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

func (r *reconciler) getClusterName(workloadClusterObjs fn.KubeObjects) string {
	clusterName := ""
	if len(workloadClusterObjs) > 0 {
		cluster, err := kubeobject.NewFromKubeObject[infrav1alpha1.WorkloadCluster](workloadClusterObjs[0])
		if err != nil {
			r.l.Error(err, "cannot get extended kubeobject")
			return clusterName
		}
		workloadCluster, err := cluster.GetGoStruct()
		if err != nil {
			r.l.Error(err, "cannot get gostruct from kubeobject")
			return clusterName
		}
		clusterName = workloadCluster.Spec.ClusterName
	}
	return clusterName
}
