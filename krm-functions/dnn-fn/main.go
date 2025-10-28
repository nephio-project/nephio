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

	"github.com/kptdev/krm-functions-sdk/go/fn"
	dnn_fn "github.com/nephio-project/nephio/krm-functions/dnn-fn/fn"
)

func main() {
	runner := fn.ResourceListProcessorFunc(dnn_fn.Run)

	if err := fn.AsMain(runner); err != nil {
		os.Exit(1)
	}
}
