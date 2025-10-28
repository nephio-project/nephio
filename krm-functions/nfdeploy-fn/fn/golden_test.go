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

package fn

import (
	"testing"

	"github.com/kptdev/krm-functions-sdk/go/fn"
	tst "github.com/nephio-project/nephio/krm-functions/lib/test"
)

const TestGoldenPath = "testdata/golden"
const TestFailurePath = "testdata/failure"

func TestGolden(t *testing.T) {
	fnRunner := fn.ResourceListProcessorFunc(Run)

	//// This golden test expects each sub-directory of `testdata` can has its input resources (in `resources.yaml`)
	//// be modified to the output resources (in `_expected_error.txt`).
	tst.RunGoldenTests(t, TestGoldenPath, fnRunner)
}

func TestFailureCases(t *testing.T) {
	fnRunner := fn.ResourceListProcessorFunc(Run)

	tst.RunGoldenTests(t, TestFailurePath, fnRunner)
}
