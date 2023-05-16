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

package v1alpha1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var nfdeploylog = logf.Log.WithName("nfdeploy-resource")

func (r *NfDeploy) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-nfdeploy-nephio-org-v1alpha1-nfdeploy,mutating=false,failurePolicy=fail,sideEffects=None,groups=nfdeploy.nephio.org,resources=nfdeploys,verbs=create,versions=v1alpha1,name=vnfdeploy.google.com,admissionReviewVersions=v1

var _ webhook.Validator = &NfDeploy{}

type void struct{}

// connected : Represents if two nodes are connected
var connected void

// present : Represents if a node is present in list
var present void

type NFType string

const UnspecifiedNFType NFType = "unspecified"

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NfDeploy) ValidateCreate() error {
	nfdeploylog.Info("validate create", "name", r.Name)
	var presentNodes = make(map[string]void)
	for _, site := range r.Spec.Sites {
		if _, isPresent := presentNodes[site.Id]; isPresent {
			return errors.New("NF with id - " + site.Id + " is already present")
		}
		presentNodes[site.Id] = present
	}
	var presentConnections = make(map[string]map[string]void)

	for _, site := range r.Spec.Sites {
		for _, connection := range site.Connectivities {
			if _, present := presentConnections[site.Id]; !present {
				presentConnections[site.Id] = make(map[string]void)
			}
			if _, isPresent := presentConnections[site.Id][connection.NeighborName]; isPresent {
				return errors.New(
					"Multiple connections found between " + site.Id +
						" and " + connection.NeighborName,
				)
			}
			presentConnections[site.Id][connection.NeighborName] = connected
		}
	}
	for _, site := range r.Spec.Sites {
		for _, connection := range site.Connectivities {
			if _, isPresent := presentNodes[connection.NeighborName]; !isPresent {
				return errors.New("NF with id " + connection.NeighborName + " is not present")
			}
			if _, isPresent := presentConnections[connection.NeighborName][site.Id]; !isPresent {
				return errors.New(
					"Connectivity between " + connection.NeighborName +
						" and " + site.Id + " is not present",
				)
			}
		}
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NfDeploy) ValidateUpdate(old runtime.Object) error {
	nfdeploylog.Info("validate update", "name", r.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NfDeploy) ValidateDelete() error {
	nfdeploylog.Info("validate delete", "name", r.Name)

	// TODO: fill in your validation logic upon object deletion.
	return nil
}
