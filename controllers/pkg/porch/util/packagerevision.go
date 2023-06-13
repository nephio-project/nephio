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
	"crypto/sha1"
	"encoding/hex"

	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func PackageRevisionResourcesHash(prr *porchv1alpha1.PackageRevisionResources) (string, error) {
	b, err := yaml.Marshal(prr.Spec.Resources)
	if err != nil {
		return "", err
	}
	hash := sha1.Sum(b)
	return hex.EncodeToString(hash[:]), nil
}
