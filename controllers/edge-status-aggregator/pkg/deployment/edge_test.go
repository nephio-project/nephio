/*
Copyright 2022-2023 The Nephio Authors.

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

package deployment

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func createSampleEdge() Edge {
	return Edge{FirstNode: sampleUPFName, SecondNode: sampleSMFName}
}

var _ = Describe(
	"IsEqual", func() {
		var edge Edge
		BeforeEach(
			func() {
				edge = createSampleEdge()
			},
		)
		Context(
			"When edges are connected", func() {
				It(
					"Should return true", func() {
						result := edge.IsEqual(sampleSMFName, sampleUPFName)
						Expect(result).To(Equal(true))
						result = edge.IsEqual(sampleUPFName, sampleSMFName)
						Expect(result).To(Equal(true))
					},
				)
			},
		)

		Context(
			"When edges are not connected", func() {
				It(
					"Should return false", func() {
						result := edge.IsEqual(sampleAMFName, sampleUPFName)
						Expect(result).To(Equal(false))
					},
				)
			},
		)
	},
)
