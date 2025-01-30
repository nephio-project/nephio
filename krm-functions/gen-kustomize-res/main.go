package main

import (
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

const (
	KustomizeGroup = "kustomize.config.k8s.io"
	KustomizeKind  = "Kustomization"
	KustomizeName  = "kustomization"
)

func main() {
	if err := fn.AsMain(fn.ResourceListProcessorFunc(Run)); err != nil {
		os.Exit(1)
	}
}

// Run is the entry point of the KRM function
func Run(rl *fn.ResourceList) (success bool, err error) {

	var targetKustObj fn.KubeObject
	var targetKustResList []string

	existingKustObj := getExistingKustomization(rl)
	if existingKustObj != nil {
		// merge the inputs to the kustomization resources list
		existingKustResources, _, _ := existingKustObj.SubObject.NestedStringSlice("resources")
		if len(existingKustResources) != 0 {
			targetKustResList = mergeAndRemoveDuplicates(existingKustResources, getNonLocalConfig(rl))
		} else {
			targetKustResList = getNonLocalConfig(rl)
		}
		targetKustObj = *existingKustObj
	} else {
		// insert new kustomization
		newKustObj := getNewKustObject()
		// set a default Path annotation
		err = newKustObj.SetAnnotation(kioutil.PathAnnotation, KustomizeName+".yaml")
		if err != nil {
			rl.LogResult(err)
			return false, nil
		}
		// set the target kustomization data
		targetKustResList = getNonLocalConfig(rl)
		targetKustObj = *newKustObj
	}
	// update the resources list
	err = targetKustObj.SubObject.SetNestedStringSlice(targetKustResList, "resources")
	if err != nil {
		rl.LogResult(err)
		return false, nil
	}
	err = rl.UpsertObjectToItems(targetKustObj, nil, true)
	if err != nil {
		rl.LogResult(err)
		return false, nil
	}

	return true, nil
}

func getNewKustObject() *fn.KubeObject {
	kustTemplate := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: upsert-kustomize-res
resources: []`
	newKustObj, _ := fn.ParseKubeObject([]byte(kustTemplate))
	return newKustObj
}

func getExistingKustomization(rl *fn.ResourceList) *fn.KubeObject {
	for _, item := range rl.Items {
		if item.IsGroupKind(schema.GroupKind{Group: KustomizeGroup, Kind: KustomizeKind}) {
			return item
		}
	}
	return nil
}

func getNonLocalConfig(rl *fn.ResourceList) []string {
	var resources []string
	for _, item := range rl.Items {
		if !item.IsLocalConfig() && !item.IsGroupKind(schema.GroupKind{Group: KustomizeGroup, Kind: KustomizeKind}) {
			resources = append(resources, item.PathAnnotation())
		}
	}
	return resources
}

func mergeAndRemoveDuplicates(slices ...[]string) []string {
	uniqueMap := make(map[string]bool)
	var result []string
	// Iterate through all slices
	for _, slice := range slices {
		for _, item := range slice {
			// If item is not in map, add it
			if !uniqueMap[item] {
				uniqueMap[item] = true
				result = append(result, item)
			}
		}
	}
	return result
}
