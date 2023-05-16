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

package utils

import (
	"context"
	"errors"

	ps "github.com/nephio-project/edge-status-aggregator/packageservice"
	"github.com/nephio-project/edge-status-aggregator/util"
)

type FakePackageService struct {
}

func (fakeps *FakePackageService) GetNFProfiles(ctx context.Context,
	req []ps.GetResourceRequest, nc util.NamingContext) (map[int][]string, error) {

	// implement this method when required
	return nil, nil
}

func (fakeps *FakePackageService) CreateDeployPackage(ctx context.Context,
	contents map[string]string, nc util.NamingContext) (string, error) {

	// implement this method when required
	return "", nil
}

func (fakeps *FakePackageService) DeleteDeployPackage(ctx context.Context,
	nc util.NamingContext) error {

	if nc.GetNfDeployName() == "nfdeploy-deletion-error" {
		return errors.New("error from porch")
	}

	return nil
}

func (fakeps *FakePackageService) CreateNFDeployActuators(ctx context.Context,
	nc util.NamingContext,
	key ps.VendorNFKey) (string, bool, error) {
	// implement this method when required
	return "", false, nil
}

func (fakeps *FakePackageService) GetVendorExtensionPackage(ctx context.Context,
	nc util.NamingContext,
	key ps.VendorNFKey) ([]string, error) {
	// implement this method when required
	return []string{}, nil
}

var _ ps.PackageServiceInterface = &FakePackageService{}
