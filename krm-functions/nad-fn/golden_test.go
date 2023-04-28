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
	"github.com/nephio-project/nephio/krm-functions/nad-fn/mutator"
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn/testhelpers"
)

const GoldenTestDataPath = "testdata/golden"
const FailureCaseDataPath = "testdata/failure_cases"

func TestFunction(t *testing.T) {
	//fnRunner := fn.WithContext(context.TODO(), &DnnFn{})
	fnRunner := fn.ResourceListProcessorFunc(mutator.Run)

	// This golden test expects each sub-directory of `testdata` can has its input resources (in `resources.yaml`)
	// be modified to the output resources (in `_expected_error.txt`).
	testhelpers.RunGoldenTests(t, GoldenTestDataPath, fnRunner)

	RunFailureCases(t, FailureCaseDataPath, fnRunner)
}
