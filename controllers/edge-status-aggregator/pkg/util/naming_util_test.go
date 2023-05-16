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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nephio-project/edge-status-aggregator/util"
)

var _ = Describe("NamingUtil", func() {
	Context("With a sample naming context", func() {
		nc, err := util.NewNamingContext("cluster", "nfDeploy")

		It("should return valid namespace", func() {
			Expect(nc.GetNamespace()).To(Equal("nephio-user"))
		})
		It("should return valid nf profiles repo name", func() {
			Expect(nc.GetNFProfileRepoName()).To(Equal("private-catalog"))
		})
		It("should return valid vendor NF related manifests repo name", func() {
			Expect(nc.GetVendorNFManifestsRepoName()).To(Equal("private-catalog"))
		})
		It("should return valid nf profiles package name", func() {
			Expect(nc.GetNFProfilePackageName()).To(Equal("nf-profiles"))
		})
		It("should return valid new deploy package name", func() {
			Expect(nc.GetDeployPackageName()).To(Equal("nfDeploy-cluster"))
		})
		It("should return valid deploy repo name", func() {
			Expect(nc.GetDeployRepoName()).To(Equal("cluster-deploy-repo"))
		})
		It("should return valid actuator package name", func() {
			Expect(nc.GetNFDeployActuatorPackageName("ABC", "1.0", "upf")).
				To(Equal("ABC/1.0/upf/actuators"))
		})
		It("should return valid vendor extension package name", func() {
			Expect(nc.GetVendorExtensionPackageName("ABC", "1.0", "upf")).
				To(Equal("ABC/1.0/upf/extension"))
		})
		It("should return valid nfDeploy name", func() {
			Expect(nc.GetNfDeployName()).To(Equal("nfDeploy"))
		})
		It("should return nil err", func() {
			Expect(err).To(Succeed())
		})
	})

	Context("With empty input values to naming context", func() {
		It("should return err for empty cluster name", func() {
			_, err := util.NewNamingContext("", "nfDeploy")
			Expect(err).To(HaveOccurred())
		})
		It("should return err for empty nfDeploy name", func() {
			_, err := util.NewNamingContext("cluster", "")
			Expect(err).To(HaveOccurred())
		})
	})
})
