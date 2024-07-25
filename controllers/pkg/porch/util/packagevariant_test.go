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

package util

import (
	"context"
	"reflect"
	"testing"

	porchapi "github.com/nephio-project/porch/api/porch/v1alpha1"
	pvapi "github.com/nephio-project/porch/controllers/packagevariants/api/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type fakeClient struct {
	object client.Object
	client.Client
}

func (f *fakeClient) Get(_ context.Context, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	mockPV := `apiVersion: config.porch.kpt.dev/v1alpha1
kind: PackageVariant
metadata:
  name: wc-argocd-argocd-cluster
  namespace: default
status:
  conditions:
  - message: successfully ensured downstream package variant
    reason: NoErrors
    status: "True"
    type: Ready
`
	yaml.Unmarshal([]byte(mockPV), obj)
	f.object = obj
	return nil
}

func TestPackageRevisionIsReady(t *testing.T) {
	tr := true
	cases := map[string]struct {
		mockClient      *fakeClient
		packageRevision porchapi.PackageRevision
		expectedOwned   bool
		expectedError   error
	}{
		"Owned": {
			mockClient: &fakeClient{},
			packageRevision: porchapi.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "config.porch.kpt.dev/v1alpha1",
							Controller: &tr,
							Kind:       reflect.TypeFor[pvapi.PackageVariant]().Name(),
							Name:       "wc-argocd-argocd-cluster",
						},
					},
				},
			},
			expectedOwned: true,
			expectedError: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actualOwned, actualError := PackageVariantReady(context.Background(), &tc.packageRevision, tc.mockClient)
			require.Equal(t, tc.expectedOwned, actualOwned)
			require.Equal(t, tc.expectedError, actualError)
		})
	}
}
