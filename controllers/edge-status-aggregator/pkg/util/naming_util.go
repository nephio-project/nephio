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

package util

import (
	"errors"
	"fmt"
)

/**
The logic should not be modified without verifying the end to end flow.
In Nephio, we are relying on naming convention in multiple places like name of  repos - source and deploy, the
Porch package names etc. To make sure the logic continues to work, always ensure the Blueprints, User guide and
the logic in this file is in sync.
*/

const (
	namespace                  = "nephio-user"
	nfProfilePackageName       = "nf-profiles"
	nfProfileRepoName          = "private-catalog"
	vendorNFManifestsRepoName  = "private-catalog"
	deployPackageNameFormat    = "%s-%s"              // nfDeployName-clusterName
	deployRepoNameFormat       = "%s-deploy-repo"     // 'clusterName'-deploy-repo
	actuatorPackageNameFormat  = "%s/%s/%s/actuators" // vendor/version/nfType/actuators
	extensionPackageNameFormat = "%s/%s/%s/extension" // vendor/version/nfType/extension
)

type NamingContext struct {
	// the members are private as we only want the values to be exposed via functions
	// to allow defining naming conventions as we need.
	clusterName  string
	nfDeployName string

	// ... we can add more fields when we need to depend on it for naming
}

// returns a new NamingContext object with the given cluster name
func NewNamingContext(cluster string, nfDeploy string) (NamingContext, error) {
	if cluster == "" || nfDeploy == "" {
		return NamingContext{}, errors.New(fmt.Sprintf("Invalid input [cluster: %s, nfDeploy: %s]. Inputs cannot be empty", cluster, nfDeploy))
	}
	return NamingContext{
		clusterName:  cluster,
		nfDeployName: nfDeploy,
	}, nil
}

// returns the namespace for the current NamingContext
func (c *NamingContext) GetNamespace() string {
	return namespace
}

// returns the package name that stores the NF profiles for the current NamingContext
func (c *NamingContext) GetNFProfilePackageName() string {
	return nfProfilePackageName
}

// returns the repository that stores the NF profiles for the current NamingContext
func (c *NamingContext) GetNFProfileRepoName() string {
	return nfProfileRepoName
}

// returns the repository that stores the Vendor NF related manifest like actuators
// and extension package for the current NamingContext
func (c *NamingContext) GetVendorNFManifestsRepoName() string {
	return vendorNFManifestsRepoName
}

// returns name for the new deployment package for the current NamingContext
func (c *NamingContext) GetDeployPackageName() string {
	return fmt.Sprintf(deployPackageNameFormat, c.nfDeployName, c.clusterName)
}

// returns the package name of the Vendor NF actuators manifest for the current NamingContext
func (c *NamingContext) GetNFDeployActuatorPackageName(vendor string, version string, nfType string) string {
	return fmt.Sprintf(actuatorPackageNameFormat, vendor, version, nfType)
}

// returns the package name of the Vendor NF extension manifest for the current NamingContext
func (c *NamingContext) GetVendorExtensionPackageName(vendor string, version string, nfType string) string {
	return fmt.Sprintf(extensionPackageNameFormat, vendor, version, nfType)
}

// returns repository name for the new deployment package for the current NamingContext
func (c *NamingContext) GetDeployRepoName() string {
	return fmt.Sprintf(deployRepoNameFormat, c.clusterName)
}

// GetNfDeployName returns the nfDeploy name for the current NamingContext
func (c *NamingContext) GetNfDeployName() string {
	return c.nfDeployName
}
