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

package bootstrappackages

import (
	"context"
	"fmt"
    "os"
	porchconfig "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type fakeClient struct {
	objects []client.Object
	client.Client
  testDataPath string
}

var _ client.Client = &fakeClient{}

func (f *fakeClient) List(_ context.Context, obj client.ObjectList, _ ...client.ListOption) error {
  data, err := os.ReadFile(f.testDataPath)
  if err != nil {
      panic(err)
  }

	switch v := obj.(type) {
	case *porchconfig.RepositoryList:
		err = yaml.Unmarshal([]byte(data), v)
		for _, o := range v.Items {
			f.objects = append(f.objects, o.DeepCopy())
		}
	case *corev1.SecretList:
		err = yaml.Unmarshal([]byte(data), v)
		for _, o := range v.Items {
			f.objects = append(f.objects, o.DeepCopy())
		}
  default:
		return fmt.Errorf("unsupported type")
	}
	return err
}
