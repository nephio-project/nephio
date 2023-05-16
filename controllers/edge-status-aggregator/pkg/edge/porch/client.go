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

package porch

import (
	"context"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/nephio-project/edge-status-aggregator/edge/approve"
	"github.com/nephio-project/edge-status-aggregator/packageservice"
	"github.com/nephio-project/edge-status-aggregator/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: Integrate auto approve workflow in NFDeployController's PackageService
type porchClient struct {
	logger logr.Logger

	*packageservice.PorchPackageService
	approve.K8sRestClient
}

func (c *porchClient) getPackageRevision(ctx context.Context, name,
	namespace string) (*porchapi.PackageRevision, error) {
	pr := &porchapi.PackageRevision{}
	if err := c.Client.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (c *porchClient) updatePackageRevision(ctx context.Context,
	pr *porchapi.PackageRevision) error {
	return c.Client.Update(ctx, pr)
}

// ApplyPackage utilizes NFDeployController's PackageService to create a package
// in Draft state and then moves it from Draft to Proposed to Published state
func (c *porchClient) ApplyPackage(ctx context.Context,
	contents map[string]string, packageName, clusterName string) error {
	logger := c.logger.WithName("ApplyPackage").WithValues(
		"packageName", packageName, "clusterName", clusterName)
	debugLogger := logger.V(1)

	debugLogger.Info("creating new naming context")
	nc, err := util.NewNamingContext(clusterName, packageName)
	if err != nil {
		logger.Error(err, "error in creating naming context")
		return err
	}

	debugLogger.Info("creating porch package")
	packageRevisionName, err := c.CreateDeployPackage(ctx, contents, nc)
	if err != nil {
		logger.Error(err, "error in creating porch package")
		return err
	}

	debugLogger.Info("fetching packageRevision",
		"packageRevisionName", packageRevisionName)
	pr, err := c.getPackageRevision(ctx, packageRevisionName, nc.GetNamespace())
	if err != nil {
		logger.Error(err, "error in fetching packageRevision",
			"packageRevisionName", packageRevisionName)
		return err
	}

	debugLogger.Info("proposing packageRevision")
	pr.Spec.Lifecycle = porchapi.PackageRevisionLifecycleProposed
	if err := c.updatePackageRevision(ctx, pr); err != nil {
		logger.Error(err, "error in proposing packageRevision",
			"packageRevisionName", pr.Name)
		return err
	}

	debugLogger.Info("fetching packageRevision",
		"packageRevisionName", packageRevisionName)
	pr, err = c.getPackageRevision(ctx, packageRevisionName, nc.GetNamespace())
	if err != nil {
		logger.Error(err, "error in fetching packageRevision",
			"packageRevisionName", packageRevisionName)
		return err
	}

	debugLogger.Info("publishing packageRevision",
		"packageRevisionName", pr.Name)
	pr.Spec.Lifecycle = porchapi.PackageRevisionLifecyclePublished
	if err := c.ApprovePackageRevision(ctx, pr); err != nil {
		logger.Error(err, "error in publishing packageRevision",
			"packageRevisionName", pr.Name)
		return err
	}

	return nil
}

type Client interface {
	// ApplyPackage create/updates a porch package and "auto-approves" it
	ApplyPackage(ctx context.Context, contents map[string]string,
		packageName, clusterName string) error
}

func NewClient(logger logr.Logger, service *packageservice.PorchPackageService,
	restClient approve.K8sRestClient) Client {
	return &porchClient{logger, service,
		restClient}
}
