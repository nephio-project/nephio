/*
Copyright 2025 The Nephio Authors.

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
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn/testhelpers"
)

const TestDataPath = "testdata"

func TestFunction(t *testing.T) {
	// This golden test expects each sub-directory of `testdata` has its input resources (in `resources.yaml`)
	// be modified to the output resources (in `_expected.yaml`).
	testhelpers.RunGoldenTests(t, TestDataPath, fn.ResourceListProcessorFunc(Run))
}
