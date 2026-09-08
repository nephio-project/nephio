// Copyright 2026 The Nephio Authors
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
	"testing"
	"time"

	porchv1alpha1 "github.com/nephio-project/porch/api/porch/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FuzzManageDelay pins the invariants rather than specific outputs: a requeue
// is never negative, and zero means the deadline has actually passed.
func FuzzManageDelay(f *testing.F) {
	f.Add(int64(0), "1h")
	f.Add(int64(-3600e9), "1h")
	f.Add(int64(1<<62), "2562047h47m16.854775807s")
	f.Add(int64(-(1 << 62)), "2562047h47m16.854775807s")

	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	f.Fuzz(func(t *testing.T, offsetNanos int64, delay string) {
		if _, err := time.ParseDuration(delay); err != nil {
			t.Skip()
		}
		created := now.Add(time.Duration(offsetNanos))
		if created.IsZero() {
			t.Skip()
		}
		pr := &porchv1alpha1.PackageRevision{ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.Time{Time: created},
			Annotations:       map[string]string{DelayAnnotationName: delay},
		}}
		got, err := manageDelayAt(pr, now)
		if err != nil {
			return
		}
		if got < 0 {
			t.Fatalf("negative requeue %v (offset=%d delay=%q)", got, offsetNanos, delay)
		}
		deadline := created.Add(mustParse(delay))
		if got == 0 && deadline.After(now) {
			t.Fatalf("approved early: deadline %v is after now %v (offset=%d delay=%q)",
				deadline, now, offsetNanos, delay)
		}
		if got > 0 && got != deadline.Sub(now) {
			t.Fatalf("requeued %v, want %v remaining (offset=%d delay=%q)",
				got, deadline.Sub(now), offsetNanos, delay)
		}
	})
}

func mustParse(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}
