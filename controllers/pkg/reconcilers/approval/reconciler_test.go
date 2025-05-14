// Copyright 2023 The Nephio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package approval

import (
	context "context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	mockReader "github.com/nephio-project/nephio/controllers/pkg/mocks/external/reader"
	porchapi "github.com/nephio-project/porch/v4/api/porch/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestShouldProcess(t *testing.T) {
	testCases := map[string]struct {
		pr             porchapi.PackageRevision
		expectedPolicy string
		expectedShould bool
	}{
		"draft with no annotation": {
			pr:             porchapi.PackageRevision{},
			expectedPolicy: "",
			expectedShould: false,
		},
		"draft with initial policy annotation": {
			pr: porchapi.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"approval.nephio.org/policy": "initial",
					},
				},
			},
			expectedPolicy: "initial",
			expectedShould: true,
		},
		"draft with always policy annotation": {
			pr: porchapi.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"approval.nephio.org/policy": "always",
					},
				},
			},
			expectedPolicy: "always",
			expectedShould: true,
		},
		"draft with no policy annotation, but delay annotation": {
			pr: porchapi.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"approval.nephio.org/delay": "20s",
					},
				},
			},
			expectedPolicy: "",
			expectedShould: false,
		},
		"published with policy annotation": {
			pr: porchapi.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"approval.nephio.org/policy": "initial",
					},
				},
				Spec: porchapi.PackageRevisionSpec{
					Lifecycle: "Published",
				},
			},
			expectedPolicy: "initial",
			expectedShould: false,
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			actualPolicy, actualShould := shouldProcess(&tc.pr)
			require.Equal(t, tc.expectedPolicy, actualPolicy)
			require.Equal(t, tc.expectedShould, actualShould)
		})
	}
}

func TestManageDelay(t *testing.T) {
	now := time.Now()
	testCases := map[string]struct {
		pr              porchapi.PackageRevision
		expectedRequeue bool
		expectedError   bool
	}{
		"no annotation": {
			pr:              porchapi.PackageRevision{},
			expectedRequeue: false,
			expectedError:   false,
		},
		"unparseable annotation": {
			pr: porchapi.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"approval.nephio.org/delay": "foo",
					},
				},
			},
			expectedRequeue: false,
			expectedError:   true,
		},
		"negative annotation": {
			pr: porchapi.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"approval.nephio.org/delay": "-5s",
					},
				},
			},
			expectedRequeue: false,
			expectedError:   true,
		},
		"not old enough": {
			pr: porchapi.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: now},
					Annotations: map[string]string{
						"approval.nephio.org/delay": "1h",
					},
				},
			},
			expectedRequeue: true,
			expectedError:   false,
		},
		"old enough": {
			pr: porchapi.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: now.AddDate(-1, 0, 0)},
					Annotations: map[string]string{
						"approval.nephio.org/delay": "1h",
					},
				},
			},
			expectedRequeue: false,
			expectedError:   false,
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			actualRequeue, actualError := manageDelay(&tc.pr)
			require.Equal(t, tc.expectedRequeue, actualRequeue > 0)
			require.Equal(t, tc.expectedError, actualError != nil)
		})
	}
}

func TestPolicyInitial(t *testing.T) {

	testCases := map[string]struct {
		pr              porchapi.PackageRevision
		prl             *porchapi.PackageRevisionList
		expectedApprove bool
		expectedError   error
		mockReturnErr   error
	}{
		"Draft with proposed lifecycle": {
			pr: porchapi.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"approval.nephio.org/policy": "initial",
					},
				},
			},
			prl: &porchapi.PackageRevisionList{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "Blah",
					Kind:       "Blah",
				},
				Items: []porchapi.PackageRevision{
					{
						Spec: porchapi.PackageRevisionSpec{
							Lifecycle: porchapi.PackageRevisionLifecycleProposed,
						},
					},
				},
			},
			expectedApprove: true,
			expectedError:   nil,
			mockReturnErr:   nil,
		},
		"Draft with existing version": {
			pr: porchapi.PackageRevision{
				Spec: porchapi.PackageRevisionSpec{
					RepositoryName: "MyRepo",
					PackageName:    "MyPackage",
				},
			},
			prl: &porchapi.PackageRevisionList{
				Items: []porchapi.PackageRevision{
					{
						Spec: porchapi.PackageRevisionSpec{
							Lifecycle:      porchapi.PackageRevisionLifecyclePublished,
							RepositoryName: "MyRepo",
							PackageName:    "MyPackage",
						},
					},
				},
			},
			expectedApprove: false,
			expectedError:   nil,
			mockReturnErr:   nil,
		},
		"runtime client list failure": {
			pr:              porchapi.PackageRevision{},
			prl:             &porchapi.PackageRevisionList{},
			expectedApprove: false,
			expectedError:   fmt.Errorf("Failed to list items"),
			mockReturnErr:   fmt.Errorf("Failed to list items"),
		},
	}
	for tn, tc := range testCases {
		// Create a new instance of the mock object
		readerMock := new(mockReader.MockReader)
		readerMock.On("List", context.TODO(), mock.AnythingOfType("*v1alpha1.PackageRevisionList")).Return(tc.mockReturnErr).Run(func(args mock.Arguments) {
			packRevList := args.Get(1).(*porchapi.PackageRevisionList)
			*packRevList = *tc.prl // tc.prl is what r.Get will store in 2nd Argument
		})
		// Create an instance of the component under test
		r := reconciler{apiReader: readerMock}
		t.Run(tn, func(t *testing.T) {
			actualApproval, actualError := r.policyInitial(context.TODO(), &tc.pr)
			require.Equal(t, tc.expectedApprove, actualApproval)
			require.Equal(t, tc.expectedError, actualError)
		})
	}
}
