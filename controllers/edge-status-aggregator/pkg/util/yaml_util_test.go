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

package util_test

import (
	util "github.com/nephio-project/edge-status-aggregator/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("YamlUtil", func() {
	validYaml :=
		`apiVersion: sourcerepo.cnrm.cloud.google.com/v1beta1
kind: SourceRepoRepository
metadata:
  name: private-repo
  namespace: config-control # kpt-set: ${config-namespace}
---
apiVersion: sourcerepo.cnrm.cloud.google.com/v1beta1
kind: SourceRepoRepository
metadata:
  name: source-repo
  namespace: config-control # kpt-set: ${config-namespace}`

	invalidYaml := `
		This is a random file like a readme and is not a valid yaml
		`

	Describe("parsing string to yaml nodes", func() {
		Context("with a valid yaml string", func() {
			nodes, err := util.ParseStringToYamlNode(validYaml)
			It("should return valid yaml nodes", func() {
				Expect(len(nodes)).To(Equal(2))
			})
			It("should have correct field values for each node", func() {
				Expect((nodes[0].GetApiVersion())).To(Equal("sourcerepo.cnrm.cloud.google.com/v1beta1"))
				Expect((nodes[0].GetKind())).To(Equal("SourceRepoRepository"))
				Expect((nodes[0].GetNamespace())).To(Equal("config-control"))
				Expect((nodes[0].GetName())).To(Equal("private-repo"))

				Expect((nodes[1].GetApiVersion())).To(Equal("sourcerepo.cnrm.cloud.google.com/v1beta1"))
				Expect((nodes[1].GetKind())).To(Equal("SourceRepoRepository"))
				Expect((nodes[1].GetNamespace())).To(Equal("config-control"))
				Expect((nodes[1].GetName())).To(Equal("source-repo"))
			})
			It("should return nil error", func() {
				Expect(err).To(Succeed())
			})
		})

		Context("with an invalid yaml string", func() {
			nodes, err := util.ParseStringToYamlNode(invalidYaml)
			It("should return nil yaml nodes", func() {
				Expect(nodes).To(BeNil())
			})
			It("should return some error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Get Matching Yaml nodes", func() {
		nodes, _ := util.ParseStringToYamlNode(validYaml)
		Context("match with apiversion and kind", func() {
			fNodes, err := util.GetMatchingYamlNodes(nodes, "sourcerepo.cnrm.cloud.google.com/v1beta1", "SourceRepoRepository", "")
			It("should return all valid yaml nodes with correct values", func() {
				Expect(len(fNodes)).To(Equal(2))
				Expect((fNodes[0])).To(Equal(nodes[0]))
				Expect((fNodes[1])).To(Equal(nodes[1]))
			})
			It("should return nil error", func() {
				Expect(err).To(Succeed())
			})
		})

		Context("match with apiversion, kind and name", func() {
			fNodes, err := util.GetMatchingYamlNodes(nodes, "sourcerepo.cnrm.cloud.google.com/v1beta1", "SourceRepoRepository", "private-repo")
			It("should return the valid yaml node", func() {
				Expect(len(fNodes)).To(Equal(1))
				Expect((fNodes[0])).To(Equal(nodes[0]))
			})
			It("should return nil error", func() {
				Expect(err).To(Succeed())
			})
		})

		Context("match with all nil values", func() {
			fNodes, err := util.GetMatchingYamlNodes(nodes, "", "", "")
			It("should return nil list", func() {
				Expect(fNodes).To(BeNil())
			})
			It("should return an error", func() {
				Expect(err).To(MatchError("Invalid input: every criteria is empty."))
			})
		})
	})
})
