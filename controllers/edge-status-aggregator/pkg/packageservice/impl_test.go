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

package packageservice_test

//go:generate ../bin/mockgen -source=interface.go -destination=mock/interface_mock.go -package=mock -copyright_file=../hack/copyright.txt
//go:generate ../bin/mockgen -destination=mocks/mock_client.go -package=mocks -copyright_file=../hack/copyright.txt sigs.k8s.io/controller-runtime/pkg/client Client

import (
	"context"
	"errors"
	"os"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/golang/mock/gomock"
	"github.com/nephio-project/edge-status-aggregator/mocks"
	"github.com/nephio-project/edge-status-aggregator/packageservice"
	"github.com/nephio-project/edge-status-aggregator/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getPackageRevisionCR(name string,
	rName string,
	pName string,
	rev string,
	isPublished bool,
	isLatest bool) porchapi.PackageRevision {
	lifecycle := porchapi.PackageRevisionLifecycleDraft
	if isPublished {
		lifecycle = porchapi.PackageRevisionLifecyclePublished
	}
	labels := map[string]string{}
	if isLatest {
		labels[porchapi.LatestPackageRevisionKey] = porchapi.LatestPackageRevisionValue
	}
	return porchapi.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "nephio-user",
			Labels:    labels,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    pName,
			RepositoryName: rName,
			Lifecycle:      lifecycle,
			Revision:       rev,
		},
	}
}

func getPackageRevisionResourceCR(resources map[string]string) *porchapi.PackageRevisionResources {
	return &porchapi.PackageRevisionResources{
		Spec: porchapi.PackageRevisionResourcesSpec{
			Resources: resources,
		},
	}
}

var _ = Describe("Impl", func() {
	var (
		ps              *packageservice.PorchPackageService
		mockCtrl        *gomock.Controller
		mockClient      *mocks.MockClient
		resourceRequest []packageservice.GetResourceRequest
		nc              util.NamingContext
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClient(mockCtrl)
		ps = &packageservice.PorchPackageService{
			Client: mockClient,
			Log:    ctrl.Log.WithName("PorchPackageService"),
		}
		nc, _ = util.NewNamingContext("clusterName", "nfDeployName")
		resourceRequest = []packageservice.GetResourceRequest{
			{
				ID:         1,
				Kind:       "SourceRepoRepository",
				ApiVersion: "sourcerepo.cnrm.cloud.google.com/v1beta1",
			},
			{
				ID:         2,
				Kind:       "Repository",
				ApiVersion: "config.porch.kpt.dev/v1alpha1",
				Name:       "private-repo",
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("testing GetNFProfiles via Porch from pre-defined package and repo", func() {
		Context("valid inputs expecting a response", func() {
			BeforeEach(func() {
				mockClient.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("prev1", "private-catalog", "nf-profiles", "v1", true, false),  // older published version
							getPackageRevisionCR("prev2", "private-catalog", "nf-profiles", "v2", true, true),   // latest published version
							getPackageRevisionCR("prev3", "private-catalog", "nf-profiles", "v3", false, false), // newer version but not published
							getPackageRevisionCR("prev4", "private-catalog", "nf-profiles-1", "v2", true, true), // latest published version of similar named package
							getPackageRevisionCR("prev5", "soure-repo", "nf-profiles", "v2", true, true),        // latest published version but in different repo
						}
					})
			})

			It("should select the latest version in private-catalog when multiple nf-profiles versions present", func() {
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, key client.ObjectKey, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
						Expect(key.Name).To(Equal("prev2")) // prev2 is the latest revision in above list
						Expect(key.Namespace).To(Equal("nephio-user"))
					})
				ps.GetNFProfiles(context.TODO(), resourceRequest, nc)
			})

			It("should correctly parsing content with multiple resources separated by '---' and return filtered manifests as per GetResourceRequest", func() {
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, key client.ObjectKey, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
						prr.Spec.Resources = map[string]string{}
						dat1, _ := os.ReadFile("samplepkg/private-repo.yaml")
						prr.Spec.Resources["private-repo.yaml"] = string(dat1)
						dat2, _ := os.ReadFile("samplepkg/source-repo.yaml")
						prr.Spec.Resources["source-repo.yaml"] = string(dat2)
					})
				profiles, err := ps.GetNFProfiles(context.TODO(), resourceRequest, nc)
				Expect(err).To(Succeed())

				// validate the profiles response
				Expect(len(profiles)).To(Equal(2))
				Expect(len(profiles[1])).To(Equal(2))
				Expect(len(profiles[2])).To(Equal(1))
				nodes, _ := util.ParseStringToYamlNode(profiles[2][0])

				// verifying response fields with request details for one of the requests
				Expect(nodes[0].GetName()).To(Equal(resourceRequest[1].Name))
				Expect(nodes[0].GetApiVersion()).To(Equal(resourceRequest[1].ApiVersion))
				Expect(nodes[0].GetKind()).To(Equal(resourceRequest[1].Kind))
			})

			It("Ignore non yaml files without throwing error as there could be readme files in package", func() {
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, key client.ObjectKey, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
						prr.Spec.Resources = map[string]string{}
						dat1, _ := os.ReadFile("samplepkg/README.md")
						prr.Spec.Resources["readme"] = string(dat1)
					})
				_, err := ps.GetNFProfiles(context.TODO(), resourceRequest, nc)
				Expect(err).To(Succeed())
			})
		})

		Context("error scenarios", func() {
			It("should error out when listing package revisions fails", func() {
				prErr := errors.New("Error out")
				mockClient.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(prErr)
				_, err := ps.GetNFProfiles(context.TODO(), resourceRequest, nc)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(prErr))
			})

			It("should error out when no package revisions found", func() {
				mockClient.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("prev1", "private-catalog", "nf-profiles", "v1", true, false),
							getPackageRevisionCR("prev4", "private-catalog", "nf-profiles-1", "v2", true, true),
							getPackageRevisionCR("prev5", "soure-repo", "nf-profiles", "v2", true, true),
						}
					})
				_, err := ps.GetNFProfiles(context.TODO(), resourceRequest, nc)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("No latest published package found"))
			})

			It("should error out when multiple package revisions found", func() {
				mockClient.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("prev1", "private-catalog", "nf-profiles", "v1", true, true),
							getPackageRevisionCR("prev4", "private-catalog", "nf-profiles", "v2", true, true),
							getPackageRevisionCR("prev5", "soure-repo", "nf-profiles", "v2", true, true),
						}
					})
				_, err := ps.GetNFProfiles(context.TODO(), resourceRequest, nc)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("More than one latest published package found"))
			})

			It("should error out when getting package resources fails", func() {
				prErr := errors.New("Error out")
				mockClient.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("prev1", "private-catalog", "nf-profiles", "v1", true, true),
						}
					})
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(prErr)

				_, err := ps.GetNFProfiles(context.TODO(), resourceRequest, nc)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(prErr))
			})

		})
	})

	Describe("testing CreateDeployPackage via Porch", func() {
		Context("valid inputs, expecting package to be created", func() {
			var prCall, prrGetCall, prrUpdateCall *gomock.Call
			var content map[string]string
			newKptfile, _ := os.ReadFile("samplepkg/Kptfile-new")
			oldKptfile, _ := os.ReadFile("samplepkg/Kptfile-old")

			BeforeEach(func() {

				content = map[string]string{}
				dat1, _ := os.ReadFile("samplepkg/private-repo.yaml")
				content["private-repo.yaml"] = string(dat1)
				dat2, _ := os.ReadFile("samplepkg/source-repo.yaml")
				content["source-repo.yaml"] = string(dat2)
				content["Kptfile"] = string(newKptfile)

				prCall = mockClient.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				prrGetCall = mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				prrUpdateCall = mockClient.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			})

			It("should create packageRevision with correct values", func() {
				prCall.Do(func(ctx context.Context, pr *porchapi.PackageRevision, arg2 ...client.CreateOption) {
					Expect(pr.ObjectMeta.Namespace).To(Equal(nc.GetNamespace()))
					Expect(pr.Spec.PackageName).To(Equal(nc.GetDeployPackageName()))
					Expect(pr.Spec.RepositoryName).To(Equal(nc.GetDeployRepoName()))
					Expect(pr.Spec.Revision).To(Equal(""))
					Expect(string(pr.Spec.WorkspaceName)).To(MatchRegexp("^v[0-9]+$"))
					Expect(pr.Spec.Tasks[0].Type).To(Equal(porchapi.TaskTypeInit))
				})
				ps.CreateDeployPackage(context.TODO(), content, nc)
			})

			It("should fetch the auto-created packageRevisionResource and update the content", func() {
				prObjectName := "repo-name-with-hashed-suffix"
				prCall.Do(func(ctx context.Context, pr *porchapi.PackageRevision, arg2 ...client.CreateOption) {
					pr.ObjectMeta.Name = prObjectName
				})
				prrGetCall.Do(func(ctx context.Context, n types.NamespacedName, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
					Expect(n.Name).To(Equal(prObjectName))
					Expect(n.Namespace).To(Equal(nc.GetNamespace()))
					prr.Spec.Resources = map[string]string{}
					prr.ObjectMeta.Namespace = nc.GetNamespace()
					prr.ObjectMeta.Name = prObjectName
				})
				prrUpdateCall.Do(func(ctx context.Context, prr *porchapi.PackageRevisionResources, arg2 ...client.UpdateOption) {
					Expect(prr.ObjectMeta.Namespace).To(Equal(nc.GetNamespace()))
					Expect(prr.ObjectMeta.Name).To(Equal(prObjectName))
					Expect(prr.Spec.Resources).To(Equal(content))
				})
				name, err := ps.CreateDeployPackage(context.TODO(), content, nc)
				Expect(name).To(Equal(prObjectName))
				Expect(err).To(Succeed())
			})

			It("should use the new kptfile when present in requested content", func() {
				prrGetCall.Do(func(ctx context.Context, n types.NamespacedName, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
					prr.Spec.Resources = map[string]string{}
					prr.Spec.Resources["Kptfile"] = string(oldKptfile)
				})
				prrUpdateCall.Do(func(ctx context.Context, prr *porchapi.PackageRevisionResources, arg2 ...client.UpdateOption) {
					Expect(prr.Spec.Resources["Kptfile"]).To(Equal(string(newKptfile)))
				})
				ps.CreateDeployPackage(context.TODO(), content, nc)
			})

			It("should use the default kptfile when kptfile absent in requested content", func() {
				prrGetCall.Do(func(ctx context.Context, n types.NamespacedName, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
					prr.Spec.Resources = map[string]string{}
					prr.Spec.Resources["Kptfile"] = string(oldKptfile)
				})
				prrUpdateCall.Do(func(ctx context.Context, prr *porchapi.PackageRevisionResources, arg2 ...client.UpdateOption) {
					Expect(prr.Spec.Resources["Kptfile"]).To(Equal(string(oldKptfile)))
				})
				delete(content, "Kptfile")
				ps.CreateDeployPackage(context.TODO(), content, nc)
			})
		})

		Context("error scenarios", func() {
			content := map[string]string{}

			It("should error out when creating package revision fails", func() {
				e := errors.New("Errored out")
				mockClient.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(e).
					Times(1)
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
				mockClient.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Times(0)

				name, err := ps.CreateDeployPackage(context.TODO(), content, nc)
				Expect(len(name)).To(Equal(0))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(e))
			})

			It("should error out when getting package revision resources fails", func() {
				e := errors.New("Errored out")
				mockClient.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(e).
					Times(1)
				mockClient.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Times(0)

				name, err := ps.CreateDeployPackage(context.TODO(), content, nc)
				Expect(len(name)).To(Equal(0))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(e))
			})

			It("should error out when updating package revision resources fails", func() {
				e := errors.New("Errored out")
				mockClient.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
				mockClient.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(e).
					Times(1)

				name, err := ps.CreateDeployPackage(context.TODO(), content, nc)
				Expect(len(name)).To(Equal(0))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(e))
			})
		})
	})

	Describe("testing DeleteDeployPackage", func() {
		pr1 := getPackageRevisionCR("prev1", "clusterName-deploy-repo", "nfDeployName-clusterName", "v1", true, false)  // older published version
		pr2 := getPackageRevisionCR("prev2", "clusterName-deploy-repo", "nfDeployName-clusterName", "v2", true, true)   // latest published version
		pr3 := getPackageRevisionCR("prev3", "clusterName-deploy-repo", "nfDeployName-clusterName", "v3", false, false) // newer version but not published
		pr4 := getPackageRevisionCR("prev4", "clusterName-deploy-repo", "nf-profiles-1", "v2", true, true)              // latest published version of similar named package
		pr5 := getPackageRevisionCR("prev5", "soure-repo", "nfDeployName-clusterName", "v2", true, true)                // latest published version but in different repo
		Context("valid inputs expecting a response", func() {
			It("should delete all the versions from deploy repo of the correct package", func() {
				mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil).Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{pr1, pr2, pr3, pr4, pr5}
					})

				mockClient.EXPECT().Delete(gomock.Any(), gomock.Eq(&pr1)).Return(nil).Times(1)
				mockClient.EXPECT().Delete(gomock.Any(), gomock.Eq(&pr2)).Return(nil).Times(1)
				mockClient.EXPECT().Delete(gomock.Any(), gomock.Eq(&pr3)).Return(nil).Times(1)
				err := ps.DeleteDeployPackage(context.Background(), nc)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("error deleting package revision", func() {
			It("should return error", func() {
				prErr := errors.New("expected error")
				mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil).Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{pr1, pr2, pr3, pr4, pr5}
					})

				mockClient.EXPECT().Delete(gomock.Any(), gomock.Eq(&pr1)).Return(prErr).Times(1)
				err := ps.DeleteDeployPackage(context.Background(), nc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(prErr.Error()))
			})
		})

		Context("error listing package revisions", func() {
			It("should return error", func() {
				prErr := errors.New("expected error")
				mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(prErr).Times(1)

				err := ps.DeleteDeployPackage(context.Background(), nc)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(prErr.Error()))
			})
		})
	})

	Describe("testing CreateNFDeployActuators via Porch", func() {
		var vendorNFKey packageservice.VendorNFKey
		var actuatorResources map[string]string
		BeforeEach(func() {
			vendorNFKey = packageservice.VendorNFKey{
				Vendor: "ABC", Version: "1.0", NFType: "Upf",
			}
			actuatorResources = map[string]string{}
			dat1, _ := os.ReadFile("samplepkg/private-repo.yaml")
			actuatorResources["actuator1/private-repo.yaml"] = string(dat1)
			dat2, _ := os.ReadFile("samplepkg/source-repo.yaml")
			actuatorResources["actuator2/source-repo.yaml"] = string(dat2)
			kptfile, _ := os.ReadFile("samplepkg/Kptfile-new")
			actuatorResources["Kptfile"] = string(kptfile)
		})

		Context("Valid responses from client for getting and creating actuator pkgs", func() {
			BeforeEach(func() {
				mockClient.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("actuatorPkg1", "private-catalog", "ABC/1.0/Upf/actuators", "v1", true, false), // older published version
							getPackageRevisionCR("actuatorPkg2", "private-catalog", "ABC/1.0/Upf/actuators", "v2", true, true),  // latest published version
						}
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{
						Namespace: nc.GetNamespace(),
						Name:      "actuatorPkg2",
					}, gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, key client.ObjectKey, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
						prr.Spec.Resources = actuatorResources
					})
			})

			It("Should create pkg when there is no existing published package in deploy repo", func() {
				mockClient.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{}
					})
				mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(1).
					Do(func(ctx context.Context, pr *porchapi.PackageRevision, arg2 ...client.CreateOption) {
						Expect(pr.ObjectMeta.Namespace).To(Equal(nc.GetNamespace()))
						Expect(pr.Spec.PackageName).To(Equal("ABC/1.0/Upf/actuators"))
						Expect(pr.Spec.RepositoryName).To(Equal(nc.GetDeployRepoName()))
						pr.Name = "newActuatorPkg"
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{Namespace: nc.GetNamespace(), Name: "newActuatorPkg"}, gomock.Any()).
					Return(nil).Times(1)
				mockClient.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil).Times(1).
					Do(func(ctx context.Context, prr *porchapi.PackageRevisionResources, arg2 ...client.UpdateOption) {
						Expect(prr.Spec.Resources).To(Equal(actuatorResources))
					})

				pName, isNew, err := ps.CreateNFDeployActuators(context.TODO(), nc, vendorNFKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(isNew).To(BeTrue())
				Expect(pName).To(Equal("newActuatorPkg"))
			})

			It("Should create pkg when there is an existing package in deploy repo with different content", func() {
				mockClient.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("oldActuatorPkg", nc.GetDeployRepoName(), "ABC/1.0/Upf/actuators", "v1", true, true),
						}
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{
						Namespace: nc.GetNamespace(),
						Name:      "oldActuatorPkg",
					}, gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, key client.ObjectKey, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
						prr.Spec.Resources = map[string]string{
							"actuator1/private-repo.yaml": actuatorResources["actuator1/private-repo.yaml"],
						}
					})
				mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(1).
					Do(func(ctx context.Context, pr *porchapi.PackageRevision, arg2 ...client.CreateOption) {
						Expect(pr.ObjectMeta.Namespace).To(Equal(nc.GetNamespace()))
						Expect(pr.Spec.PackageName).To(Equal("ABC/1.0/Upf/actuators"))
						Expect(pr.Spec.RepositoryName).To(Equal(nc.GetDeployRepoName()))
						pr.Name = "newActuatorPkg"
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{Namespace: nc.GetNamespace(), Name: "newActuatorPkg"}, gomock.Any()).
					Return(nil).Times(1)
				mockClient.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil).Times(1).
					Do(func(ctx context.Context, prr *porchapi.PackageRevisionResources, arg2 ...client.UpdateOption) {
						Expect(prr.Spec.Resources).To(Equal(actuatorResources))
					})

				pName, isNew, err := ps.CreateNFDeployActuators(context.TODO(), nc, vendorNFKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(isNew).To(BeTrue())
				Expect(pName).To(Equal("newActuatorPkg"))
			})

			It("Should not create pkg when there is an existing package in deploy repo with same content", func() {
				mockClient.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("oldActuatorPkg", nc.GetDeployRepoName(), "ABC/1.0/Upf/actuators", "v1", true, true),
						}
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{
						Namespace: nc.GetNamespace(),
						Name:      "oldActuatorPkg",
					}, gomock.Any()).
					Return(nil).
					Times(1).
					Do(func(ctx context.Context, key client.ObjectKey, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
						prr.Spec.Resources = actuatorResources
					})
				mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(0)
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(0)
				mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).Times(0)

				pName, isNew, err := ps.CreateNFDeployActuators(context.TODO(), nc, vendorNFKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(isNew).To(BeFalse())
				Expect(pName).To(Equal("oldActuatorPkg"))
			})
		})

		Context("error scenarios", func() {
			It("Should fail when error while fetching actuator resources", func() {
				cause := errors.New("error when fetching actuator pkg")
				mockClient.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(cause).
					Times(1)
				mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(0)
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(0)
				mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).Times(0)
				pName, isNew, err := ps.CreateNFDeployActuators(context.TODO(), nc, vendorNFKey)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(cause))
				Expect(isNew).To(BeFalse())
				Expect(pName).To(Equal(""))
			})

			It("Should fail when error while fetching existing actuator resources in deploy", func() {
				cause := errors.New("error when fetching existing actuator pkg")
				mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil).Times(2).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("actuatorPkg2", "private-catalog", "ABC/1.0/Upf/actuators", "v2", true, true),
							getPackageRevisionCR("oldActuatorPkg", nc.GetDeployRepoName(), "ABC/1.0/Upf/actuators", "v1", true, true),
						}
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{Namespace: nc.GetNamespace(), Name: "actuatorPkg2"}, gomock.Any()).
					Return(nil).Times(1).
					Do(func(ctx context.Context, key client.ObjectKey, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
						prr.Spec.Resources = actuatorResources
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{Namespace: nc.GetNamespace(), Name: "oldActuatorPkg"}, gomock.Any()).
					Return(cause).Times(1)
				mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(0)
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(0)
				mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).Times(0)
				pName, isNew, err := ps.CreateNFDeployActuators(context.TODO(), nc, vendorNFKey)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(cause))
				Expect(isNew).To(BeFalse())
				Expect(pName).To(Equal(""))
			})

			It("Should fail when error while creating actuator resources", func() {
				cause := errors.New("error when creating actuator pkg")
				mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil).Times(2).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("actuatorPkg2", "private-catalog", "ABC/1.0/Upf/actuators", "v2", true, true),
							getPackageRevisionCR("oldActuatorPkg", nc.GetDeployRepoName(), "ABC/1.0/Upf/actuators", "v1", true, true),
						}
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{Namespace: nc.GetNamespace(), Name: "actuatorPkg2"}, gomock.Any()).
					Return(nil).Times(1).
					Do(func(ctx context.Context, key client.ObjectKey, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
						prr.Spec.Resources = actuatorResources
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{Namespace: nc.GetNamespace(), Name: "oldActuatorPkg"}, gomock.Any()).
					Return(nil).Times(1)
				mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(cause).Times(1)
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(0)
				mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).Times(0)
				pName, isNew, err := ps.CreateNFDeployActuators(context.TODO(), nc, vendorNFKey)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(cause))
				Expect(isNew).To(BeFalse())
				Expect(pName).To(Equal(""))
			})
		})
	})

	Describe("testing GetVendorExtensionPackage via Porch", func() {
		var vendorNFKey packageservice.VendorNFKey
		var extnResources map[string]string
		var expectedResources []string
		BeforeEach(func() {
			vendorNFKey = packageservice.VendorNFKey{
				Vendor: "ABC", Version: "1.0", NFType: "Upf",
			}
			extnResources = map[string]string{}
			dat1, _ := os.ReadFile("samplepkg/private-repo.yaml")
			extnResources["dummy-extn.yaml"] = string(dat1)
			kptfile, _ := os.ReadFile("samplepkg/Kptfile-new")
			extnResources["Kptfile"] = string(kptfile)

			rNodes, _ := util.ParseStringToYamlNode(string(dat1))
			expectedResources = []string{}
			for _, rNode := range rNodes {
				expectedResources = append(expectedResources, rNode.MustString())
			}
		})
		Context("Valid responses from client for getting extension pkg", func() {
			It("Should return extn resources in different strings when present", func() {
				mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil).Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("extensionPkg1", "private-catalog", "ABC/1.0/Upf/extension", "v1", true, true),
						}
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{Namespace: nc.GetNamespace(), Name: "extensionPkg1"}, gomock.Any()).
					Return(nil).Times(1).
					Do(func(ctx context.Context, key client.ObjectKey, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
						prr.Spec.Resources = extnResources
					})
				actualResources, err := ps.GetVendorExtensionPackage(context.TODO(), nc, vendorNFKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualResources).To(ConsistOf(expectedResources))
			})

			It("Should return empty extn resources when package missing in porch", func() {
				mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil).Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("extensionPkg1", "private-catalog", "ABC/1.0/Upf/extension", "v1", true, false),
						}
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{Namespace: nc.GetNamespace(), Name: "extensionPkg1"}, gomock.Any()).
					Return(nil).Times(0)
				actualResources, err := ps.GetVendorExtensionPackage(context.TODO(), nc, vendorNFKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(actualResources)).To(Equal(0))
			})

			It("Should ignore err when package has invalid object in yaml and invalid yamls", func() {
				dat1, _ := os.ReadFile("samplepkg/invalidobject.yaml")
				extnResources["invalidobject.yaml"] = string(dat1)
				extnResources["invalidfile.yaml"] = `
				This will fail on parsing.
				`
				mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil).Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("extensionPkg1", "private-catalog", "ABC/1.0/Upf/extension", "v1", true, true),
						}
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{Namespace: nc.GetNamespace(), Name: "extensionPkg1"}, gomock.Any()).
					Return(nil).Times(1).
					Do(func(ctx context.Context, key client.ObjectKey, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
						prr.Spec.Resources = extnResources
					})
				actualResources, err := ps.GetVendorExtensionPackage(context.TODO(), nc, vendorNFKey)
				Expect(actualResources).To(ConsistOf(expectedResources))
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should succeed when namespace not present", func() {
				resources := map[string]string{}
				dat1, _ := os.ReadFile("samplepkg/repo-withoutns.yaml")
				resources["repo-withoutns.yaml"] = string(dat1)
				rNodes, _ := util.ParseStringToYamlNode(string(dat1))
				expectedResources := []string{rNodes[0].MustString()}

				mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil).Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("extensionPkg1", "private-catalog", "ABC/1.0/Upf/extension", "v1", true, true),
						}
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{Namespace: nc.GetNamespace(), Name: "extensionPkg1"}, gomock.Any()).
					Return(nil).Times(1).
					Do(func(ctx context.Context, key client.ObjectKey, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
						prr.Spec.Resources = resources
					})
				actualResources, err := ps.GetVendorExtensionPackage(context.TODO(), nc, vendorNFKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualResources).To(ConsistOf(expectedResources))
			})
		})

		Context("Invalid responses from client for getting extension pkg", func() {
			It("Should return error when error fetching package revision", func() {
				cause := errors.New("error fetching package revision")
				mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(cause).Times(1)
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).Times(0)
				actualResources, err := ps.GetVendorExtensionPackage(context.TODO(), nc, vendorNFKey)
				Expect(actualResources).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(cause))
			})

			It("Should return err when no valid object found in extension package", func() {
				invalidResources := map[string]string{}
				dat1, _ := os.ReadFile("samplepkg/invalidobject.yaml")
				invalidResources["invalidobject.yaml"] = string(dat1)
				invalidResources["invalidfile.yaml"] = `
				This will fail on parsing.
				`
				mockClient.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil).Times(1).
					Do(func(ctx context.Context, prList *porchapi.PackageRevisionList, arg2 ...client.ListOption) {
						prList.Items = []porchapi.PackageRevision{
							getPackageRevisionCR("extensionPkg1", "private-catalog", "ABC/1.0/Upf/extension", "v1", true, true),
						}
					})
				mockClient.EXPECT().
					Get(gomock.Any(), client.ObjectKey{Namespace: nc.GetNamespace(), Name: "extensionPkg1"}, gomock.Any()).
					Return(nil).Times(1).
					Do(func(ctx context.Context, key client.ObjectKey, prr *porchapi.PackageRevisionResources, opts ...client.GetOption) {
						prr.Spec.Resources = invalidResources
					})
				actualResources, err := ps.GetVendorExtensionPackage(context.TODO(), nc, vendorNFKey)
				Expect(actualResources).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("No valid vendor extension k8s object found"))
			})
		})
	})
})
