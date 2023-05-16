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

package packageservice

import (
	"context"

	util "github.com/nephio-project/edge-status-aggregator/util"
)

type PackageServiceInterface interface {
	// GetNFProfiles fetches the requested Nephio CRs (like UpfType, SmfType) and
	// different profile CRs (like CapacityProfile) and returns these
	// resources in a map with key same as the ID in request and value will be the
	// actual CRs in string format
	GetNFProfiles(ctx context.Context, req []GetResourceRequest, nc util.NamingContext) (map[int][]string, error)

	// CreateDeployPackage creates a package in the deploy repo and returns the package k8s resource name
	CreateDeployPackage(ctx context.Context, contents map[string]string, nc util.NamingContext) (string, error)

	// DeleteDeployPackage deletes packages from the deploy repo
	DeleteDeployPackage(ctx context.Context, nc util.NamingContext) error

	// CreateNFDeployActuators creates the NFDeployActuators in the deploy repo and returns
	// 1. the package k8s resource name.
	// 2. True if the package was newly created or updated else false if the package
	//    was already present with same content.
	// 3. Error if any occurred else nil.
	CreateNFDeployActuators(ctx context.Context, nc util.NamingContext, key VendorNFKey) (string, bool, error)

	// GetVendorExtensionPackage returns the list of k8s object yamls as string in the extension package.
	// Each string represent a single k8s object.
	GetVendorExtensionPackage(ctx context.Context, nc util.NamingContext, key VendorNFKey) ([]string, error)
}

// GetResourceRequest is used as the input for fetching NF Profiles
type GetResourceRequest struct {
	// ID uniquely identifies the request
	ID int
	// ApiVersion is the ApiVersion of the requested resource
	ApiVersion string
	// Kind is the Kind of the requested resource
	Kind string
	// Name is the Name of the requested resource, this is the optional field
	Name string
}

// VendorNFKey is used to fetch and create the actuators or extension packages.
// specific to that vendor's NF type.
// The fields names should exactly match with what is present
// in the private catalogue of the customer.
type VendorNFKey struct {
	// The identifier for representing the vendor in our manifests for e.g. Casa.
	Vendor string

	// The version of the vendor's NF type for e.g. 1.0
	Version string

	// NF Type like Upf, Smf etc.
	NFType string
}
