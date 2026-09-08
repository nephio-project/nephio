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
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	mockReader "github.com/nephio-project/nephio/controllers/pkg/mocks/external/reader"
	porchapi "github.com/nephio-project/porch/api/porch/v1alpha1"
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

	// withDelay builds a PackageRevision created at the given time with the
	// delay annotation set.
	withDelay := func(created time.Time, delay string) porchapi.PackageRevision {
		return porchapi.PackageRevision{
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: metav1.Time{Time: created},
				Annotations:       map[string]string{"approval.nephio.org/delay": delay},
			},
		}
	}

	testCases := map[string]struct {
		pr            porchapi.PackageRevision
		expectedDelay time.Duration
		expectedError bool
	}{
		"no annotation": {
			pr:            porchapi.PackageRevision{},
			expectedDelay: 0,
		},
		"unparseable annotation": {
			pr:            withDelay(now, "foo"),
			expectedError: true,
		},
		"negative annotation": {
			pr:            withDelay(now, "-5s"),
			expectedError: true,
		},
		"zero delay is valid and returns zero": {
			pr:            withDelay(now, "0s"),
			expectedDelay: 0,
		},
		"partway through the delay returns the remaining time": {
			pr:            withDelay(now.Add(-50*time.Minute), "1h"),
			expectedDelay: 10 * time.Minute,
		},
		"exactly at the delay boundary returns zero": {
			pr:            withDelay(now.Add(-time.Hour), "1h"),
			expectedDelay: 0,
		},
		"past the delay returns zero": {
			pr:            withDelay(now.Add(-2*time.Hour), "1h"),
			expectedDelay: 0,
		},
		"future creation timestamp is well defined": {
			// A creation timestamp ahead of now (e.g. clock skew) must not
			// panic; elapsed time is negative, so the full delay plus the skew
			// remains.
			pr:            withDelay(now.Add(10*time.Minute), "1h"),
			expectedDelay: 70 * time.Minute,
		},
		"the largest delay does not overflow into an immediate approval": {
			// Subtracting a negative elapsed time from the largest duration
			// wraps to MinInt64, which reads as "already elapsed" and approves
			// at once. Time.Sub saturates, so the answer is the maximum.
			pr:            withDelay(now.Add(time.Nanosecond), time.Duration(math.MaxInt64).String()),
			expectedDelay: time.Duration(math.MaxInt64),
		},
		"missing creation timestamp is refused rather than approved": {
			pr: porchapi.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"approval.nephio.org/delay": "1h"},
				},
			},
			expectedError: true,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			actualDelay, err := manageDelayAt(&tc.pr, now)
			if tc.expectedError {
				require.Error(t, err)
				require.Zero(t, actualDelay)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedDelay, actualDelay)
		})
	}

	// A zero delay says the same thing its absence says, so it answers the
	// same way even on an object that has never been persisted and has no
	// creation timestamp to measure from. The README calls the two equivalent.
	for name, pr := range map[string]porchapi.PackageRevision{
		"annotation absent": {},
		"annotation 0s": {ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
			DelayAnnotationName: "0s"}}},
	} {
		d, err := manageDelayAt(&pr, now)
		require.NoError(t, err, name)
		require.Zero(t, d, name)
	}

	// manageDelay must read the wall clock. Bracketing the call pins it to a
	// time between before and after rather than to a tolerance, so a suspended
	// runner cannot make this flake and a frozen clock cannot slip through.
	created := time.Now()
	nowPR := withDelay(created, "1h")
	before := time.Now()
	d, err := manageDelay(&nowPR)
	after := time.Now()
	require.NoError(t, err)
	deadline := created.Add(time.Hour)
	require.LessOrEqual(t, d, deadline.Sub(before))
	require.GreaterOrEqual(t, d, deadline.Sub(after))
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
