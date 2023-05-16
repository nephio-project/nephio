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

package v1alpha1

import (
	"fmt"
	"math/rand"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("NFDeploy Validator Webhook", func() {
	Context("Test NfDeploy creation", Ordered, func() {
		var namespace *corev1.Namespace
		var object *NfDeploy
		BeforeAll(func() {
			namespace = &corev1.Namespace{
				ObjectMeta: v1.ObjectMeta{
					Name: fmt.Sprintf("namespace-%v", rand.Intn(100)),
				},
			}
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
		})
		BeforeEach(func(ctx SpecContext) {

			object = &NfDeploy{
				TypeMeta:   v1.TypeMeta{APIVersion: "nfdeploy.nephio.org/v1alpha1", Kind: "NfDeploy"},
				ObjectMeta: v1.ObjectMeta{Name: "test-nfdeploy", Namespace: namespace.Name},
				Spec: NfDeploySpec{
					Sites: []Site{{Id: "upf"}, {Id: "smf"}},
				},
			}

		})
		AfterAll(func() {
			Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
		})
		When("correct nfdeploy is provided", func() {
			It("Should return no error", func(ctx SpecContext) {
				err := k8sClient.Create(ctx, object)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("duplicate node is provided", func() {
			It("Should return error", func(ctx SpecContext) {
				object.Spec.Sites = append(object.Spec.Sites, Site{Id: object.Spec.Sites[0].Id})
				err := k8sClient.Create(ctx, object)
				Expect(err).To(HaveOccurred())
				Expect(apierrors.ReasonForError(err)).To(Equal(v1.StatusReason("NF with id - upf is already present")))

			})
		})
		When("When multiple connections between two NFs are provided", func() {
			It("Should return error", func(ctx SpecContext) {
				object.Spec.Sites[0].Connectivities = []Connectivity{{NeighborName: object.Spec.Sites[1].Id}, {NeighborName: object.Spec.Sites[1].Id}}
				object.Spec.Sites[1].Connectivities = []Connectivity{{NeighborName: object.Spec.Sites[0].Id}}
				err := k8sClient.Create(ctx, object)
				Expect(err).To(HaveOccurred())
				Expect(apierrors.ReasonForError(err)).To(Equal(v1.StatusReason("Multiple connections found between upf and smf")))

			})
		})
		When("When a connected NF is not present in any site", func() {
			It("Should return error", func(ctx SpecContext) {
				object.Spec.Sites[0].Connectivities = []Connectivity{{NeighborName: "random"}, {NeighborName: object.Spec.Sites[1].Id}}
				object.Spec.Sites[1].Connectivities = []Connectivity{{NeighborName: object.Spec.Sites[0].Id}}
				err := k8sClient.Create(ctx, object)
				Expect(err).To(HaveOccurred())
				Expect(apierrors.ReasonForError(err)).To(Equal(v1.StatusReason("NF with id random is not present")))
			})
		})
		When("When connection between two NFs is not mutual", func() {
			It("Should return error", func(ctx SpecContext) {
				object.Spec.Sites[1].Connectivities = []Connectivity{{NeighborName: object.Spec.Sites[0].Id}}
				err := k8sClient.Create(ctx, object)
				Expect(err).To(HaveOccurred())
				Expect(apierrors.ReasonForError(err)).To(Equal(v1.StatusReason("Connectivity between upf and smf is not present")))

			})
		})
	})
})
