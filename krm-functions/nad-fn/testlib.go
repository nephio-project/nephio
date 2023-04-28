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

package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn/testhelpers"
	"sigs.k8s.io/yaml"
)

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
		content := testhelpers.MustReadFile(t, filepath.Join(dir, f.Name()))
		fileItems, err := fn.ParseKubeObjects(content)
		if err != nil {
			t.Fatalf("failed to parse objects from file %q: %v", filepath.Join(dir, f.Name()), err)
		}
		items = append(items, fileItems...)
	}

	var functionConfig *fn.KubeObject = fn.NewEmptyKubeObject()
	// config := mustParseFile(t, filepath.Join(dir, "_fnconfig.yaml"))
	// if len(config) == 0 {
	// 	functionConfig = fn.NewEmptyKubeObject()
	// } else if len(config) == 1 {
	// 	functionConfig = config[0]
	// } else {
	// 	t.Fatalf("found multiple config objects in %s", filepath.Join(dir, "_fnconfig.yaml"))
	// }
	return &fn.ResourceList{Items: items, FunctionConfig: functionConfig}
}

func CheckRunError(t *testing.T, expectedErrorFile string, actualError error) {
	content, err := os.ReadFile(expectedErrorFile)
	if err != nil {
		if os.IsNotExist(err) {
			// if no expected errors are set: handle KRM function errors normally
			if actualError != nil {
				t.Errorf("the KRM function failed unexpectedly: %v", actualError)
			}
			return
		} else {
			t.Fatalf("couldn't read file %q: %v", expectedErrorFile, err)
		}
	}

	expectedError := strings.TrimSpace(string(content))
	if actualError == nil {
		t.Errorf("the KRM function hasn't returned any errors, but it should have failed with this error message: %v", expectedError)
	}
	if !strings.Contains(actualError.Error(), expectedError) {
		t.Errorf("the KRM function returned with the wrong error message.\n    expected error: %v\n    actual error: %v", expectedError, actualError)
	}
}

func CheckResults(t *testing.T, expectedResultsFile string, rl *fn.ResourceList) {
	resultsBytes, err := os.ReadFile(expectedResultsFile)
	if err != nil {
		if os.IsNotExist(err) {
			// if no expected results are set: then pass the test
			return
		} else {
			t.Fatalf("couldn't read file %q: %v", expectedResultsFile, err)
		}
	}
	var expectedResults fn.Results
	err = yaml.Unmarshal(resultsBytes, &expectedResults)
	if err != nil {
		t.Fatalf("coudln't parse expected results list: %v", err)
	}
	found := false
	for _, er := range expectedResults {
		for _, ar := range rl.Results {
			if er.Severity == ar.Severity && er.Message == ar.Message {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing %q result with this message: %v", er.Severity, er.Message)
		}
	}
}

func RunFailureCases(t *testing.T, basedir string, krmFunction fn.ResourceListProcessor) {
	dirEntries, err := os.ReadDir(basedir)
	if err != nil {
		t.Fatalf("ReadDir(%q) failed: %v", basedir, err)
	}

	for _, dirEntry := range dirEntries {
		dir := filepath.Join(basedir, dirEntry.Name())
		if !dirEntry.IsDir() {
			t.Fatalf("expected directory, found %s", dir)
			continue
		}

		t.Run(dir, func(t *testing.T) {
			rl := ParseResourceListFromDir(t, dir)
			_, processError := krmFunction.Process(rl)

			p := filepath.Join(dir, "_expected_error.txt")
			CheckRunError(t, p, processError)

			p = filepath.Join(dir, "_expected_result.yaml")
			CheckResults(t, p, rl)
		})
	}
}
