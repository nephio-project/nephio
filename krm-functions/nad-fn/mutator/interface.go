package mutator

import (
	"strings"

	"sigs.k8s.io/kustomize/kyaml/utils"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func GetInterfaceValue(source *kyaml.RNode, fp string) string {
	fieldPath := utils.SmarterPathSplitter(fp, ".")
	foundValue, lookupErr := source.Pipe(&kyaml.PathGetter{Path: fieldPath})
	if lookupErr != nil {
		return ""
	}
	return strings.TrimSuffix(foundValue.MustString(), "\n")
}

func GetInterfaceName(source *kyaml.RNode) string {
	return GetInterfaceValue(source, "metadata.name")
}

func GetInterfaceCniType(source *kyaml.RNode) string {
	return GetInterfaceValue(source, "spec.cniType")
}
