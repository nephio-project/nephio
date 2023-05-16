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

package approve

import (
	"context"
	"fmt"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

// K8sRestClient is a wrapper over rest.Interface
type K8sRestClient interface {
	// ApprovePackageRevision uses rest.Interface to move a packageRevision from
	// Proposed to Published state
	ApprovePackageRevision(ctx context.Context, pr *porchapi.PackageRevision,
	) error
}

type client struct {
	rest.Interface
	scheme *runtime.Scheme
}

func (c *client) ApprovePackageRevision(ctx context.Context,
	pr *porchapi.PackageRevision) error {
	if err := c.Put().
		Namespace(pr.Namespace).
		Resource("packagerevisions").
		Name(pr.Name).
		SubResource("approval").
		VersionedParams(&metav1.UpdateOptions{}, runtime.NewParameterCodec(c.scheme)).
		Body(pr).
		Do(ctx).
		Into(&porchapi.PackageRevision{}); err != nil {
		return fmt.Errorf("error returned by the rest client: %v", err.Error())
	}
	return nil
}

func NewK8sRestClient(config *rest.Config, scheme *runtime.Scheme) (
	K8sRestClient, error) {
	codecs := serializer.NewCodecFactory(scheme)
	gv := porchapi.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = codecs.WithoutConversion()
	restClient, err := rest.RESTClientFor(config)
	return &client{Interface: restClient, scheme: scheme}, err
}
