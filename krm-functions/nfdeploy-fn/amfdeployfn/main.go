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
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	nephiodeployv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/nfdeploy-fn/common"
	"os"
)

func Run(rl *fn.ResourceList) (bool, error) {
	return common.Run[nephiodeployv1alpha1.AMFDeployment](rl, nephiodeployv1alpha1.AMFDeploymenteGroupVersionKind)
}

func main() {
	runner := fn.ResourceListProcessorFunc(Run)

	if err := fn.AsMain(runner); err != nil {
		os.Exit(1)
	}
}
