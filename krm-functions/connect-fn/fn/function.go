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

package fn

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	ko "github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	invv1alpha1 "github.com/nokia/k8s-ipam/apis/inv/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

type connectFn struct {
	sdk             condkptsdk.KptCondSDK
	workloadCluster *infrav1alpha1.WorkloadCluster
	cluster         *capiv1beta1.Cluster
	nodes           []invv1alpha1.Node
}

func Run(rl *fn.ResourceList) (bool, error) {
	myFn := connectFn{
		nodes: []invv1alpha1.Node{},
	}
	var err error
	myFn.sdk, err = condkptsdk.New(
		rl,
		&condkptsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
				Kind:       "Link",
			},
			Owns: map[corev1.ObjectReference]condkptsdk.ResourceKind{
				{
					APIVersion: invv1alpha1.GroupVersion.Identifier(),
					Kind:       invv1alpha1.LinkKind,
				}: condkptsdk.ChildLocal,
				{
					APIVersion: invv1alpha1.GroupVersion.Identifier(),
					Kind:       invv1alpha1.EndpointKind,
				}: condkptsdk.ChildLocal,
			},
			Watch: map[corev1.ObjectReference]condkptsdk.WatchCallbackFn{
				{
					APIVersion: infrav1alpha1.GroupVersion.Identifier(),
					Kind:       infrav1alpha1.WorkloadClusterKind,
				}: myFn.WorkloadClusterCallbackFn,
				{
					APIVersion: invv1alpha1.GroupVersion.Identifier(),
					Kind:       invv1alpha1.NodeKind,
				}: myFn.NodeCallbackFn,
				{
					APIVersion: capiv1beta1.GroupVersion.Identifier(),
					Kind:       reflect.TypeOf(capiv1beta1.Cluster{}).Name(),
				}: myFn.ClusterCallbackFn,
			},
			PopulateOwnResourcesFn: myFn.desiredOwnedResourceList,
			UpdateResourceFn:       myFn.updateResource,
		},
	)
	if err != nil {
		rl.Results.ErrorE(err)
		return false, err
	}
	return myFn.sdk.Run()
}

// WorkloadClusterCallbackFn provides a callback for the workload cluster
// resources in the resourceList
func (f *connectFn) WorkloadClusterCallbackFn(o *fn.KubeObject) error {
	var err error

	if f.workloadCluster != nil {
		return fmt.Errorf("multiple WorkloadCluster objects found in the kpt package")
	}
	f.workloadCluster, err = ko.KubeObjectToStruct[infrav1alpha1.WorkloadCluster](o)
	if err != nil {
		return err
	}

	// validate check the specifics of the spec, like mandatory fields
	return f.workloadCluster.Spec.Validate()
}

// NodeCallbackFn provides a callback for the node and adds the nodes to the inventory
// resources in the resourceList
func (f *connectFn) NodeCallbackFn(o *fn.KubeObject) error {
	n, err := ko.KubeObjectToStruct[invv1alpha1.Node](o)
	if err != nil {
		return err
	}
	if n != nil {
		f.nodes = append(f.nodes, *n)
	}
	return nil
}

// ClusterCallbackFn provides a callback for the node and adds the nodes to the inventory
// resources in the resourceList
func (f *connectFn) ClusterCallbackFn(o *fn.KubeObject) error {
	var err error

	if f.cluster != nil {
		return fmt.Errorf("multiple capi cluster objects found in the kpt package")
	}
	f.cluster, err = ko.KubeObjectToStruct[capiv1beta1.Cluster](o)
	if err != nil {
		return err
	}
	return nil
}

// desiredOwnedResourceList returns with the list of all child KubeObjects
// belonging to the parent Interface "for object"
func (f *connectFn) desiredOwnedResourceList(o *fn.KubeObject) (fn.KubeObjects, error) {
	if f.workloadCluster == nil {
		// no WorkloadCluster resource in the package
		return nil, fmt.Errorf("workload cluster is missing from the kpt package")
	}
	if f.cluster == nil {
		// no capi cluster resource in the package
		return nil, fmt.Errorf("capi cluster is missing from the kpt package")
	}
	if len(f.nodes) == 0 {
		// no WorkloadCluster resource in the package
		return nil, fmt.Errorf("node is missing from the kpt package")
	}
	if len(f.nodes) > 2 {
		// no WorkloadCluster resource in the package
		return nil, fmt.Errorf("configuration not supported, we only support single homed or dual homed topolgies")
	}

	totalWorkerNodes := 0
	workerNodePrefix := ""
	if f.cluster.Spec.Topology != nil {
		/*
		WE DONT WIRE THE CONTROL PLANE AS IT IS CONNECTED VIA THE DOCKER BRIDGE
		if f.cluster.Spec.Topology.ControlPlane.Replicas != nil {
			totalClusterNodes += int(*f.cluster.Spec.Topology.ControlPlane.Replicas)
			controlPlaneNodes = int(*f.cluster.Spec.Topology.ControlPlane.Replicas)
		}
		*/
		if f.cluster.Spec.Topology.Workers != nil {
			for _, m := range f.cluster.Spec.Topology.Workers.MachineDeployments {
				if m.Replicas != nil {
					totalWorkerNodes += int(*m.Replicas)
				}
				workerNodePrefix = m.Name
			}
		}
	}
	if totalWorkerNodes == 0 {
		return nil, fmt.Errorf("configuration error, a cluster without nodes seems odd")
	}
	if totalWorkerNodes > 3 {
		return nil, fmt.Errorf("configuration not supported, max 4 nodes per cluster")
	}

	/*
		TO BE ADDED BACK IF WE HAVE A REAL REQ API OBject
		linkKOE, err := ko.NewFromKubeObject[nephioreqv1alpha1.Link](o)
		if err != nil {
			return nil, err
		}

		link, err := linkKOE.GetGoStruct()
		if err != nil {
			return nil, err
		}
	*/

	linkNodeId, err := getLinkNodeId(o.GetLabels())
	if err != nil {
		return nil, err
	}
	serverItfceName := o.GetLabels()["nephio.org/interfaceName"]
	provider := f.nodes[linkNodeId].Spec.Provider
	topologyName := f.nodes[linkNodeId].GetLabels()[invv1alpha1.NephioTopologyKey]
	linkId, err := getLinkId(o.GetLabels())
	if err != nil {
		return nil, err
	}
	if linkId == 2 && len(f.nodes) == 1 {
		return nil, fmt.Errorf("configuration not supported, 2 links only supported with redeundent nodes")
	}
	clusterName := f.workloadCluster.Spec.ClusterName
	offset, err := getClusterOffset(f.workloadCluster.Spec.ClusterName)
	if err != nil {
		return nil, err
	}

	// we assume the nodes attached to the cluster have an id that allow us to  wire them properly
	// TBD if we have another strategy going forward
	sort.SliceStable(f.nodes, func(i, j int) bool {
		return f.nodes[i].Name < f.nodes[j].Name
	})

	resources := fn.KubeObjects{}

	for i := 0; i < totalWorkerNodes; i++ {
		/* example naming bu capi kind 
			test-2r75l-8sgzr -> CP
			test-2r75l-99p9w -> CP
			test-md-0-c5zs2-599ff4b546x89n6j-sj6zg -> Worker
			test-md-0-c5zs2-599ff4b546x89n6j-sq4r6 -> Worker
			test-lb
		*/
		clusterNodeName := fmt.Sprintf("%s-%s-%d", clusterName, workerNodePrefix, i)
		ifNbr := offset + linkId + i
		netwNodeIfName := fmt.Sprintf("e1-%d", ifNbr)
		netwNodeName := fmt.Sprintf("clab-%s-%s", topologyName, f.nodes[linkNodeId].Name)
		clusterNodeIfName := serverItfceName
		
		linkName := fmt.Sprintf("%s-%s-%s-%s", clusterNodeName, clusterNodeIfName, netwNodeName, netwNodeIfName)

		linkMeta := metav1.ObjectMeta{
			Name:      linkName,
			Namespace: o.GetNamespace(),
			Labels: map[string]string{
				invv1alpha1.NephioTopologyKey: topologyName,
				invv1alpha1.NephioLinkNameKey: linkName,
			},
		}
		obj, err := f.getLink(linkMeta, invv1alpha1.LinkSpec{
			Endpoints: []invv1alpha1.LinkEndpoint{
				{
					NodeName:      clusterNodeName,
					InterfaceName: clusterNodeIfName,
				},
				{
					NodeName:      netwNodeName,
					InterfaceName: netwNodeIfName,
				},
			},
		})
		if err != nil {
			return nil, err
		}
		resources = append(resources, obj)

		epMeta := metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", netwNodeName, netwNodeIfName),
			Namespace: o.GetNamespace(),
			Labels: map[string]string{
				invv1alpha1.NephioTopologyKey:      topologyName,
				invv1alpha1.NephioNodeNameKey:      netwNodeName,
				invv1alpha1.NephioProviderKey:      provider,
				invv1alpha1.NephioInterfaceNameKey: netwNodeIfName,
				invv1alpha1.NephioLinkNameKey:      linkName,
				invv1alpha1.NephioClusterNameKey:   clusterName,
			},
		}
		obj, err = f.getEndpoint(epMeta, invv1alpha1.EndpointSpec{
			Provider: invv1alpha1.Provider{
				Provider: provider,
			},
			EndpointProperties: invv1alpha1.EndpointProperties{
				InterfaceName: netwNodeIfName,
				NodeName:      netwNodeName,
			},
		})
		if err != nil {
			return nil, err
		}
		resources = append(resources, obj)
	}

	// resources contain the list of child resources
	// belonging to the parent object

	return resources, nil
}

func (f *connectFn) updateResource(forObj *fn.KubeObject, objs fn.KubeObjects) (*fn.KubeObject, error) {
	if forObj == nil {
		return nil, fmt.Errorf("expected a for object but got nil")
	}
	return forObj, nil
}

func (f *connectFn) getLink(meta metav1.ObjectMeta, spec invv1alpha1.LinkSpec) (*fn.KubeObject, error) {
	claim := invv1alpha1.BuildLink(
		meta,
		spec,
		invv1alpha1.LinkStatus{},
	)

	return fn.NewFromTypedObject(claim)
}

func (f *connectFn) getEndpoint(meta metav1.ObjectMeta, spec invv1alpha1.EndpointSpec) (*fn.KubeObject, error) {
	claim := invv1alpha1.BuildEndpoint(
		meta,
		spec,
		invv1alpha1.EndpointStatus{},
	)

	return fn.NewFromTypedObject(claim)
}

func getLinkNodeId(labels map[string]string) (int, error) {
	v, ok := labels["nephio.org/nodeId"]
	if !ok {
		return 0, fmt.Errorf("expecting label key %s and values 0 or 1", "nephio.org/nodeId")
	}
	id, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	switch id {
	case 0:
		return 0, nil
	case 1:
		return 1, nil
	default:
		return 0, fmt.Errorf("expecting label key %s and values 0 or 1, got %d", "nephio.org/nodeId", id)
	}
}

func getLinkId(labels map[string]string) (int, error) {
	v, ok := labels["nephio.org/linkId"]
	if !ok {
		return 0, fmt.Errorf("expecting label key %s and values 0 or 1", "nephio.org/linkId")
	}
	id, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	if id > 2 {
		return 0, fmt.Errorf("current fn supports max 2 links per cluster node")
	}
	if id == 0 {
		return 0, fmt.Errorf("we count from 1, not from 0")
	}
	return id, nil
}

const edge = "edge"
const region = "regional"

// This is specific for a lab environment. In reality we dont need this
// but in a lab we connect multiple clusters to the same device
func getClusterOffset(clusterName string) (int, error) {
	switch {
	case strings.HasPrefix(clusterName, edge):
		offset, err := getOffset(clusterName, edge)
		if err != nil {
			return 0, err
		}
		return ((offset - 1) * 4), nil
	case strings.HasPrefix(clusterName, region):
		offset, err := getOffset(clusterName, region)
		if err != nil {
			return 0, err
		}
		return ((offset - 1) * 4) + 16, nil
	default:
		return 0, fmt.Errorf("configuration not supported, got %s, supported %s or %s", clusterName, edge, region)
	}
}

func getOffset(clusterName, prefix string) (int, error) {
	nbr := strings.TrimPrefix(clusterName, prefix)
	if len(nbr) == 0 {
		return 1, nil
	}
	i, err := strconv.Atoi(nbr)
	if err != nil {
		return 0, err
	}
	if i > 4 {
		return 0, fmt.Errorf("configuration not supported, max 4 nodes of type edge support, got %d", i)
	}
	if i == 0 {
		return 0, fmt.Errorf("configuration not supported, we count from 1, got 0")
	}
	return i, nil
}
