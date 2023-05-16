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

package deployment

import (
	"github.com/nephio-project/edge-status-aggregator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

type DeploymentManager interface {

	// ReportNFDeployEvent := Any changes in NFDeploy spec are reported to ReportNFDeployEvent.
	// These changes can be addition/update/deletion of any sites or their
	// connectivities in NFDeploy spec.
	// If the deployment corresponding to this NFDeploy is not present in DeploymentSet,
	// ReportNFDeployEvent is responsible for creating new deployment and its
	// subscription for edge events. It also routes the given NFDeploy struct to its
	// corresponding deployment, which syncs the deployment graph with NFDeploy spec.
	// ReportNFDeployEvent is a synchronous method and should be called in a separate
	// thread to prevent blocking on it.
	ReportNFDeployEvent(
		nfdeploy v1alpha1.NfDeploy, namespacedName types.NamespacedName,
	)

	// ReportNFDeployDeleteEvent :=
	// This method cleans up the state maintained for the Deployment, which includes:
	// 1. Terminating routines which are listening for edge events, for the corresponding NFs of the NFDeploy.
	// 2. Removing the data-structure which store the deployment state in DeploymentSet
	// 3. Cancelling edge watcher subscription - edge watcher expects all subscribers to consume all edge events.
	// In case a subscriber does not consume the event, it never leaves edge watcher's event queue,
	// resulting in error.
	ReportNFDeployDeleteEvent(
		nfdeploy v1alpha1.NfDeploy,
	)
}
