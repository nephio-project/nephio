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

package controllers

import (
	"context"
	"fmt"

	nfdeployments "github.com/nephio-project/api/nf_deployments/v1alpha1"

	"github.com/nephio-project/edge-status-aggregator/api/v1alpha1"
	"github.com/nephio-project/edge-status-aggregator/tests/utils"
	"github.com/nephio-project/edge-status-aggregator/util"
	"github.com/nephio-project/edge-watcher/preprocessor"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

var (
	crNfDeployPath         = "../config/samples/nfdeploy_v1alpha1_nfdeploy.yaml"
	crCompleteNfDeployPath = "../config/samples/nfdeploy_with_all_nfs.yaml"
)

func generateUPFEdgeEvent(
	stalledStatus metav1.ConditionStatus, availableStatus metav1.ConditionStatus,
	readyStatus metav1.ConditionStatus, peeringStatus metav1.ConditionStatus,
	reconcilingStatus metav1.ConditionStatus, name string,
) preprocessor.Event {

	upfDeploy := nfdeployments.UPFDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{util.NFSiteIDLabel: name},
		},
		Status: nfdeployments.UPFDeploymentStatus{NFDeploymentStatus: nfdeployments.NFDeploymentStatus{
			Conditions: []metav1.Condition{
				{Type: string(nfdeployments.Stalled), Status: stalledStatus},
				{Type: string(nfdeployments.Reconciling), Status: reconcilingStatus},
				{Type: string(nfdeployments.Available), Status: availableStatus},
				{Type: string(nfdeployments.Peering), Status: peeringStatus},
				{Type: string(nfdeployments.Ready), Status: readyStatus},
			},
		},
		},
	}

	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&upfDeploy)
	Expect(err).To(
		BeNil(),
		"unable to convert UpfDeploy type to unstructured.Unstructured",
	)

	return preprocessor.Event{
		Key: preprocessor.RequestKey{Namespace: "upf", Kind: "UPFDeployment"},
		Object: &unstructured.Unstructured{
			Object: data,
		},
	}
}

func generateSMFEdgeEvent(
	stalledStatus metav1.ConditionStatus, availableStatus metav1.ConditionStatus,
	readyStatus metav1.ConditionStatus, peeringStatus metav1.ConditionStatus,
	reconcilingStatus metav1.ConditionStatus, name string,
) preprocessor.Event {

	smfDeploy := nfdeployments.SMFDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{util.NFSiteIDLabel: name},
		},
		Status: nfdeployments.SMFDeploymentStatus{NFDeploymentStatus: nfdeployments.NFDeploymentStatus{
			Conditions: []metav1.Condition{
				{Type: string(nfdeployments.Stalled), Status: stalledStatus},
				{Type: string(nfdeployments.Reconciling), Status: reconcilingStatus},
				{Type: string(nfdeployments.Available), Status: availableStatus},
				{Type: string(nfdeployments.Peering), Status: peeringStatus},
				{Type: string(nfdeployments.Ready), Status: readyStatus},
			},
		},
		},
	}

	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&smfDeploy)
	Expect(err).To(
		BeNil(),
		"unable to convert SmfDeploy type to unstructured.Unstructured",
	)

	return preprocessor.Event{
		Key: preprocessor.RequestKey{Namespace: "smf", Kind: "SMFDeployment"},
		Object: &unstructured.Unstructured{
			Object: data,
		},
	}
}

func getNfDeployCr(path string) (*v1alpha1.NfDeploy, error) {
	u, err := utils.ParseYaml(
		path, schema.GroupVersionKind{
			Group:   "nfdeploy.nephio.org",
			Version: "v1alpha1",
			Kind:    "NfDeploy",
		},
	)
	if err != nil {
		return nil, err
	}
	nfDeploy := &v1alpha1.NfDeploy{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u, nfDeploy)
	if err != nil {
		return nil, err
	}
	nfDeploy.Namespace = "default"
	return nfDeploy, nil
}

func executeAndTestEdgeEventSequence(
	edgeEvents []preprocessor.Event,
	finalExpectedStatus map[v1alpha1.NFDeployConditionType]corev1.ConditionStatus,
	nfDeployName string,
) {
	nfDeploy, err := getNfDeployCr(crNfDeployPath)
	Expect(err).NotTo(HaveOccurred())
	nfDeploy.Name = nfDeployName
	Expect(k8sClient.Create(context.TODO(), nfDeploy)).Should(Succeed())
	Eventually(fakeDeploymentManager.SignalChan).Should(Receive(nil))
	req := <-fakeDeploymentManager.SubscriptionReqChan
	req.Error <- nil
	for _, edgeEvent := range edgeEvents {
		req.Channel <- edgeEvent
	}
	var newNfDeploy v1alpha1.NfDeploy
	Eventually(
		func() map[v1alpha1.NFDeployConditionType]corev1.ConditionStatus {
			// fetching latest nfDeploy
			err := k8sClient.Get(
				ctx, types.NamespacedName{
					Namespace: nfDeploy.Namespace,
					Name:      nfDeploy.Name,
				}, &newNfDeploy,
			)
			if err != nil {
				return nil
			}
			newMap := make(map[v1alpha1.NFDeployConditionType]corev1.ConditionStatus)
			for _, c := range newNfDeploy.Status.Conditions {
				newMap[c.Type] = c.Status
			}
			return newMap
		},
	).Should(Equal(finalExpectedStatus))

}

var _ = Describe(
	"NfDeploy Controller", func() {

		Context(
			"When NfDeploy is created and updated", func() {
				It(
					"Should report to Deployment Manager", func() {
						nfDeploy, err := getNfDeployCr(crNfDeployPath)
						Expect(err).NotTo(HaveOccurred())
						nfDeploy.Name = "deployment-manager-report-test"
						Expect(k8sClient.Create(context.TODO(), nfDeploy)).Should(Succeed())
						Eventually(fakeDeploymentManager.SignalChan).Should(Receive(nil))
						req := <-fakeDeploymentManager.SubscriptionReqChan
						req.Error <- nil
						err = retry.RetryOnConflict(
							retry.DefaultRetry, func() error {
								// fetching latest nfDeploy
								var newNfDeploy v1alpha1.NfDeploy
								if err := k8sClient.Get(
									context.TODO(), types.NamespacedName{
										Namespace: nfDeploy.Namespace,
										Name:      nfDeploy.Name,
									}, &newNfDeploy,
								); err != nil {
									return err
								}
								newNfDeploy.Spec.Plmn.MCC = newNfDeploy.Spec.Plmn.MCC + 1
								err := k8sClient.Update(context.TODO(), &newNfDeploy)
								return err
							},
						)
						Expect(err).To(BeNil())
						Eventually(fakeDeploymentManager.SignalChan).Should(Receive(nil))
					},
				)
			},
		)
		Context(
			"When nfdeploy validation fails", func() {
				It(
					"Should never call deployment manager", func() {
						nfDeploy, err := getNfDeployCr(crNfDeployPath)
						Expect(err).NotTo(HaveOccurred())
						newSites := []v1alpha1.Site{nfDeploy.Spec.Sites[0]}
						nfDeploy.Spec.Sites = newSites
						nfDeploy.Name = "deployment-nfdeploy-validation"
						Expect(k8sClient.Create(context.TODO(), nfDeploy)).Should(Succeed())
						Consistently(fakeDeploymentManager.SignalChan).Should(Not(Receive(nil)))
					},
				)
			},
		)

		Context(
			"NfDeploy is deleted", func() {
				It(
					"Should delete the nfDeploy resource",
					func() {
						nfDeploy, err := getNfDeployCr(crNfDeployPath)
						Expect(err).NotTo(HaveOccurred())
						nfDeploy.Name = "nfdeploy-deletion"

						Expect(k8sClient.Create(context.TODO(), nfDeploy)).Should(Succeed())
						Eventually(fakeDeploymentManager.SignalChan).Should(Receive(nil))
						var newNfDeploy v1alpha1.NfDeploy
						req := <-fakeDeploymentManager.SubscriptionReqChan
						req.Error <- nil
						Expect(
							k8sClient.Get(
								ctx, types.NamespacedName{
									Namespace: nfDeploy.Namespace,
									Name:      nfDeploy.Name,
								}, &newNfDeploy,
							),
						).Should(Succeed())
						Expect(k8sClient.Delete(ctx, &newNfDeploy)).Should(Succeed())
						Eventually(fakeDeploymentManager.SignalChan).Should(Receive(nil))
						err = k8sClient.Get(
							ctx, types.NamespacedName{
								Namespace: nfDeploy.Namespace,
								Name:      nfDeploy.Name,
							}, &newNfDeploy,
						)
						Expect(err).NotTo(HaveOccurred())
						Expect(newNfDeploy.ObjectMeta.DeletionTimestamp.IsZero()).To(BeFalse())
						Eventually(
							func() error {
								return k8sClient.Get(
									ctx, types.NamespacedName{
										Namespace: nfDeploy.Namespace,
										Name:      nfDeploy.Name,
									}, &newNfDeploy,
								)
							},
						).ShouldNot(Succeed())
					},
				)
			},
		)

		Context(
			"Deployment - NFDeployController Integration", func() {

				Context(
					"When edge returns error during connection establishing", func() {
						It(
							"Should set nfdeploy resource status", func() {
								nfDeploy, _ := getNfDeployCr(crNfDeployPath)
								nfDeploy.Name = "connection-failure-test"

								Expect(
									k8sClient.Create(
										context.TODO(), nfDeploy,
									),
								).Should(Succeed())
								Eventually(fakeDeploymentManager.SignalChan).Should(Receive(nil))
								req := <-fakeDeploymentManager.SubscriptionReqChan
								req.Error <- fmt.Errorf("test error from edgewatcher")
								var newNfDeploy v1alpha1.NfDeploy

								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}
										conditionTypes := []v1alpha1.NFDeployConditionType{}
										for _, c := range newNfDeploy.Status.Conditions {
											conditionTypes = append(conditionTypes, c.Type)
											if c.Type == v1alpha1.DeploymentReconciling {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("EdgeConnectionFailure"))
								for _, value := range newNfDeploy.Status.Conditions {
									Expect(value.Status).To(Equal(corev1.ConditionUnknown))
									Expect(value.Reason).To(Equal("EdgeConnectionFailure"))
								}
							},
						)
					},
				)

				Context(
					"When edge channel closes unexpectedly during connection establishing",
					func() {
						It(
							"Should set nfdeploy resource status", func() {

								nfdeploy2, _ := getNfDeployCr(crNfDeployPath)
								nfdeploy2.Name = "edge-channel-test"
								Expect(
									k8sClient.Create(
										context.TODO(), nfdeploy2,
									),
								).Should(Succeed())
								Eventually(fakeDeploymentManager.SignalChan).Should(Receive(nil))
								req := <-fakeDeploymentManager.SubscriptionReqChan
								close(req.Error)
								var newNfDeploy v1alpha1.NfDeploy
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfdeploy2.Namespace,
												Name:      nfdeploy2.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}
										conditionTypes := []v1alpha1.NFDeployConditionType{}
										for _, c := range newNfDeploy.Status.Conditions {
											conditionTypes = append(conditionTypes, c.Type)
											if c.Type == v1alpha1.DeploymentReconciling {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("EdgeConnectionFailure"))
								for _, value := range newNfDeploy.Status.Conditions {
									Expect(value.Status).To(Equal(corev1.ConditionUnknown))
									Expect(value.Reason).To(Equal("EdgeConnectionFailure"))
								}
							},
						)
					},
				)
				Context(
					"When edge channel closes unexpectedly while listening to edge events",
					func() {
						It(
							"Should set nfdeploy resource status", func() {
								nfDeploy, _ := getNfDeployCr(crNfDeployPath)
								nfDeploy.Name = "edge-connection-closed-test"
								Expect(
									k8sClient.Create(
										context.TODO(), nfDeploy,
									),
								).Should(Succeed())
								Eventually(fakeDeploymentManager.SignalChan).Should(Receive(nil))

								req := <-fakeDeploymentManager.SubscriptionReqChan
								req.Error <- nil
								close(req.SubscriberInfo.Channel)
								var newNfDeploy v1alpha1.NfDeploy
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}
										conditionTypes := []v1alpha1.NFDeployConditionType{}
										for _, c := range newNfDeploy.Status.Conditions {
											conditionTypes = append(conditionTypes, c.Type)
											if c.Type == v1alpha1.DeploymentReconciling {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("EdgeConnectionBroken"))
								for _, value := range newNfDeploy.Status.Conditions {
									Expect(value.Status).To(Equal(corev1.ConditionUnknown))
									Expect(value.Reason).To(Equal("EdgeConnectionBroken"))
								}
							},
						)
					},
				)

				Context(
					"When edge event with ambiguous condition set is provided", func() {
						It(
							"Should not update nfdeploy resource status", func() {
								nfDeploy, _ := getNfDeployCr(crNfDeployPath)
								nfDeploy.Name = "ambiguous-test"
								Expect(
									k8sClient.Create(
										context.TODO(), nfDeploy,
									),
								).Should(Succeed())
								Eventually(fakeDeploymentManager.SignalChan).Should(Receive(nil))

								req := <-fakeDeploymentManager.SubscriptionReqChan
								req.Error <- nil
								req.SubscriberInfo.Channel <- generateUPFEdgeEvent(
									metav1.ConditionUnknown, metav1.ConditionUnknown,
									metav1.ConditionUnknown, metav1.ConditionUnknown,
									metav1.ConditionUnknown, "upf-dummy",
								)
								var newNfDeploy v1alpha1.NfDeploy
								Consistently(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										conditionTypes := []v1alpha1.NFDeployConditionType{}
										for _, c := range newNfDeploy.Status.Conditions {
											conditionTypes = append(conditionTypes, c.Type)
											if c.Type == v1alpha1.DeploymentReconciling {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("NewVersionAvailable"))

							},
						)
					},
				)

				Context(
					"Testing reconciling status for valid edge events", func() {
						It(
							"Should update nfdeploy status", func() {
								// Two NFs in test resource
								nfDeploy, _ := getNfDeployCr(crNfDeployPath)
								nfDeploy.Name = "reconciling-test"
								Expect(
									k8sClient.Create(
										context.TODO(), nfDeploy,
									),
								).Should(Succeed())
								Eventually(fakeDeploymentManager.SignalChan).Should(Receive(nil))

								req := <-fakeDeploymentManager.SubscriptionReqChan
								req.Error <- nil

								// first NF reconciling
								req.SubscriberInfo.Channel <- generateUPFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionTrue, "upf-dummy",
								)
								var newNfDeploy v1alpha1.NfDeploy
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentReconciling {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("SomeNFsReconciling"))

								// second NF reconciling
								req.SubscriberInfo.Channel <- generateSMFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionTrue, "smf-dummy",
								)
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentReconciling {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("AllUnReconciledNFsReconciling"))

								// No NF reconciling
								req.SubscriberInfo.Channel <- generateUPFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, "upf-dummy",
								)
								req.SubscriberInfo.Channel <- generateSMFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, "smf-dummy",
								)
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentReconciling {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("NoNFsReconciling"))

								// All NFs reconciled
								req.SubscriberInfo.Channel <- generateUPFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionTrue,
									metav1.ConditionTrue, metav1.ConditionFalse,
									metav1.ConditionFalse, "upf-dummy",
								)
								req.SubscriberInfo.Channel <- generateSMFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionTrue,
									metav1.ConditionTrue, metav1.ConditionFalse,
									metav1.ConditionFalse, "smf-dummy",
								)
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentReconciling {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("AllNFsReconciled"))
							},
						)
					},
				)
				Context(
					"Testing peering status for valid edge events", func() {
						It(
							"Should update nfdeploy status", func() {
								// Two NFs in test resource
								nfDeploy, _ := getNfDeployCr(crNfDeployPath)
								nfDeploy.Name = "peering-test"
								Expect(
									k8sClient.Create(
										context.TODO(), nfDeploy,
									),
								).Should(Succeed())
								Eventually(fakeDeploymentManager.SignalChan).Should(Receive(nil))

								req := <-fakeDeploymentManager.SubscriptionReqChan
								req.Error <- nil

								// first NF peering
								req.SubscriberInfo.Channel <- generateUPFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionTrue,
									metav1.ConditionFalse, metav1.ConditionTrue,
									metav1.ConditionTrue, "upf-dummy",
								)
								var newNfDeploy v1alpha1.NfDeploy
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentPeering {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("SomeNFsPeering"))

								// second NF peering
								req.SubscriberInfo.Channel <- generateSMFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionTrue,
									metav1.ConditionFalse, metav1.ConditionTrue,
									metav1.ConditionTrue, "smf-dummy",
								)
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentPeering {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("AllUnPeeredNFsPeering"))

								// No NF peering
								req.SubscriberInfo.Channel <- generateUPFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, "upf-dummy",
								)
								req.SubscriberInfo.Channel <- generateSMFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, "smf-dummy",
								)
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentPeering {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("NoNFsPeering"))

								// All NFs peered
								req.SubscriberInfo.Channel <- generateUPFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionTrue,
									metav1.ConditionTrue, metav1.ConditionFalse,
									metav1.ConditionFalse, "upf-dummy",
								)
								req.SubscriberInfo.Channel <- generateSMFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionTrue,
									metav1.ConditionTrue, metav1.ConditionFalse,
									metav1.ConditionFalse, "smf-dummy",
								)
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentPeering {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("AllNFsPeered"))
							},
						)
					},
				)
				Context(
					"Testing ready status for valid edge events", func() {
						It(
							"Should update nfdeploy status", func() {
								// Two NFs in test resource
								nfDeploy, _ := getNfDeployCr(crNfDeployPath)
								nfDeploy.Name = "ready-test"
								Expect(
									k8sClient.Create(
										context.TODO(), nfDeploy,
									),
								).Should(Succeed())
								Eventually(fakeDeploymentManager.SignalChan).Should(Receive(nil))

								req := <-fakeDeploymentManager.SubscriptionReqChan
								req.Error <- nil

								// first NF ready
								req.SubscriberInfo.Channel <- generateUPFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionTrue,
									metav1.ConditionTrue, metav1.ConditionFalse,
									metav1.ConditionFalse, "upf-dummy",
								)
								var newNfDeploy v1alpha1.NfDeploy
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentReady {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("SomeNFsReady"))

								// second NF ready
								req.SubscriberInfo.Channel <- generateSMFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionTrue,
									metav1.ConditionTrue, metav1.ConditionFalse,
									metav1.ConditionFalse, "smf-dummy",
								)
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentReady {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("AllNFsReady"))

								// No NF ready
								req.SubscriberInfo.Channel <- generateUPFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, "upf-dummy",
								)
								req.SubscriberInfo.Channel <- generateSMFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, "smf-dummy",
								)
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentReady {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("NoNFsReady"))

							},
						)
					},
				)
				Context(
					"Testing stalled status for valid edge events", func() {
						It(
							"Should update nfdeploy status", func() {
								// Two NFs in test resource
								nfDeploy, _ := getNfDeployCr(crNfDeployPath)
								nfDeploy.Name = "stalled-test"
								Expect(
									k8sClient.Create(
										context.TODO(), nfDeploy,
									),
								).Should(Succeed())
								Eventually(fakeDeploymentManager.SignalChan).Should(Receive(nil))

								req := <-fakeDeploymentManager.SubscriptionReqChan
								req.Error <- nil

								// first NF stalled
								req.SubscriberInfo.Channel <- generateUPFEdgeEvent(
									metav1.ConditionTrue, metav1.ConditionFalse,
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, "upf-dummy",
								)
								req.SubscriberInfo.Channel <- generateSMFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionTrue,
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, "smf-dummy",
								)
								var newNfDeploy v1alpha1.NfDeploy
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentStalled {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("SomeNFsStalled"))

								// second NF stalled
								req.SubscriberInfo.Channel <- generateSMFEdgeEvent(
									metav1.ConditionTrue, metav1.ConditionFalse,
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, "smf-dummy",
								)
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentStalled {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("AllNFsStalled"))

								// No NF stalled
								req.SubscriberInfo.Channel <- generateUPFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionTrue,
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, "upf-dummy",
								)
								req.SubscriberInfo.Channel <- generateSMFEdgeEvent(
									metav1.ConditionFalse, metav1.ConditionTrue,
									metav1.ConditionFalse, metav1.ConditionFalse,
									metav1.ConditionFalse, "smf-dummy",
								)
								Eventually(
									func() string {
										// fetching latest nfDeploy
										err := k8sClient.Get(
											ctx, types.NamespacedName{
												Namespace: nfDeploy.Namespace,
												Name:      nfDeploy.Name,
											}, &newNfDeploy,
										)
										if err != nil {
											return ""
										}

										for _, c := range newNfDeploy.Status.Conditions {
											if c.Type == v1alpha1.DeploymentStalled {
												return c.Reason
											}
										}
										return ""
									},
								).Should(Equal("NoNFsStalled"))

							},
						)
					},
				)
				Context(
					"Testing edge event sequence", func() {
						It(
							"Should update correct NFDeploy status", func() {
								var edgeEvents []preprocessor.Event
								edgeEvents = append(
									edgeEvents, generateUPFEdgeEvent(
										metav1.ConditionFalse, metav1.ConditionFalse,
										metav1.ConditionFalse, metav1.ConditionFalse,
										metav1.ConditionFalse, "upf-dummy",
									), generateUPFEdgeEvent(
										metav1.ConditionFalse, metav1.ConditionFalse,
										metav1.ConditionTrue, metav1.ConditionFalse,
										metav1.ConditionFalse, "upf-dummy",
									),
									generateUPFEdgeEvent(
										metav1.ConditionFalse, metav1.ConditionTrue,
										metav1.ConditionFalse, metav1.ConditionTrue,
										metav1.ConditionFalse, "upf-dummy",
									),
									generateUPFEdgeEvent(
										metav1.ConditionFalse, metav1.ConditionFalse,
										metav1.ConditionTrue, metav1.ConditionTrue,
										metav1.ConditionFalse, "upf-dummy",
									),
									generateUPFEdgeEvent(
										metav1.ConditionTrue, metav1.ConditionFalse,
										metav1.ConditionTrue, metav1.ConditionFalse,
										metav1.ConditionFalse, "upf-dummy",
									), generateUPFEdgeEvent(
										metav1.ConditionFalse, metav1.ConditionTrue,
										metav1.ConditionTrue, metav1.ConditionFalse,
										metav1.ConditionFalse, "upf-dummy",
									), generateSMFEdgeEvent(
										metav1.ConditionFalse, metav1.ConditionTrue,
										metav1.ConditionTrue, metav1.ConditionFalse,
										metav1.ConditionFalse, "smf-dummy",
									),
								)
								finalExpectedStatus := make(map[v1alpha1.NFDeployConditionType]corev1.ConditionStatus)
								finalExpectedStatus[v1alpha1.DeploymentStalled] = corev1.ConditionFalse
								finalExpectedStatus[v1alpha1.DeploymentPeering] = corev1.ConditionFalse
								finalExpectedStatus[v1alpha1.DeploymentReady] = corev1.ConditionTrue
								finalExpectedStatus[v1alpha1.DeploymentReconciling] = corev1.ConditionFalse
								executeAndTestEdgeEventSequence(
									edgeEvents, finalExpectedStatus, "edge-sequence-test-1",
								)
							},
						)
					},
				)
				Context(
					"Testing edge events with unknown status condition set", func() {
						It(
							"Should update correct NFDeploy status", func() {
								var edgeEvents []preprocessor.Event
								edgeEvents = append(
									edgeEvents, generateUPFEdgeEvent(
										metav1.ConditionFalse, metav1.ConditionFalse,
										metav1.ConditionFalse, metav1.ConditionFalse,
										metav1.ConditionFalse, "upf-dummy",
									), generateUPFEdgeEvent(
										metav1.ConditionUnknown, metav1.ConditionUnknown,
										metav1.ConditionTrue, metav1.ConditionUnknown,
										metav1.ConditionUnknown, "upf-dummy",
									), generateSMFEdgeEvent(
										metav1.ConditionTrue, metav1.ConditionUnknown,
										metav1.ConditionUnknown, metav1.ConditionUnknown,
										metav1.ConditionTrue, "smf-dummy",
									),
								)
								finalExpectedStatus := make(map[v1alpha1.NFDeployConditionType]corev1.ConditionStatus)
								finalExpectedStatus[v1alpha1.DeploymentStalled] = corev1.ConditionTrue
								finalExpectedStatus[v1alpha1.DeploymentPeering] = corev1.ConditionFalse
								finalExpectedStatus[v1alpha1.DeploymentReady] = corev1.ConditionFalse
								finalExpectedStatus[v1alpha1.DeploymentReconciling] = corev1.ConditionTrue
								executeAndTestEdgeEventSequence(
									edgeEvents, finalExpectedStatus, "edge-sequence-test-2",
								)
							},
						)
					},
				)
			},
		)
		//TODO : Integrate Deployment state cleanup on NFDeploy deletion Unit tests.

	},
)
