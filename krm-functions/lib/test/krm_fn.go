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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn/testhelpers"
	"sigs.k8s.io/yaml"
)

// NOTE: functions in this file are candidates to be eventually merged to
// github.com/GoogleContainerTools/kpt-functions-sdk/go/fn/testhelpers

// RunGoldenTests provides the exact same functionality as its upstream counterpart: testhelpers.RunGoldenTests
// with the only exception that after running a testcase it creates a file named _actual_output.yaml in its subdirectory
// containing the actual output of the KRM function. This file can be the basis of creating _expected.yaml later.
//
// Here is the documentation comment of the original testhelpers.RunGoldenTests:
// RunGoldenTests provides the test infra to run golden test.
// "basedir" should be the parent directory, under where the sub-directories contains test data.
// For example, the "testdata" is the basedir. It contains two cases "test1" and "test2"
//
//	└── testdata
//	    └── test1
//	        ├── _expected.yaml
//	        ├── _fnconfig.yaml
//	        └── resources.yaml
//	    └── test2
//	        ├── _expected.yaml
//	        ├── _fnconfig.yaml
//	        └── resources.yaml
//
// - "krmFunction" should be your ResourceListProcessor implementation.
func RunGoldenTests(t *testing.T, basedir string, krmFunction fn.ResourceListProcessor) {
	dirEntries, err := os.ReadDir(basedir)
	if err != nil {
		t.Fatalf("ReadDir(%q) failed: %v", basedir, err)
	}

	for _, dirEntry := range dirEntries {
		dir := filepath.Join(basedir, dirEntry.Name())
		if !dirEntry.IsDir() {
			t.Errorf("expected directory, found %s", dir)
			continue
		}

		t.Run(dir, func(t *testing.T) {
			rl := ParseResourceListFromDir(t, dir)
			_, processErr := krmFunction.Process(rl)

			rlYAML, err := rl.ToYAML()
			if err != nil {
				t.Fatalf("failed to convert resource list to yaml: %v", err)
			}
			_ = os.WriteFile(filepath.Join(dir, "_actual_output.yaml"), rlYAML, 0666)

			CheckRunError(t, dir, processErr)
			CheckResults(t, dir, rl)
			p := filepath.Join(dir, "_expected.yaml")
			testhelpers.CompareGoldenFile(t, p, rlYAML)
		})
	}
}

// RunFailureCases provides the test infra to run test similar to testhelpers.RunGoldenTests,
// but instead of checking the whole output of the KRM function, RunFailureCases is able to
// check:
//   - if the main method of the KRM function (`Run`) returned with an error
//   - if the `Results` field of the output contained the expected Results
//
// - "basedir" should be the parent directory, under where the sub-directories contains test data.
// For example, the "testdata" is the basedir. It contains two cases "test1" and "test2"
//
//	└── testdata
//	    └── test1
//	        ├── _expected_error.txt
//	        ├── _fnconfig.yaml
//	        └── resources.yaml
//	    └── test2
//	        ├── _expected_results.yaml
//	        ├── _fnconfig.yaml
//	        └── resources.yaml
//
// if `_expected_error.txt` is present in the testdir, then the KRM function is expected to return with
// an error whose message contains the string in the file
// if `_expected_results.yaml` is present in the testdir: it should contain a list of fn.Result objects (serialized in YAML),
// that are checked against the actual Results list of the KRM functions output
//
// - "krmFunction" should be your ResourceListProcessor implementation.
func RunFailureCases(t *testing.T, basedir string, krmFunction fn.ResourceListProcessor) {
	dirEntries, err := os.ReadDir(basedir)
	if err != nil {
		t.Fatalf("ReadDir(%q) failed: %v", basedir, err)
	}

	for _, dirEntry := range dirEntries {
		dir := filepath.Join(basedir, dirEntry.Name())
		if !dirEntry.IsDir() {
			t.Errorf("expected directory, found %s", dir)
			continue
		}

		t.Run(dir, func(t *testing.T) {
			rl := ParseResourceListFromDir(t, dir)
			_, processError := krmFunction.Process(rl)

			CheckRunError(t, dir, processError)
			CheckResults(t, dir, rl)
		})
	}
}

func CheckRunError(t *testing.T, dir string, actualError error) {
	content, err := os.ReadFile(filepath.Clean(filepath.Join(dir, "_expected_error.txt")))
	if err != nil {
		if os.IsNotExist(err) {
			// if no expected errors are set: handle KRM function errors normally
			if actualError != nil {
				t.Errorf("the KRM function failed unexpectedly: %v", actualError)
			}
			return
		} else {
			t.Fatalf("couldn't read file %v/_expected_error.txt: %v", dir, err)
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

func CheckResults(t *testing.T, dir string, rl *fn.ResourceList) {
	resultsBytes, err := os.ReadFile(filepath.Clean(filepath.Join(dir, "_expected_results.yaml")))
	if err != nil {
		if os.IsNotExist(err) {
			// if no expected results are set: then pass the test
			return
		} else {
			t.Fatalf("couldn't read file %v/_expected_results.yaml: %v", dir, err)
		}
	}
	var expectedResults fn.Results
	err = yaml.Unmarshal(resultsBytes, &expectedResults)
	if err != nil {
		t.Fatalf("couldn't parse expected results list: %v", err)
	}
	for _, er := range expectedResults {
		found := false
		for _, ar := range rl.Results {
			if er.Severity == ar.Severity && strings.Contains(ar.Message, er.Message) {
				found = true
				break
			}

		}
		if !found {
			t.Errorf("missing result from the output of the function:\n  wanted: %v with msg: %v\n  got:\n%v", er.Severity, er.Message, rl.Results)
		}
	}
}
