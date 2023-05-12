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

package pipeline_tests

import (
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	tlib "github.com/nephio-project/nephio/krm-functions/lib/test"

	nephiodeployv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	dnn_fn "github.com/nephio-project/nephio/krm-functions/dnn-fn/fn"
	if_fn "github.com/nephio-project/nephio/krm-functions/interface-fn/fn"
	ipam_fn "github.com/nephio-project/nephio/krm-functions/ipam-fn/fn"
	nad_fn "github.com/nephio-project/nephio/krm-functions/nad-fn/mutator"
	nfdeploy_fn "github.com/nephio-project/nephio/krm-functions/nfdeploy-fn/common"
	vlan_fn "github.com/nephio-project/nephio/krm-functions/vlan-fn/fn"
	"github.com/nokia/k8s-ipam/pkg/proxy/clientproxy/ipam"
	"github.com/nokia/k8s-ipam/pkg/proxy/clientproxy/vlan"
)

const testdir = "testdata"

func UpfRun(rl *fn.ResourceList) (bool, error) {
	return nfdeploy_fn.Run[nephiodeployv1alpha1.UPFDeployment](rl, nephiodeployv1alpha1.UPFDeploymentGroupVersionKind)
}

func SmfRun(rl *fn.ResourceList) (bool, error) {
	return nfdeploy_fn.Run[nephiodeployv1alpha1.SMFDeployment](rl, nephiodeployv1alpha1.SMFDeploymentGroupVersionKind)
}

func AmfRun(rl *fn.ResourceList) (bool, error) {
	return nfdeploy_fn.Run[nephiodeployv1alpha1.AMFDeployment](rl, nephiodeployv1alpha1.AMFDeploymentGroupVersionKind)
}

func TestRelease1Scenario(t *testing.T) {
	ipamFn := &ipam_fn.FnR{
		ClientProxy: ipam.NewMock(),
	}
	vlanFn := &vlan_fn.FnR{
		ClientProxy: vlan.NewMock(),
	}

	testcaseDir := filepath.Join(testdir, "release1")
	var pipeline = []fn.ResourceListProcessorFunc{
		if_fn.Run,
		dnn_fn.Run,
		ipamFn.Run,
		vlanFn.Run,
		nad_fn.Run,
		dnn_fn.Run,
		if_fn.Run,
		UpfRun,
	}

	tlib.RunGoldenTestForPipelineOfFuncs(t, testcaseDir, pipeline)
}
