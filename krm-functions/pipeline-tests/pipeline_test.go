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

	dnn_fn "github.com/nephio-project/nephio/krm-functions/dnn-fn/fn"
	if_fn "github.com/nephio-project/nephio/krm-functions/interface-fn/fn"
	ipam_fn "github.com/nephio-project/nephio/krm-functions/ipam-fn/fn"
	nad_fn "github.com/nephio-project/nephio/krm-functions/nad-fn/fn"
	nfdeploy_fn "github.com/nephio-project/nephio/krm-functions/nfdeploy-fn/fn"
	vlan_fn "github.com/nephio-project/nephio/krm-functions/vlan-fn/fn"
	"github.com/nokia/k8s-ipam/pkg/proxy/clientproxy/ipam"
	"github.com/nokia/k8s-ipam/pkg/proxy/clientproxy/vlan"
)

const testdir = "testdata"

func nfFn(rl *fn.ResourceList) (bool, error) {
	return nfdeploy_fn.Run(rl)
}

var ipamFn = ipam_fn.New(ipam.NewMock())
var vlanFn = vlan_fn.New(vlan.NewMock())

type TestCase struct {
	pipeline        []fn.ResourceListProcessorFunc
	inputDir        string
	expectedDataDir string
}

func TestPipelines(t *testing.T) {
	tcs := []TestCase{
		{
			inputDir:        "upf_pkg_init",
			expectedDataDir: "workload_cluster_not_ready",
			pipeline: []fn.ResourceListProcessorFunc{
				nfFn,
				if_fn.Run,
				dnn_fn.Run,

				ipamFn.Run,
				vlanFn.Run,

				nad_fn.Run,
				if_fn.Run,
				dnn_fn.Run,
				nfFn,
			},
		},
		{
			inputDir:        "upf_pkg",
			expectedDataDir: "simplified_deployment",
			pipeline: []fn.ResourceListProcessorFunc{
				nfFn,
				if_fn.Run,
				dnn_fn.Run,

				ipamFn.Run,
				vlanFn.Run,

				nad_fn.Run,
				if_fn.Run,
				dnn_fn.Run,
				nfFn,
			},
		},
		{
			inputDir:        "upf_pkg",
			expectedDataDir: "real_deployment",
			pipeline: []fn.ResourceListProcessorFunc{
				nfFn,
				if_fn.Run,
				nad_fn.Run,
				if_fn.Run,
				dnn_fn.Run,
				nfFn,

				ipamFn.Run,
				vlanFn.Run,

				nad_fn.Run,
				if_fn.Run,
				dnn_fn.Run,
				nfFn,
			},
		},
		{
			inputDir:        "upf_pkg",
			expectedDataDir: "real_deployment_2",
			pipeline: []fn.ResourceListProcessorFunc{
				nfFn,
				if_fn.Run,
				nad_fn.Run,
				if_fn.Run,
				dnn_fn.Run,
				nfFn,

				ipamFn.Run,

				nad_fn.Run,
				if_fn.Run,
				dnn_fn.Run,
				nfFn,

				vlanFn.Run,

				nad_fn.Run,
				if_fn.Run,
				dnn_fn.Run,
				nfFn,
			},
		},
		{
			inputDir:        "upf_pkg",
			expectedDataDir: "real_deployment_2",
			pipeline: []fn.ResourceListProcessorFunc{
				nfFn,
				nfFn,
				if_fn.Run,
				nad_fn.Run,
				if_fn.Run,
				dnn_fn.Run,
				nfFn,

				ipamFn.Run,

				nad_fn.Run,
				if_fn.Run,
				dnn_fn.Run,
				nfFn,

				vlanFn.Run,

				nad_fn.Run,
				if_fn.Run,
				dnn_fn.Run,
				nfFn,
				nfFn,
				nfFn,
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.expectedDataDir, func(t *testing.T) {
			inputDir := filepath.Join(testdir, tc.inputDir)
			expectedDir := filepath.Join(testdir, tc.expectedDataDir)
			tlib.RunGoldenTestForPipelineOfFuncs(t, inputDir, tc.pipeline, expectedDir)
		})
	}
}
