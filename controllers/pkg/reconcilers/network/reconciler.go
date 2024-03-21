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

package network

import (
	"context"
	"fmt"
	"reflect"

	configv1alpha1 "github.com/henderiw-nephio/network/apis/config/v1alpha1"
	infra2v1alpha1 "github.com/henderiw-nephio/network/apis/infra2/v1alpha1"
	"github.com/henderiw-nephio/network/pkg/endpoints"
	"github.com/henderiw-nephio/network/pkg/ipam"
	"github.com/henderiw-nephio/network/pkg/network"
	"github.com/henderiw-nephio/network/pkg/nodes"
	"github.com/henderiw-nephio/network/pkg/resources"
	ctrlconfig "github.com/nephio-project/nephio/controllers/pkg/reconcilers/config"

	//"github.com/henderiw-nephio/network/pkg/targets"
	"github.com/henderiw-nephio/network/pkg/vlan"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	reconcilerinterface "github.com/nephio-project/nephio/controllers/pkg/reconcilers/reconciler-interface"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	invv1alpha1 "github.com/nokia/k8s-ipam/apis/inv/v1alpha1"
	resourcev1alpha1 "github.com/nokia/k8s-ipam/apis/resource/common/v1alpha1"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/resource/ipam/v1alpha1"
	vlanv1alpha1 "github.com/nokia/k8s-ipam/apis/resource/vlan/v1alpha1"
	"github.com/nokia/k8s-ipam/pkg/meta"
	"github.com/nokia/k8s-ipam/pkg/proxy/clientproxy"
	"github.com/openconfig/ygot/ygot"

	"github.com/pkg/errors"
	"github.com/srl-labs/ygotsrl/v22"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	reconcilerinterface.Register("networks", &reconciler{})
}

const (
	finalizer        = "infra.nephio.org/finalizer"
	nokiaSRLProvider = "srl.nokia.com"
	// errors
	errGetCr        = "cannot get cr"
	errUpdateStatus = "cannot update status"
)

//+kubebuilder:rbac:groups=infra.nephio.org,resources=networks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infra.nephio.org,resources=networks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ipam.resource.nephio.org,resources=networkinstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ipam.resource.nephio.org,resources=networkinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ipam.resource.nephio.org,resources=ipprefixes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ipam.resource.nephio.org,resources=ipprefixes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.resource.nephio.org,resources=networks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.resource.nephio.org,resources=networks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=inv.nephio.org,resources=endpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=inv.nephio.org,resources=endpoints/status,verbs=get;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {

	cfg, ok := c.(*ctrlconfig.ControllerConfig)
	if !ok {
		return nil, fmt.Errorf("cannot initialize, expecting controllerConfig, got: %s", reflect.TypeOf(c).Name())
	}

	if err := infrav1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}
	if err := ipamv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}
	if err := vlanv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}
	if err := invv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}
	if err := configv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	r.APIPatchingApplicator = resource.NewAPIPatchingApplicator(mgr.GetClient())
	r.finalizer = resource.NewAPIFinalizer(mgr.GetClient(), finalizer)
	r.devices = map[string]*ygotsrl.Device{}
	r.VlanClientProxy = cfg.VlanClientProxy
	r.IpamClientProxy = cfg.IpamClientProxy
	//r.targets = cfg.Targets

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named("NetworkController").
		For(&infrav1alpha1.Network{}).
		Owns(&ipamv1alpha1.NetworkInstance{}).
		Owns(&vlanv1alpha1.VLANIndex{}).
		Owns(&configv1alpha1.Network{}).
		Watches(&invv1alpha1.Endpoint{}, &endpointEventHandler{client: mgr.GetClient()}).
		Watches(&invv1alpha1.Endpoint{}, &nodeEventHandler{client: mgr.GetClient()}).
		Complete(r)

}

type reconciler struct {
	resource.APIPatchingApplicator
	finalizer       *resource.APIFinalizer
	IpamClientProxy clientproxy.Proxy[*ipamv1alpha1.NetworkInstance, *ipamv1alpha1.IPClaim]
	VlanClientProxy clientproxy.Proxy[*vlanv1alpha1.VLANIndex, *vlanv1alpha1.VLANClaim]

	devices map[string]*ygotsrl.Device
	//targets   targets.Target
	resources resources.Resources // get initialized for every cr/reconcile loop
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("reconcile", "req", req)

	network := &infrav1alpha1.Network{}
	if err := r.Get(ctx, req.NamespacedName, network); err != nil {
		// if the resource no longer exists the reconcile loop is done
		if resource.IgnoreNotFound(err) != nil {
			log.Error(err, errGetCr)
			return ctrl.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetCr)
		}
		return ctrl.Result{}, nil
	}

	// TODO validation
	// validate interface/node or selector
	// validate in rt + bd -> the interface/node or selector is coming from the bd

	if meta.WasDeleted(network) {
		if err := r.finalizer.RemoveFinalizer(ctx, network); err != nil {
			log.Error(err, "cannot remove finalizer")
			network.SetConditions(infrav1alpha1.Failed(err.Error()))
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, network), errUpdateStatus)
		}

		log.Info("Successfully deleted resource")
		return ctrl.Result{Requeue: false}, nil
	}

	// add finalizer to avoid deleting the token w/o it being deleted from the git server
	if err := r.finalizer.AddFinalizer(ctx, network); err != nil {
		log.Error(err, "cannot add finalizer")
		network.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, network), errUpdateStatus)
	}

	eps, err := r.getProviderEndpoints(ctx, network.Spec.Topology)
	if err != nil {
		log.Error(err, "cannot list provider endpoints")
		network.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, network), errUpdateStatus)
	}

	nodes, err := r.getProviderNodes(ctx, network.Spec.Topology)
	if err != nil {
		log.Error(err, "cannot list provider nodes")
		network.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, network), errUpdateStatus)
	}

	r.resources = resources.New(
		r.APIPatchingApplicator,
		resources.Config{
			CR:             network,
			MatchingLabels: resourcev1alpha1.GetOwnerLabelsFromCR(network),
			Owns: []schema.GroupVersionKind{
				configv1alpha1.NetworkGroupVersionKind,
			},
		},
	)

	log.Info("apply initial resources")
	if err := r.applyInitialresources(ctx, network, eps, nodes); err != nil {
		log.Error(err, "cannot apply initial resources")
		network.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, network), errUpdateStatus)
	}

	log.Info("get new resources")
	if err := r.getNewResources(ctx, network, eps, nodes); err != nil {
		log.Error(err, "cannot get new resources")
		network.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, network), errUpdateStatus)
	}

	log.Info("apply all resources")
	if err := r.resources.APIApply(ctx); err != nil {
		log.Error(err, "cannot apply resources to the API")
		network.SetConditions(infrav1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, network), errUpdateStatus)
	}

	network.SetConditions(infrav1alpha1.Ready())
	return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, network), errUpdateStatus)
}

func getMatchingNodeLabels(cr client.Object, nodeName string) client.MatchingLabels {
	labels := resourcev1alpha1.GetOwnerLabelsFromCR(cr)
	labels[invv1alpha1.NephioNodeNameKey] = nodeName
	return labels
}

func (r *reconciler) getProviderEndpoints(ctx context.Context, topology string) (*endpoints.Endpoints, error) {
	opts := []client.ListOption{
		client.MatchingLabels{
			invv1alpha1.NephioProviderKey: nokiaSRLProvider,
			invv1alpha1.NephioTopologyKey: topology,
		},
	}
	eps := &invv1alpha1.EndpointList{}
	if err := r.List(ctx, eps, opts...); err != nil {
		log.FromContext(ctx).Error(err, "cannot list endpoints")
		return nil, err
	}
	return &endpoints.Endpoints{EndpointList: eps}, nil
}

func (r *reconciler) getProviderNodes(ctx context.Context, topology string) (*nodes.Nodes, error) {
	opts := []client.ListOption{
		client.MatchingLabels{
			invv1alpha1.NephioProviderKey: nokiaSRLProvider,
			invv1alpha1.NephioTopologyKey: topology,
		},
	}
	nos := &invv1alpha1.NodeList{}
	if err := r.List(ctx, nos, opts...); err != nil {
		log.FromContext(ctx).Error(err, "cannot list nodes")
		return nil, err
	}
	return &nodes.Nodes{NodeList: nos}, nil
}

func (r *reconciler) applyInitialresources(ctx context.Context, cr *infrav1alpha1.Network, eps *endpoints.Endpoints, nodes *nodes.Nodes) error {
	n := network.New(&network.Config{
		Config:    &infra2v1alpha1.NetworkConfig{},
		Apply:     true,
		Resources: r.resources,
		Endpoints: eps,
		Nodes:     nodes,
		Ipam:      ipam.NewIPAM(r.IpamClientProxy),
		Vlan:      vlan.NewVLAN(r.VlanClientProxy),
	})

	if err := n.Run(ctx, cr); err != nil {
		log.FromContext(ctx).Error(err, "cannot execute network run")
		return err
	}
	if err := r.resources.APIApply(ctx); err != nil {
		log.FromContext(ctx).Error(err, "cannot apply resources to the API")
		return err
	}
	return nil
}

func (r *reconciler) getNewResources(ctx context.Context, cr *infrav1alpha1.Network, eps *endpoints.Endpoints, nodes *nodes.Nodes) error {
	n := network.New(&network.Config{
		Config:    &infra2v1alpha1.NetworkConfig{},
		Apply:     false,
		Resources: r.resources,
		Endpoints: eps,
		Nodes:     nodes,
		Ipam:      ipam.NewIPAM(r.IpamClientProxy),
		Vlan:      vlan.NewVLAN(r.VlanClientProxy),
	})

	if err := n.Run(ctx, cr); err != nil {
		log.FromContext(ctx).Error(err, "cannot execute network run")
		return err
	}

	// list all networkConfigs
	opts := []client.ListOption{
		resourcev1alpha1.GetOwnerLabelsFromCR(cr),
		client.InNamespace(cr.Namespace),
	}
	ncs := &configv1alpha1.NetworkList{}
	if err := r.List(ctx, ncs, opts...); err != nil {
		return err
	}
	networkConfigs := map[string]configv1alpha1.Network{}
	for _, nc := range ncs.Items {
		networkConfigs[nc.Name] = nc
	}

	for nodeName, device := range n.GetDevices() {
		log.FromContext(ctx).Info("node config", "nodeName", nodeName)

		j, err := ygot.EmitJSON(device, &ygot.EmitJSONConfig{
			Format: ygot.RFC7951,
			Indent: "  ",
			RFC7951Config: &ygot.RFC7951JSONConfig{
				AppendModuleName: true,
			},
			SkipValidation: false,
		})
		if err != nil {
			log.FromContext(ctx).Error(err, "cannot construct json device info")
			return err
		}

		o := configv1alpha1.BuildNetworkConfig(
			metav1.ObjectMeta{
				Name:            fmt.Sprintf("%s-%s", cr.Name, nodeName),
				Namespace:       cr.Namespace,
				Labels:          getMatchingNodeLabels(cr, nodeName),
				OwnerReferences: []metav1.OwnerReference{{APIVersion: cr.APIVersion, Kind: cr.Kind, Name: cr.Name, UID: cr.UID, Controller: pointer.Bool(true)}},
			}, configv1alpha1.NetworkSpec{
				Config: runtime.RawExtension{
					Raw: []byte(j),
				},
			}, configv1alpha1.NetworkStatus{})
		if existingNetwNodeConfig, ok := networkConfigs[fmt.Sprintf("%s-%s", cr.Name, nodeName)]; ok {
			o.Status.LastAppliedConfig = existingNetwNodeConfig.Status.LastAppliedConfig
		}

		r.resources.AddNewResource(o)
	}
	return nil
}
