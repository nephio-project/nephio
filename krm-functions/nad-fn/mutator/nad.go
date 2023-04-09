package mutator

import (
	"reflect"
	"strings"

	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/kyaml/utils"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

type Config struct {
	CniVersion string
	CniType    string
	Master     string
	IPPrefix   string
	Gateway    string
}

type NadCOnfig struct {
	CniVersion string          `json:"cniVersion"`
	Plugins    []PluginCniType `json:"plugins"`
}

type PluginCniType struct {
	Type         string       `json:"type"`
	Capabilities Capabilities `json:"capabilities"`
	Master       string       `json:"master"`
	Mode         string       `json:"mode"`
	Ipam         Ipam         `json:"ipam"`
}

type Capabilities struct {
	Ips bool `json:"ips"`
	Mac bool `json:"mac"`
}

type Ipam struct {
	Type      string      `json:"type"`
	Addresses []Addresses `json:"addresses"`
}

type Addresses struct {
	Address string `json:"address"`
	Gateway string `json:"gateway"`
}

func GetNadValue(source *kyaml.RNode, fp string) string {
	fieldPath := utils.SmarterPathSplitter(fp, ".")
	foundValue, lookupErr := source.Pipe(&kyaml.PathGetter{Path: fieldPath})
	if lookupErr != nil {
		return ""
	}
	return strings.TrimSuffix(foundValue.MustString(), "\n")
}

func GetNadPrefixKind(source *kyaml.RNode) string {
	return GetNadValue(source, "spec.kind")
}

func GetNadName(source *kyaml.RNode) string {
	return GetNadValue(source, "metadata.name")
}

func GetNadNamespace(source *kyaml.RNode) string {
	return GetNadValue(source, "metadata.namespace")
}

func GetNadGV(source *kyaml.RNode) string {
	return GetNadValue(source, "apiVersion")
}

func GetNadGVKN(source *kyaml.RNode) string {
	return GetNadValue(source, "apiVersion") + "." +
		reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name() + "." + GetNadValue(source, "metadata.name")
}

func GetObjectReference(source *kyaml.RNode) *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:       reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name(),
		APIVersion: GetNadValue(source, "apiVersion"),
		Name:       GetNadValue(source, "metadata.name"),
		Namespace:  GetNadValue(source, "metadata.namespace"),
	}
}

func GetNadRnode(c *Config) (NadCOnfig, error) {
	result := NadCOnfig{
		CniVersion: c.CniVersion,
		Plugins: []PluginCniType{
			{
				Type: c.CniType,
				Capabilities: Capabilities{
					Ips: true,
				},
				Master: c.Master,
				Mode:   "bridge",
				Ipam: Ipam{
					Type: "static",
					Addresses: []Addresses{
						{
							Address: c.IPPrefix,
							Gateway: c.Gateway,
						},
					},
				},
			},
			{
				Type: c.CniType,
				Capabilities: Capabilities{
					Mac: true,
				},
			},
		},
	}

	return result, nil

	/*
			var nadTemplate = `'{"cniVersion": "{{.CniVersion}}",
				"plugins": [
					{
						"type": "{{.CniType}}",
						"capabilities": { "ips": true },
						"master": "{{.Master}}",
						"mode": "bridge",
						"ipam": {
							"type": "static",
							"addresses": [
								{
									"address": "{{.IPPrefix}}",
									"gateway": "{{.Gateway}}"
								}
							]
						}
					},
					{
						"capabilities": { "mac": true },
						"type": "tuning"
					}
				]
			}'
		`
			tmpl, err := template.New("nad").Parse(nadTemplate)
			if err != nil {
				return result, err
			}
			var buf bytes.Buffer
			err = tmpl.Execute(&buf, map[string]interface{}{
				"CniVersion": c.CniVersion,
				"CniType":    c.CniType,
				"Master":     c.Master,
				"IPPrefix":   c.IPPrefix,
				"Gateway":    c.Gateway,
			})
			if err != nil {
				return result, err
			}
	*/
}
