/*
Copyright 2023 Nephio.

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
	"bytes"
	"fmt"
	"text/template"
	//"github.com/google/safetext/yamltemplate"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GenConfigMapEntry struct {
	Type  string
	Key   string
	Value string
}

type GenConfigMap struct {
	ConfigMapMetadata metav1.ObjectMeta
	Params            map[string]string
	Data              []GenConfigMapEntry
}

func (entry *GenConfigMapEntry) generate(params, data map[string]string) error {
	switch entry.Type {
	case "gotmpl":
		t, err := template.New(entry.Key).Parse(entry.Value)
		if err != nil {
			return err
		}
		buf := &bytes.Buffer{}
		err = t.Execute(buf, params)
		if err != nil {
			return err
		}
		data[entry.Key] = buf.String()
	default:
		data[entry.Key] = entry.Value
	}

	return nil
}

func (fc *GenConfigMap) Validate() error {
	for i, entry := range fc.Data {
		if entry.Key == "" {
			return fmt.Errorf("data entry %d, key must not be empty", i)
		}
	}
	return nil
}

func (processor *GenConfigMap) Process(rl *fn.ResourceList) (bool, error) {
	// read our fc into a new struct
	fc := &GenConfigMap{}
	err := rl.FunctionConfig.As(fc)
	if err != nil {
		return false, err
	}

	name := fc.ConfigMapMetadata.Name
	if name == "" {
		name = rl.FunctionConfig.GetName()
	}

	err = fc.Validate()
	if err != nil {
		return false, err
	}

	cmko := fn.NewEmptyKubeObject()
	cmko.SetAPIVersion("v1")
	cmko.SetKind("ConfigMap")
	err = cmko.SetNestedField(fc.ConfigMapMetadata, "metadata")
	if err != nil {
		return false, err
	}
	_, err = cmko.RemoveNestedField("metadata", "creationTimestamp")
	if err != nil {
		return false, err
	}
	cmko.SetName(name)

	data := make(map[string]string, len(fc.Data))
	for _, entry := range fc.Data {
		entry.generate(fc.Params, data)
	}
	if len(data) > 0 {
		err = cmko.SetNestedField(data, "data")
		if err != nil {
			return false, err
		}
	}
	err = rl.UpsertObjectToItems(cmko, nil, true)
	if err != nil {
		return false, err
	}

	return true, nil
}
