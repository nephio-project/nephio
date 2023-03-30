/*
Copyright 2023 Samsung.

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


package diff

import (
    "bytes"
    "fmt"
    v1 "github.com/GoogleContainerTools/kpt-functions-sdk/go/api/kptfile/v1"
    "github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
    kptv1 "github.com/GoogleContainerTools/kpt"
    corev1 "k8s.io/api/core/v1"
)

type Inventory interface {
    AddExistingConditions(*corev1.ObjectReference, *kptv1.Condition)
    AddExistingObj(*corev1.ObjectReference, *fn.KubeObject)
    AddNewObj(*corev1.ObjectReference, *fn.KubeObject)
    Diff(rl *fn.ResourceList) InventoryDiff
}

type InventoryDiff struct {
    DeleteObj []*corev1.ObjectReference
    UpdateObj []*corev1.ObjectReference
    CreateObj []*corev1.ObjectReference
}

/*
Should the error be added to fn.ResourceList.Results?
can we take currentRL and Items KubeObjects as input parameter and then manipulate?
In current implementation we can just return newRL *fn.ResourceList as it will be the final RL
*/

//GVK - Group, Version, Kind
// Diff takes in only filtered GVK list of new and old ResourceList  types
func Diff(currentRL *fn.ResourceList, newRL *fn.ResourceList) (*fn.ResourceList, error) {
    currentRLYaml, err := currentRL.ToYAML()
    if err != nil {
        return nil, fmt.Errorf("unable to convert current recource list to yaml %v", err)
    }

    newRLYaml, err := newRL.ToYAML()
    if err != nil {
        return nil, fmt.Errorf("unable to convert new recource list to yaml %v", err)
    }

    if bytes.Equal(currentRLYaml, newRLYaml) {
        return currentRL, nil
    }
    //^ Both recource List are exactly similar

    //Create a map of names and index in Items[]
    currentItems := make(map[string]int)
    for index, items := range currentRL.Items {
        currentItems[items.GetName()] = index
    }

    newItems := make(map[string]int)
    for index, items := range newRL.Items {
        newItems[items.GetName()] = index
    }

    // add index we will take from new Resource List
    addItems := make(map[string]int)
    // ex currentItems = [[a, 1], [b, 0], [c, 2]] newItems = [[b, 1], [c, 0], [d, 2]] and nothing changed in b, c Items then
    // addItems = [[d, 2]]

    // delete index we will take from current Resource List
    // ex currentItems = [[a, 1], [b, 0], [c, 2]] newItems = [[b, 1], [c, 0], [d, 2]] and nothing changed in b, c Items then
    // deleteItems = [[a, 1]]
    deleteItems := make(map[string]int)

    // ex currentItems = [[a, 1], [b, 0], [c, 2]] newItems = [[b, 1], [c, 0], [d, 2]] and if something changed in c Items then
    // addItems = [[d, 2], [c, 0]]
    // deleteItems = [[a, 1], [c, 2]]

    //updateItems := make(map[string]int)

    for name, index := range newItems {
        value, isPresent := currentItems[name]
        if isPresent {
            // We are converting into yaml and compare Items
            currentItemYamlString := currentRL.Items[value].String()
            newItemYamlString := newRL.Items[index].String()
            if currentItemYamlString != newItemYamlString {
                deleteItems[name] = value
                addItems[name] = index
                //updateItems[name] = index
            }
        }else {
            addItems[name] = index
        }
    }

    for name, index := range currentItems {
        _, isPresent := newItems[name]
        if !isPresent {
            deleteItems[name] = index
        }
    }

    //Iterate through delete

    for _, index := range deleteItems {
        currentRL.Items[index].
    }

    // //Iterate through add

    return newRL, nil

}

func removeResourceList(resources fn.KubeObjects, resourceIndex int) fn.KubeObjects {
    //Deleted the respective Condition from KPTFile
    got := v1.GetConditionType(&corev1.ObjectReference{
        APIVersion: "tt.input.apiVersion",
        Kind:       "tt.input.kind",
        Name:       "tt.input.name",
    })
    if got != "" {
        var kpt v1.KptFile
        v1.KptFile.DeleteCondition(kpt, got)
    }
    return append(resources[:resourceIndex], resources[resourceIndex+1:]...)
}

func addResourceList(resources fn.KubeObjects, e *fn.KubeObject) fn.KubeObjects {
    //Add the respective Condition to KPTFile
    got := kptv1.Condition(&corev1.ObjectReference{
        Type: "b",
        Status: kptv1.ConditionFalse,
        Reason: "",
        Message: ""
    })
    if got != "" {
        var kpt v1.KptFile
        v1.KptFile.SetConditions(kpt, got)
    }
    return append(resources, e)
}

func updateResourceList(resources fn.KubeObjects, resourceIndex int, e *fn.KubeObject) fn.KubeObjects {
    //Update the respective Condition to KPTFile

    got := kptv1.Condition(&corev1.ObjectReference{
        Type: "b",
        Status: kptv1.ConditionFalse,
        Reason: "kpt condition was updated",
        Message: "change occured due to kpt function"
    })
    if got != "" {
        var kpt v1.KptFile
        v1.KptFile.SetConditions(kpt, got)
    }
    resources[resourceIndex] = e
    return resources
}
