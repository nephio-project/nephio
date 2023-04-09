package mutator

import (
	"strings"

	"sigs.k8s.io/kustomize/kyaml/utils"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func GetIpamValue(source *kyaml.RNode, fp string) string {
	fieldPath := utils.SmarterPathSplitter(fp, ".")
	foundValue, lookupErr := source.Pipe(&kyaml.PathGetter{Path: fieldPath})
	if lookupErr != nil {
		return ""
	}
	return strings.TrimSuffix(foundValue.MustString(), "\n")
}

func GetPrefixKind(source *kyaml.RNode) string {
	return GetIpamValue(source, "spec.kind")
}

func GetIpamName(source *kyaml.RNode) string {
	return GetIpamValue(source, "metadata.name")
}

func GetIpamInterfaceName(source *kyaml.RNode) string {
	return source.GetLabels()["nephio.org/interface"]
}

func GetGateway(source *kyaml.RNode) string {
	return GetIpamValue(source, "status.gateway")
}

func GetPrefix(source *kyaml.RNode) string {
	return GetIpamValue(source, "status.prefix")
}
