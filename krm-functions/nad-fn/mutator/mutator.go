/*
Copyright 2022 Samsung.

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

package mutator

import (
	//v1 "github/henderiw-nephio/nad-inject-fn/kptfile/v1"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	infrav1alpha1 "github.com/nephio-project/nephio-controller-poc/apis/infra/v1alpha1"
	difflib "github.com/nephio-project/nephio/krm-functions/nad-fn/lib/diff"
	kptfilev1 "github.com/nephio-project/nephio/krm-functions/nad-fn/lib/kptfile/v1"
	nadlib "github.com/nephio-project/nephio/krm-functions/nad-fn/lib/nad/v1"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/ipam/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	defaultCniVersion = "0.3.1"
)

// SetNad contains the information to perform the mutator function on a package
type SetNad struct {
	endPoints        map[string]*endPoint
	cniType          string
	masterInterface  string
	namespace        string
	interfaceCniType map[string]string
	existingNads     map[string]int // element to keep track of update
	kptFileLocation  int
	inventory        difflib.Inventory
	kptConditions    []kptv1.Condition
	kptfile          kptfilev1.KptFile
}

type endPoint struct {
	prefix  string
	gateway string
	nadName string
}

//Check all the cni-type
//Validate it with clusterCOntext
// the spec is just the same

func Run(rl *fn.ResourceList) (bool, error) {
	t := &SetNad{
		interfaceCniType: map[string]string{},
		endPoints:        map[string]*endPoint{},
		existingNads:     map[string]int{},
		inventory:        difflib.New(),
		kptConditions:    []kptv1.Condition{},
		kptfile:          kptfilev1.NewEmpty(),
	}
	// gathers the ip info from the ip-allocations
	t.GatherInfo(rl)

	//fmt.Printf("cniType: %s\n", t.cniType)
	//fmt.Printf("mastreInterface: %s\n", t.masterInterface)

	// transforms the upf with the ip info collected/gathered
	t.GenerateNad(rl)
	return true, nil
}

func (t *SetNad) GatherInfo(rl *fn.ResourceList) {
	for i, o := range rl.Items {
		// parse the node using kyaml
		rn, err := yaml.Parse(o.String())
		if err != nil {
			rl.Results = append(rl.Results, fn.ErrorConfigObjectResult(err, o))
		}
		if rn.GetApiVersion() == nephioreqv1alpha1.GroupVersion.Identifier() && rn.GetKind() == nephioreqv1alpha1.InterfaceKind {
			t.interfaceCniType[GetInterfaceName(rn)] = GetInterfaceCniType(rn)
		}
		if rn.GetApiVersion() == ipamv1alpha1.GroupVersion.Identifier() && rn.GetKind() == ipamv1alpha1.IPAllocationKind {
			if GetPrefixKind(rn) == string(ipamv1alpha1.PrefixKindNetwork) {
				t.endPoints[GetIpamInterfaceName(rn)] = &endPoint{
					prefix:  GetPrefix(rn),
					gateway: GetGateway(rn),
					nadName: GetIpamName(rn),
				}
			}
			t.namespace = rn.GetNamespace()
		}
		if rn.GetApiVersion() == infrav1alpha1.GroupVersion.Identifier() && rn.GetKind() == reflect.TypeOf(infrav1alpha1.ClusterContext{}).Name() {
			t.cniType = GetCniType(rn)
			t.masterInterface = GetMasterInterface(rn)
		}
		if rn.GetApiVersion() == kptv1.KptFileAPIVersion && rn.GetKind() == kptv1.TypeMeta.Kind {
			kf, err := kptfilev1.New(rn.MustString())
			if err != nil {
				fmt.Println("cannot unmarshal file:", err.Error())
			}
			t.kptConditions = kf.GetConditions()
			t.kptFileLocation = i
			t.kptfile = kf
		}
		if rn.GetApiVersion() == nadv1.SchemeGroupVersion.Identifier() && rn.GetKind() == reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name() {
			b, err := yaml.Marshal(rn.MustString())
			if err != nil {
				fmt.Println("cannot unmarshal file:", err.Error())
			}
			nadObject, _ := fn.ParseKubeObject(b)
			fmt.Println("to delete", GetObjectReference(rn), string(b))
			t.inventory.AddExistingResource(GetObjectReference(rn), nadObject)
			for _, kptCondition := range t.kptConditions {
				if kptCondition.Type == GetNadGVKN(rn) {
					t.inventory.AddExistingCondition(GetObjectReference(rn), &kptCondition)
				}
			}
		}
	}
}

func (t *SetNad) GenerateNad(rl *fn.ResourceList) {

	for epName, ep := range t.endPoints {
		if t.interfaceCniType[epName] == t.cniType {
			nadNode, err := GetNadRnode(&Config{
				CniVersion: defaultCniVersion,
				CniType:    t.cniType,
				Master:     t.masterInterface,
				IPPrefix:   ep.prefix,
				Gateway:    ep.gateway,
			})
			if err != nil {
				fmt.Println("PRINGINT ERROR", err)
			}
			b, err := json.Marshal(nadNode)
			if err != nil {
				fmt.Println("PRINGINT ERROR", err)
			}
			nadReceived := nadlib.NewGenerator(metav1.ObjectMeta{
				Name:      ep.nadName,
				Namespace: t.namespace,
			}, nadv1.NetworkAttachmentDefinitionSpec{
				Config: string(b),
			})
			nadKubeObject, err := nadReceived.ParseKubeObject()
			if err != nil {
				fmt.Println("PRINGINT ERROR", err)
			}

			t.inventory.AddNewResource(&corev1.ObjectReference{
				Kind:       nadKubeObject.GetKind(),
				APIVersion: nadKubeObject.GetAPIVersion(),
				Name:       nadKubeObject.GetName(),
				Namespace:  nadKubeObject.GetNamespace(),
			}, nadKubeObject)

			fmt.Println("to add", &corev1.ObjectReference{
				Kind:       nadKubeObject.GetKind(),
				APIVersion: nadKubeObject.GetAPIVersion(),
				Name:       nadKubeObject.GetName(),
				Namespace:  nadKubeObject.GetNamespace(),
			}, nadKubeObject)
		} else {
			fmt.Println("PRINGINT ERROR")
			fmt.Println("Interface CNIType doesn't match with ClusterContext", t.interfaceCniType[epName], t.cniType)
		}
	}
	inventoryDiff, err := t.inventory.Diff()
	if err != nil {
		fmt.Println("PRINGINT ERROR", err)
	}
	for _, inventoryNewItems := range inventoryDiff.CreateObjs {
		rl.Items = append(rl.Items, &inventoryNewItems.Obj)
		if err != nil {
			fmt.Println("cannot unmarshal file:", err.Error())
		}
	}
	for _, inventoryNewItems := range inventoryDiff.DeleteObjs {
		fmt.Println("ITEM to DELETE")
		for i, o := range rl.Items {
			if inventoryNewItems.Obj.GetAPIVersion() == o.GetAPIVersion() {
				if inventoryNewItems.Obj.GetKind() == o.GetKind() {
					if inventoryNewItems.Obj.GetName() == o.GetName() {
						fmt.Println("ITEM to DELETE INSIDE")
						rl.Items = append(rl.Items[:i], rl.Items[i+1:]...)
					}
				}
			}
		}
	}
	for _, inventoryNewCondition := range inventoryDiff.CreateConditions {
		t.kptfile.SetConditions(kptv1.Condition{
			Type:    inventoryNewCondition.Ref.APIVersion + "." + inventoryNewCondition.Ref.Kind + "." + inventoryNewCondition.Ref.Name,
			Status:  kptv1.ConditionTrue,
			Reason:  "New NAD condition is set",
			Message: "New NAD condition is set",
		})
		kptK8sObject, _ := t.kptfile.ParseKubeObject()
		rl.Items[t.kptFileLocation] = kptK8sObject
	}
	for _, inventoryNewCondition := range inventoryDiff.DeleteConditions {
		t.kptfile.DeleteCondition(inventoryNewCondition.Ref.APIVersion + "." + inventoryNewCondition.Ref.Kind + "." + inventoryNewCondition.Ref.Name)
		kptK8sObject, _ := t.kptfile.ParseKubeObject()
		rl.Items[t.kptFileLocation] = kptK8sObject
	}
	for _, inventoryNewCondition := range inventoryDiff.UpdateConditions {
		t.kptfile.SetConditions(kptv1.Condition{
			Type:    inventoryNewCondition.Ref.APIVersion + "." + inventoryNewCondition.Ref.Kind + "." + inventoryNewCondition.Ref.Name,
			Status:  kptv1.ConditionTrue,
			Reason:  "New NAD condition is updated",
			Message: "New NAD condition is updated",
		})
		kptK8sObject, _ := t.kptfile.ParseKubeObject()
		rl.Items[t.kptFileLocation] = kptK8sObject
	}
}
