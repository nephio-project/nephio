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
	"errors"
	"fmt"
	"reflect"
	"time"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/go-logr/logr"
	util "github.com/nephio-project/edge-status-aggregator/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	KptfileName string = "Kptfile"
)

// PorchPackageService implements PackageServiceInterface.
type PorchPackageService struct {
	// Rest client to intract with Porch server.
	Client client.Client
	// PorchPackageService specific logger.
	Log logr.Logger
}

// fetches the requested Nephio profiles CRs (like UpfType, SmfType, CapacityProfile) and returns these
// resources as per the requestID in a map.
func (ps *PorchPackageService) GetNFProfiles(ctx context.Context, req []GetResourceRequest, nc util.NamingContext) (map[int][]string, error) {
	ps.Log.Info(fmt.Sprintf("Fetching latest package: %s from repo: %s", nc.GetNFProfilePackageName(), nc.GetNFProfileRepoName()))
	pr, prr, _, err := ps.getLatestPackage(ctx, nc.GetNamespace(), nc.GetNFProfilePackageName(), nc.GetNFProfileRepoName())
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch latest package: %s : %w", nc.GetNFProfilePackageName(), err)
	}
	ps.Log.Info(fmt.Sprintf("Successfully fetched package %s with revision %s", pr.ObjectMeta.Name, pr.Spec.Revision))

	allNodes := ps.convertResourcesToYamlNodes(prr)
	// initialize result map
	res := map[int][]string{}
	for _, r := range req {
		res[r.ID] = []string{}
	}

	for _, r := range req {
		ps.Log.Info(fmt.Sprintf("Finding yamls matching the request %#v", r))
		mList, err := util.GetMatchingYamlNodes(allNodes, r.ApiVersion, r.Kind, r.Name)
		if err != nil {
			return nil, fmt.Errorf("Failed to match fetched resources to the request: %w", err)
		}
		ps.Log.Info(fmt.Sprintf("Found %d yamls matching the request %#v", len(mList), r))
		for _, m := range mList {
			res[r.ID] = append(res[r.ID], m.MustString())
		}
	}

	return res, nil
}

// creates the package in the relevant deploy repository and returns the new package name
func (ps *PorchPackageService) CreateDeployPackage(ctx context.Context, contents map[string]string, nc util.NamingContext) (string, error) {
	// create the PackageRevision resource for the request
	deployRepo := nc.GetDeployRepoName()
	pName := nc.GetDeployPackageName()
	ps.Log.Info(fmt.Sprintf("Creating package: %s in deploy repo: %s in namespace: %s", pName, deployRepo, nc.GetNamespace()))
	pr, _, err := ps.createPackage(ctx, nc.GetNamespace(), pName, deployRepo, contents)
	if err != nil {
		return "", fmt.Errorf("Failed to create package: %s in deploy repo: %s  in namespace %s : %w", pName, deployRepo, nc.GetNamespace(), err)
	}
	ps.Log.Info(fmt.Sprintf("Successfully created package: %s in deploy repo: %s", pName, deployRepo))
	return pr.ObjectMeta.Name, nil
}

// Verifies the existing package and creates the actuator package in deploy repo if missing.
// It pulls the required operators from the private catalogue of the Customer. Returns:
// a. The name of the package k8s resource that has the actuator manifests
// b. A bool value representing if the package was newly created
// c. error if any. If the error is not nil, other values should not be used.
func (ps *PorchPackageService) CreateNFDeployActuators(ctx context.Context,
	nc util.NamingContext,
	key VendorNFKey) (string, bool, error) {
	actuatorPkgName := nc.GetNFDeployActuatorPackageName(key.Vendor, key.Version, key.NFType)
	actuatorSrcRepo := nc.GetVendorNFManifestsRepoName()
	actuatorDstRepo := nc.GetDeployRepoName()
	createNewPkg := false
	// fetch the actuator resources from relevant source repository.
	_, actuatorPRR, _, err := ps.getLatestPackage(ctx, nc.GetNamespace(), actuatorPkgName, actuatorSrcRepo)
	if err != nil {
		return "", false, fmt.Errorf("Failed to fetch actuator resources: %w", err)
	}
	// fetch the existing resources if any from relevant destination repository.
	existingActuatorPR, existingActuatorPRR, isAbsent, err := ps.getLatestPackage(ctx, nc.GetNamespace(), actuatorPkgName, actuatorDstRepo)
	if err != nil && isAbsent {
		// no existing actuator package
		createNewPkg = true
	} else if err != nil {
		// other errors in fetching the package resources
		return "", false, fmt.Errorf("Failed to fetch existing actuator resources: %w", err)
	} else {
		// create new package only if existing content is different
		isSameContent := reflect.DeepEqual(actuatorPRR.Spec.Resources, existingActuatorPRR.Spec.Resources)
		createNewPkg = !isSameContent
	}

	if !createNewPkg {
		return existingActuatorPR.Name, false, nil
	}
	newPR, _, err := ps.createPackage(ctx, nc.GetNamespace(), actuatorPkgName, actuatorDstRepo, actuatorPRR.Spec.Resources)
	if err != nil {
		return "", false, fmt.Errorf("Failed to create actuators package in deploy repo: %w", err)
	}
	return newPR.Name, true, nil
}

// GetVendorExtensionPackage returns the list of valid k8s object yamls as string in the extension package.
// A valid k8s object is one with valid GVK present along with namespace and name.
// Each string represent a single k8s object. Returns empty list if no extension package present.
// All other files and objects will be ignored on the assumption that the package could
// have non-yaml files too like Readme etc for users.
// Returns an error if package was present but no valid k8s object was found.
func (ps *PorchPackageService) GetVendorExtensionPackage(
	ctx context.Context,
	nc util.NamingContext,
	key VendorNFKey) ([]string, error) {
	extnRepoName := nc.GetVendorNFManifestsRepoName()
	extnPkgName := nc.GetVendorExtensionPackageName(key.Vendor, key.Version, key.NFType)
	extnPR, extnPRR, isAbsent, err := ps.getLatestPackage(ctx, nc.GetNamespace(), extnPkgName, extnRepoName)
	if err != nil && isAbsent {
		ps.Log.V(1).Info(
			fmt.Sprintf("No vendor extension latest published package present for vendor NF: %#v, returning empty list", key))
		return []string{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("Failed to fetch vendor extension package: %w", err)
	}
	ps.Log.V(1).Info(fmt.Sprintf("Retrieving vendor extension k8s objects from packageRevision %s for vendor NF: %#v",
		extnPR.Name, key))
	extnResources := []string{}
	for name, content := range extnPRR.Spec.Resources {
		if name == KptfileName {
			ps.Log.V(1).Info(fmt.Sprintf("Skipping kptfile %s in vendor extension package for vendor NF: %#v",
				name, key))
			continue
		}
		rNodes, err := util.ParseStringToYamlNode(content)
		if err != nil {
			ps.Log.V(1).Error(err,
				fmt.Sprintf("Skipping file %s in vendor extension package for vendor NF: %#v, failed to parse.",
					name, key))
			continue
		}
		for _, rNode := range rNodes {
			if rNode.GetApiVersion() == "" || rNode.GetKind() == "" || rNode.GetName() == "" {
				ps.Log.V(1).Info(fmt.Sprintf("Skipping file %s in vendor extension package for vendor NF: %#v, invalid k8s object",
					name, key))
				continue
			}
			extnResources = append(extnResources, rNode.MustString())
		}
	}
	if len(extnResources) == 0 {
		return nil, errors.New(
			fmt.Sprintf("No valid vendor extension k8s object found in packageRevision %s for vendorNF %#v", extnPR.Name, key))
	}
	return extnResources, nil
}

func (ps *PorchPackageService) convertResourcesToYamlNodes(prr *porchapi.PackageRevisionResources) []*yaml.RNode {
	allNodes := []*yaml.RNode{}
	for n, c := range prr.Spec.Resources {
		nodes, err := util.ParseStringToYamlNode(c)
		if err != nil {
			ps.Log.Error(err, fmt.Sprintf("Failed to parse file : %s package resource, skipping", n))
		} else {
			allNodes = append(allNodes, nodes...)
		}
	}

	return allNodes
}

// creates the package in porch by creating a Package revision and then updating
// the auto created PackageRevisionResources with the given content.
func (ps *PorchPackageService) createPackage(ctx context.Context,
	namespace string,
	pkgName string,
	repo string,
	contents map[string]string) (*porchapi.PackageRevision, *porchapi.PackageRevisionResources, error) {
	pr, err := ps.createPackageRevision(ctx, namespace, pkgName, repo)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create package revision for the package name %s: %w", pkgName, err)
	}
	ps.Log.V(1).Info(fmt.Sprintf("Created package revision: %s for package: %s in deploy repo: %s",
		pr.ObjectMeta.Name, pkgName, repo))

	// fetch the PackageRevisionResources resource automatically created by previous step to update package contents
	ps.Log.V(1).Info(fmt.Sprintf("Fetching resources for package %s ", pkgName))
	prr, err := ps.getPackageRevisionResources(ctx, namespace, pr.ObjectMeta.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to fetch package revision resources for package: %s: %w", pkgName, err)
	}
	ps.Log.V(1).Info(fmt.Sprintf("Successfully fetched resources from package %s. Updating content...", pkgName))
	// adding the default kptfile in contents if not present
	if _, ok := contents[KptfileName]; !ok {
		contents[KptfileName] = prr.Spec.Resources[KptfileName]
		ps.Log.V(1).Info(fmt.Sprintf("Added default kptfile in package %s.", pkgName))
	}
	prr.Spec.Resources = contents
	err = ps.updatePackageRevisionResources(ctx, prr)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to update package revision resources for package: %s: %w", pkgName, err)
	}
	ps.Log.V(1).Info(fmt.Sprintf("Successfully updated resources in package %s with requested content.", pkgName))
	return pr, prr, nil
}

// creates a new PackageRevision in Porch server for the deploy package given the naming context.
func (ps *PorchPackageService) createPackageRevision(ctx context.Context, namespace string, pkgName string, repo string) (*porchapi.PackageRevision, error) {
	newPR := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    pkgName,
			WorkspaceName:  porchapi.WorkspaceName(fmt.Sprintf("v%d", time.Now().Unix())),
			RepositoryName: repo,
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeInit,
					Init: &porchapi.PackageInitTaskSpec{
						Description: "Created by Nephio for cluster deployment",
					},
				},
			},
		},
	}
	if err := ps.Client.Create(ctx, newPR); err != nil {
		return nil, err
	}
	return newPR, nil
}

// returns the latest package revision and package revision resources by retrieving
// the latest revision for the given arguments.
// also returns a bool flag and an error if any. The flag represents if the error was because the package revision was
// not present.
func (ps *PorchPackageService) getLatestPackage(ctx context.Context,
	namespace string,
	pkgName string,
	repo string) (*porchapi.PackageRevision, *porchapi.PackageRevisionResources, bool, error) {
	pr, isAbsent, err := ps.getLatestPackageRevision(ctx, namespace, pkgName, repo)
	if err != nil {
		return nil, nil, isAbsent, fmt.Errorf("Failed to fetch package revisions : %w", err)
	}
	ps.Log.V(1).Info(fmt.Sprintf("Fetching resources from package %s with revision %s", pr.ObjectMeta.Name, pr.Spec.Revision))
	prr, err := ps.getPackageRevisionResources(ctx, namespace, pr.ObjectMeta.Name)
	if err != nil {
		return nil, nil, false, fmt.Errorf("Failed to fetch package revision resources: %w", err)
	}

	return pr, prr, false, nil
}

// retrieves the latest published PackageRevision from Porch server for the given namespace, packageName and repository.
// there should be exactly one latest published revision for the given parameters.
// also returns a bool flag and an error if any. The flag represents if the error was because the package revision was
// not present.
func (ps *PorchPackageService) getLatestPackageRevision(ctx context.Context,
	namespace string,
	packageName string,
	repoName string) (*porchapi.PackageRevision, bool, error) {
	var prList porchapi.PackageRevisionList
	err := ps.Client.List(ctx, &prList)
	if err != nil {
		return nil, false, err
	}
	fList := []porchapi.PackageRevision{}
	for _, pr := range prList.Items {
		if pr.ObjectMeta.Namespace == namespace &&
			pr.Spec.RepositoryName == repoName &&
			pr.Spec.PackageName == packageName &&
			pr.Spec.Lifecycle == porchapi.PackageRevisionLifecyclePublished &&
			pr.ObjectMeta.Labels != nil &&
			pr.ObjectMeta.Labels[porchapi.LatestPackageRevisionKey] == porchapi.LatestPackageRevisionValue {
			fList = append(fList, pr)
		}
	}
	if len(fList) == 0 {
		return nil, true, errors.New(
			fmt.Sprintf("No latest published package found for [namespace: %s, packageName: %s, repoName: %s]", namespace, packageName, repoName))
	} else if len(fList) > 1 {
		return nil, false, errors.New(
			fmt.Sprintf("More than one latest published package found for [namespace: %s, packageName: %s, repoName: %s]", namespace, packageName, repoName))
	} else {
		return &fList[0], false, nil
	}
}

// getPackageRevisionResources method retrives PackageRevisionResources from Porch server for given Namespace and PackageName.
// A successful operation returns instance of PackageRevisionResources != nil and err == nil.
// A unsuccessful operation returns PackageRevisionResources == nil and err != nil.
func (ps *PorchPackageService) getPackageRevisionResources(ctx context.Context, namespace string, packageName string) (*porchapi.PackageRevisionResources, error) {
	var packageRevision porchapi.PackageRevisionResources
	if err := ps.Client.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      packageName,
	}, &packageRevision); err != nil {
		return nil, err
	}
	return &packageRevision, nil
}

// updates PackageRevisionResources resource and returns an error if any
func (ps *PorchPackageService) updatePackageRevisionResources(ctx context.Context, resources *porchapi.PackageRevisionResources) error {
	if err := ps.Client.Update(ctx, resources); err != nil {
		return err
	}
	return nil
}

// DeleteDeployPackage deletes packages from the deploy repo
func (ps *PorchPackageService) DeleteDeployPackage(ctx context.Context, nc util.NamingContext) error {
	deployRepo := nc.GetDeployRepoName()
	pName := nc.GetDeployPackageName()
	ps.Log.Info(fmt.Sprintf("Deleting package revisions for package: %s in deploy repo: %s", pName, deployRepo))
	if err := ps.deleteDeployPackageRevisions(ctx, nc); err != nil {
		return err
	}
	ps.Log.Info(fmt.Sprintf("Successfully deleted deploy packages for nfDeploy: %s", nc.GetNfDeployName()))
	return nil
}

func (ps *PorchPackageService) deleteDeployPackageRevisions(ctx context.Context, nc util.NamingContext) error {
	var prList porchapi.PackageRevisionList
	if err := ps.Client.List(ctx, &prList); err != nil {
		return err
	}
	for _, pr := range prList.Items {
		if pr.ObjectMeta.Namespace == nc.GetNamespace() &&
			pr.Spec.RepositoryName == nc.GetDeployRepoName() &&
			pr.Spec.PackageName == nc.GetDeployPackageName() {

			ps.Log.Info("Deleting deploy package revision", "name", pr.Name)
			if err := ps.Client.Delete(ctx, &pr); err != nil {
				return fmt.Errorf("error deleting package revision %s: %w", pr.Name, err)
			}
		}
	}
	return nil
}
