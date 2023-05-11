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

// RunGoldenTests provides the functionality of its upstream counterpart: testhelpers.RunGoldenTests, but with
// some extra functionality (i.e. _expected_error.txt, _expected_results.yaml, _actual_output.yaml).
//
// RunGoldenTests provides the test infra to run golden test.
// "basedir" should be the parent directory, under where each sub-directory contains data for a test case.
// "krmFunction" should be your ResourceListProcessor implementation that is to be tested.
//
// For example, if "testdata" is the basedir, it contains two cases "test1" and "test2":
//
//	└── testdata
//	    └── test1
//	        ├── _expected.yaml
//	        ├── _fnconfig.yaml
//	        └── resources.yaml
//	    └── test2
//	        ├── _expected_error.txt
//	        ├── _expected_results.yaml
//	        ├── Kptfile
//	        ├── service.yaml
//	        └── deployment.yaml
//
// The files in a test case's subdirectory are interpreted as follows:
//   - YAML files whose name doesn't start with an underscore (including the Kptfile) are
//     interpreted as parts of the input kpt package. They should contain YAML serialized KRM resources.
//   - _fnconfig.yaml, if present, holds the configuration parameters of the KRM functions
//   - _expected.yaml holds the expected output of the KRM function. If present, then the YAML output of the
//     KRM function is compared to it character-by-character
//   - _expected_results.yaml allows to test if certain results (typically errors) are present in the output's
//     `Results` list without comparing the rest of the output. If present, it should contain a list of fn.Result
//     objects, and each of them should be present in the output.
//   - _expected_error.txt allows testing if the KRM function's implementation (the Run function) returned with an error.
//     If present, then the KRM function is expected to return with an error whose message contains the string in the file.
//     If this file is not present in a test case's subdirectory, and the KRM function returns with an error, then the test
//     is considered failed.
//
// After running a testcase RunGoldenTests creates a file named _actual_output.yaml in its subdirectory,
// containing the actual output of the KRM function. This file can be used to compare with _expected.yaml by an external diff (GUI) tool.
//
// If the `WRITE_GOLDEN_OUTPUT` environment variable is set with a non-empty value, then the _expected.yaml file is overwritten with
// actual output of the KRM function.
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

			CheckRunError(t, dir, processErr)
			CheckResults(t, dir, rl)
			CheckExpectedOutput(t, dir, rl)
		})
	}
}

// RunFailureCases is an alias of RunGoldenTests.
// This is kept here only temporarily for backward compatiblity reasons.
func RunFailureCases(t *testing.T, basedir string, krmFunction fn.ResourceListProcessor) {
	RunGoldenTests(t, basedir, krmFunction)
}

// RunGoldenTestForPipeline tests a sequence (pipeline) of KRM functions that are applied to a kpt package one after the other,
// and the final output is tested against some expected output.
// RunGoldenTestForPipeline behaves similar to RunGoldenTests, but it runs only one testcase whose data is in `dir`.
// The files in `dir` are interpreted the same as by RunGoldenTests. (See details in the documentation of RunGoldenTests)
func RunGoldenTestForPipeline(t *testing.T, dir string, krmFunctions []fn.ResourceListProcessor) {
	var err error
	rl := ParseResourceListFromDir(t, dir)

	for _, krm_fn := range krmFunctions {
		_, err = krm_fn.Process(rl)
		if err != nil {
			CheckRunError(t, dir, err)
			break
		}
	}

	CheckResults(t, dir, rl)
	CheckExpectedOutput(t, dir, rl)
}

// RunGoldenTestForPipelineOfFuncs calls RunGoldenTestForPipeline after converting `krmFunctions` to the expected format
func RunGoldenTestForPipelineOfFuncs(t *testing.T, dir string, krmFunctions []fn.ResourceListProcessorFunc) {
	processors := make([]fn.ResourceListProcessor, 0, len(krmFunctions))
	for _, f := range krmFunctions {
		processors = append(processors, f)
	}
	RunGoldenTestForPipeline(t, dir, processors)
}

func CheckRunError(t *testing.T, dir string, actualError error) {
	content, err := os.ReadFile(filepath.Clean(filepath.Join(dir, "_expected_error.txt")))
	if err != nil {
		if os.IsNotExist(err) {
			// if no expected errors are set: handle KRM function errors normally
			if actualError != nil {
				t.Fatalf("the KRM function failed unexpectedly: %v", actualError)
			}
			return
		} else {
			t.Fatalf("couldn't read file %v/_expected_error.txt: %v", dir, err)
		}
	}

	expectedError := strings.TrimSpace(string(content))
	if actualError == nil {
		t.Errorf("the KRM function hasn't returned any errors, but it should have failed with this error message: %v", expectedError)
		return
	}
	if !strings.Contains(actualError.Error(), expectedError) {
		t.Errorf("the KRM function returned with the wrong error message.\n    expected error: %v\n    actual error: %v", expectedError, actualError)
		return
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

func CheckExpectedOutput(t *testing.T, dir string, rl *fn.ResourceList) {
	p := filepath.Clean(filepath.Join(dir, "_expected.yaml"))
	_, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			// skip test if _expected.yaml is missing
			return
		}
		t.Fatalf("failed to access %v/_expected.yaml", dir)
	}

	rlYAML, err := rl.ToYAML()
	if err != nil {
		t.Fatalf("failed to convert resource list to yaml: %v", err)
	}
	_ = os.WriteFile(filepath.Join(dir, "_actual_output.yaml"), rlYAML, 0666)
	testhelpers.CompareGoldenFile(t, p, rlYAML)
}
