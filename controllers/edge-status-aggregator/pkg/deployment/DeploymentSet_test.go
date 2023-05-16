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
	"math/rand"
	"strconv"
	"sync"

	"github.com/go-logr/logr"
	"github.com/nephio-project/edge-status-aggregator/api/v1alpha1"
	edgewatcher "github.com/nephio-project/edge-watcher"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func createFakeDeploymentManager() *deploymentManager {
	subscriberChan := make(chan *edgewatcher.SubscriptionReq)
	cancellationChan := make(chan *edgewatcher.SubscriptionReq, 10)
	var deploymentManager = *NewDeploymentManager(
		subscriberChan, cancellationChan, nil,
		nil, logr.Discard(),
	)
	return &deploymentManager
}
func generateRandomNfDeploy(
	nfDeployTemplate v1alpha1.NfDeploy, expectedNFs int,
) v1alpha1.NfDeploy {
	numNFs := rand.Intn(expectedNFs) + 1
	numUPFs := rand.Intn(numNFs)
	numSMFs := 0
	if numNFs-numUPFs > 0 {
		numSMFs = rand.Intn(numNFs - numUPFs)
	}
	numAMFs := numNFs - numUPFs - numSMFs
	adjListSMFForUPFs := make(map[int][]int)
	adjListSMFForAMFs := make(map[int][]int)

	adjListUPF := make(map[int][]int)
	adjListAMF := make(map[int][]int)

	for upfId := 0; upfId < numUPFs; upfId++ {
		randomConnections := rand.Intn(numSMFs + 1)
		randomPermutation := rand.Perm(numSMFs + 1)
		for smfId, index := range randomPermutation {
			if index == randomConnections {
				break
			}
			adjListSMFForUPFs[smfId] = append(adjListSMFForUPFs[smfId], upfId)
			adjListUPF[upfId] = append(adjListUPF[upfId], smfId)
		}
	}
	for smfId := 0; smfId < numSMFs; smfId++ {
		randomConnections := rand.Intn(numAMFs + 1)
		randomPermutation := rand.Perm(numAMFs + 1)
		for amfId, index := range randomPermutation {
			if index == randomConnections {
				break
			}
			adjListSMFForAMFs[smfId] = append(adjListSMFForAMFs[smfId], amfId)
			adjListAMF[amfId] = append(adjListAMF[amfId], smfId)
		}
	}
	nfDeploy := v1alpha1.NfDeploy{
		TypeMeta:   nfDeployTemplate.TypeMeta,
		ObjectMeta: nfDeployTemplate.ObjectMeta,
		Spec: v1alpha1.NfDeploySpec{
			Capacity: "test", Sites: []v1alpha1.Site{},
		},
	}
	nfDeploy.Name = nfDeploy.Name + strconv.Itoa(rand.Intn(100))
	for upfId := 0; upfId < numUPFs; upfId++ {
		site := v1alpha1.Site{
			Id: "test-upf" + strconv.Itoa(upfId), NFType: "Upf",
			NFTypeName: "upf-test",
		}
		var connectivityList []v1alpha1.Connectivity
		for _, smfId := range adjListUPF[upfId] {
			connectivityList = append(
				connectivityList,
				v1alpha1.Connectivity{NeighborName: "test-smf" + strconv.Itoa(smfId)},
			)
		}
		site.Connectivities = connectivityList
		nfDeploy.Spec.Sites = append(nfDeploy.Spec.Sites, site)
	}
	for smfId := 0; smfId < numSMFs; smfId++ {
		site := v1alpha1.Site{
			Id: "test-smf" + strconv.Itoa(smfId), NFType: "Smf",
			NFTypeName: "smf-test",
		}
		var connectivityList []v1alpha1.Connectivity
		for _, upfId := range adjListSMFForUPFs[smfId] {
			connectivityList = append(
				connectivityList,
				v1alpha1.Connectivity{NeighborName: "test-upf" + strconv.Itoa(upfId)},
			)
		}
		for _, amfId := range adjListSMFForAMFs[smfId] {
			connectivityList = append(
				connectivityList,
				v1alpha1.Connectivity{NeighborName: "test-amf" + strconv.Itoa(amfId)},
			)
		}
		site.Connectivities = connectivityList
		nfDeploy.Spec.Sites = append(nfDeploy.Spec.Sites, site)
	}
	for amfId := 0; amfId < numAMFs; amfId++ {
		site := v1alpha1.Site{
			Id: "test-amf" + strconv.Itoa(amfId), NFType: "Amf",
			NFTypeName: "amf-test",
		}
		var connectivityList []v1alpha1.Connectivity
		for _, smfId := range adjListAMF[amfId] {
			connectivityList = append(
				connectivityList,
				v1alpha1.Connectivity{NeighborName: "test-smf" + strconv.Itoa(smfId)},
			)
		}
		site.Connectivities = connectivityList
		nfDeploy.Spec.Sites = append(nfDeploy.Spec.Sites, site)
	}
	return nfDeploy
}
func generateNfDeploys(
	nfDeployTemplate v1alpha1.NfDeploy, size int,
) []v1alpha1.NfDeploy {

	var nfDeployList []v1alpha1.NfDeploy
	for i := 0; i < size; i++ {
		x := rand.Intn(size) + 1
		nfDeployList = append(
			nfDeployList, generateRandomNfDeploy(nfDeployTemplate, x),
		)
	}
	return nfDeployList
}

func fakeEdgeWatcherSubscribe(
	deploymentManager *deploymentManager,
) {
	for {
		req, ok := <-deploymentManager.subscriberChan
		if !ok {
			break
		} else {
			nfDeployName := req.SubscriptionName

			deploymentManager.deploymentSet.deploymentSetMu.Lock()
			deployment := deploymentManager.deploymentSet.deployments[nfDeployName].deployment
			deploymentManager.deploymentSet.deploymentSetMu.Unlock()

			deployment.deploymentMu.Lock()
			// The thread running ReportNFDeploy waits on EdgeWatcher first for
			// subscription error and then for edge events. The below statements terminates
			// the thread by sending nil error and cancelling deployment context
			deployment.edgeErrorChan <- nil
			deployment.cancelCtx()
			deployment.deploymentMu.Unlock()
		}
	}
}

var _ = Describe(
	"ReportNFDeployEvent", func() {
		var deploymentManager deploymentManager
		var nfDeploy v1alpha1.NfDeploy
		BeforeEach(
			func() {
				nfDeploy = v1alpha1.NfDeploy{}
				deploymentManager = *createFakeDeploymentManager()
				testFile, _ := yaml.ReadFile("testfiles/nfdeploy.yaml")
				testFile.YNode().Decode(&nfDeploy)
			},
		)
		Context(
			"When NFDeploy is provided and deployment is not present for the NFDeploy",
			func() {
				It(
					"Should create new deployment and send a subscribe request to edgewatcher",
					func() {
						var wg sync.WaitGroup
						var wgWatcher sync.WaitGroup
						wgWatcher.Add(1)
						go func() {
							fakeEdgeWatcherSubscribe(&deploymentManager)
							wgWatcher.Done()
						}()
						wg.Add(1)
						go func() {
							deploymentManager.ReportNFDeployEvent(
								nfDeploy, types.NamespacedName{Name: nfDeploy.Name},
							)
							wg.Done()
						}()
						wg.Wait()
						close(deploymentManager.subscriberChan)
						wgWatcher.Wait()
						deploymentManager.deploymentSet.deploymentSetMu.Lock()
						Expect(len(deploymentManager.deploymentSet.deployments)).To(Equal(1))
						deploymentManager.deploymentSet.deploymentSetMu.Unlock()
					},
				)
			},
		)

		Context(
			"Fuzzy test", func() {
				It(
					"Should entertain all requests without errors",
					func() {
						rand.Seed(GinkgoRandomSeed())
						var wg sync.WaitGroup
						var randomNFDeployCount = rand.Intn(200)
						var nfDeployList = generateNfDeploys(nfDeploy, randomNFDeployCount)

						var nfDeploySet = make(map[string]void)
						var wgWatcher sync.WaitGroup
						wgWatcher.Add(1)
						go func() {
							fakeEdgeWatcherSubscribe(&deploymentManager)
							wgWatcher.Done()
						}()
						wg.Add(randomNFDeployCount)
						for i := 0; i < randomNFDeployCount; i++ {
							index := i
							namespacedName := types.NamespacedName{Name: nfDeployList[index].Name}
							go func() {
								deploymentManager.ReportNFDeployEvent(
									nfDeployList[index], namespacedName,
								)
								wg.Done()
							}()
						}
						for i := 0; i < randomNFDeployCount; i++ {
							nfDeploySet[nfDeployList[i].Name] = void{}
						}
						wg.Wait()
						close(deploymentManager.subscriberChan)
						wgWatcher.Wait()
						deploymentManager.deploymentSet.deploymentSetMu.Lock()
						Expect(len(deploymentManager.deploymentSet.deployments)).To(Equal(len(nfDeploySet)))
						deploymentManager.deploymentSet.deploymentSetMu.Unlock()
					},
				)
			},
		)

		Context(
			"When NFDeploy is provided and deployment is present for the NFDeploy",
			func() {
				It(
					"Should only update the existing deployment", func() {
						deployment := createSampleDeployment()
						deploymentInfo := DeploymentInfo{
							deploymentName: deployment.name, deployment: deployment,
						}
						deploymentManager.deploymentSet.deployments[deployment.name] = &deploymentInfo
						nfDeploy.Name = deployment.name
						deploymentManager.ReportNFDeployEvent(
							nfDeploy, types.NamespacedName{Name: nfDeploy.Name},
						)
						deploymentManager.deploymentSet.deploymentSetMu.Lock()
						Expect(len(deploymentManager.deploymentSet.deployments)).To(Equal(1))
						deploymentManager.deploymentSet.deploymentSetMu.Unlock()
					},
				)
			},
		)
	},
)

var _ = Describe("ReportNFDeployDeleteEvent", func() {

	var deploymentManager deploymentManager
	var nfDeploy v1alpha1.NfDeploy
	Context("When NFDeploy is provided for deletion", func() {
		BeforeEach(
			func() {
				nfDeploy = v1alpha1.NfDeploy{}
				deploymentManager = *createFakeDeploymentManager()
				testFile, _ := yaml.ReadFile("testfiles/nfdeploy.yaml")

				testFile.YNode().Decode(&nfDeploy)
			},
		)
		It("Should remove deployment from deploymentSet and cancel edgewatcher subscription", func() {
			go func() {
				fakeEdgeWatcherSubscribe(&deploymentManager)
			}()
			deploymentManager.ReportNFDeployEvent(
				nfDeploy, types.NamespacedName{Name: nfDeploy.Name},
			)
			Expect(len(deploymentManager.deploymentSet.deployments)).To(Equal(1))
			deploymentManager.ReportNFDeployDeleteEvent(
				nfDeploy,
			)
			cancelReq := <-deploymentManager.cancellationChan
			Expect(cancelReq).NotTo(BeNil())
			Expect(len(deploymentManager.deploymentSet.deployments)).To(Equal(0))

		})
	})
})
