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
package test

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn/testhelpers"
)

// MustParseKubeObjects reads a list of KubeObjects from the given YAML file or fails the test
func MustParseKubeObjects(t *testing.T, path string) fn.KubeObjects {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	b := testhelpers.MustReadFile(t, path)

	objects, err := fn.ParseKubeObjects(b)
	if err != nil {
		t.Fatalf("failed to parse objects from file %q: %v", path, err)
	}
	return objects
}

// MustParseKubeObject reads one KubeObject from the given YAML file or fails the test
func MustParseKubeObject(t *testing.T, path string) *fn.KubeObject {
	b := testhelpers.MustReadFile(t, path)
	object, err := fn.ParseKubeObject(b)
	if err != nil {
		t.Fatalf("failed to parse object from file %q: %v", path, err)
	}
	return object
}

// ParseResourceListFromDir parses all YAML files from the given `dir`,
// and gives back the parsed objects in a ResourceList, or fails the test
func ParseResourceListFromDir(t *testing.T, dir string) *fn.ResourceList {
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory %q: %v", dir, err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	var items fn.KubeObjects
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "_") {
			continue
		}
		// Users can put other types of files to the test dir, but they won't be read.
		// A default kpt package contains README file.
		if !testhelpers.IsValidYAMLOrKptfile(f.Name()) {
			continue
		}
		fileItems := MustParseKubeObjects(t, filepath.Join(dir, f.Name()))
		items = append(items, fileItems...)
	}

	var functionConfig *fn.KubeObject
	config := MustParseKubeObjects(t, filepath.Join(dir, "_fnconfig.yaml"))
	if len(config) == 0 {
		functionConfig = fn.NewEmptyKubeObject()
	} else if len(config) == 1 {
		functionConfig = config[0]
	} else {
		t.Fatalf("found multiple config objects in %s", filepath.Join(dir, "_fnconfig.yaml"))
	}
	return &fn.ResourceList{Items: items, FunctionConfig: functionConfig}
}
