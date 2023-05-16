/*
Copyright 2022-2023 The Nephio Authors.

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

package util

import (
	"bytes"
	"errors"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	YamlObjectDelimiter = "---"
)

// Parses string representation of a yaml file into Yaml nodes. The string can have multiple
// resources and each of them would be parsed into a different yaml node.
func ParseStringToYamlNode(s string) ([]*yaml.RNode, error) {
	return (&kio.ByteReader{
		Reader:                bytes.NewBufferString(s),
		OmitReaderAnnotations: true,
	}).Read()
}

// filter yaml nodes that matches the criteria passed in the parameters.
// Atleast one of the criteria should be present.
func GetMatchingYamlNodes(nodes []*yaml.RNode, apiversion string, kind string, name string) ([]*yaml.RNode, error) {
	if apiversion == "" && kind == "" && name == "" {
		return nil, errors.New("Invalid input: every criteria is empty.")
	}

	s := framework.Selector{}
	if apiversion != "" {
		s.APIVersions = []string{apiversion}
	}
	if kind != "" {
		s.Kinds = []string{kind}
	}
	if name != "" {
		s.Names = []string{name}
	}

	f, err := s.Filter(nodes)
	if err != nil {
		return nil, err
	} else {
		return f, nil
	}
}
