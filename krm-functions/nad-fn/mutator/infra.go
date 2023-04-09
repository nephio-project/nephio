/*
Copyright 2022 Nokia.

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
	"strings"

	"sigs.k8s.io/kustomize/kyaml/utils"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

//nadCniType = clusterContext.Spec.CNIConfig.CNIType
//nadMasterInterface = clusterContext.Spec.CNIConfig.MasterInterface

func GetInfraValue(source *kyaml.RNode, fp string) string {
	fieldPath := utils.SmarterPathSplitter(fp, ".")
	foundValue, lookupErr := source.Pipe(&kyaml.PathGetter{Path: fieldPath})
	if lookupErr != nil {
		return ""
	}
	return strings.TrimSuffix(foundValue.MustString(), "\n")
}

func GetCniType(source *kyaml.RNode) string {
	return GetInfraValue(source, "spec.cniConfig.cniType")
}

func GetMasterInterface(source *kyaml.RNode) string {
	return GetInfraValue(source, "spec.cniConfig.masterInterface")
}

func GetEndPoints(source *kyaml.RNode) map[string]string {
	fps := map[string]string{
		"n3": "spec.n3endpoints",
		"n4": "spec.n4endpoints",
		"n6": "spec.n6endpoints",
		"n9": "spec.n9endpoints",
	}
	endpoints := map[string]string{}
	for epName, fp := range fps {
		if GetInfraValue(source, fp) != "0" {
			endpoints[epName] = epName
		}
	}
	return endpoints
}
