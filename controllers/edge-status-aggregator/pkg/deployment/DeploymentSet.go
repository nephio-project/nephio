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
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/go-logr/logr"
	edgewatcher "github.com/nephio-project/edge-watcher"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nephio-project/edge-status-aggregator/api/v1alpha1"
)

// DeploymentInfo : It contains the information of a single deployment
// corresponding to a single NfDeploy
type DeploymentInfo struct {
	deploymentName            string
	deployment                *Deployment
	edgewatcherSubscriberName string
}

var _ DeploymentManager = &deploymentManager{}

// DeploymentSet : DeploymentSet stores the address of all deployments in a map
// data structure
// A thread-safe set of Deployments
type DeploymentSet struct {
	deploymentSetMu sync.Mutex
	deployments     map[string]*DeploymentInfo
}

// deploymentManager : deploymentManager implements  Deployment Manager interface
type deploymentManager struct {
	deploymentSet    DeploymentSet
	subscriberChan   chan *edgewatcher.SubscriptionReq
	cancellationChan chan *edgewatcher.SubscriptionReq
	statusReader     client.Reader
	statusWriter     client.StatusWriter
	log              logr.Logger
}

// NewDeploymentManager : returns an initialised deploymentManager object
func NewDeploymentManager(
	subscriberChan chan *edgewatcher.SubscriptionReq,
	cancellationChan chan *edgewatcher.SubscriptionReq,
	statusReader client.Reader,
	statusWriter client.StatusWriter,
	log logr.Logger,
) *deploymentManager {
	deploymentManager := deploymentManager{}
	deploymentManager.subscriberChan = subscriberChan
	deploymentManager.cancellationChan = cancellationChan
	deploymentManager.deploymentSet = DeploymentSet{deployments: map[string]*DeploymentInfo{}}
	deploymentManager.statusReader = statusReader
	deploymentManager.statusWriter = statusWriter
	// TODO: segregate logs of different verbosity in deployment. Currently
	// all logs are with debug verbosity
	deploymentManager.log = log.V(1)
	return &deploymentManager
}

// ReportNFDeployEvent := Any changes in NFDeploy spec are reported to ReportNFDeployEvent.
// These changes can be addition/update/deletion of any sites or their
// connectivities in NFDeploy spec.
// If the deployment corresponding to this NFDeploy is not present in DeploymentSet,
// ReportNFDeployEvent is responsible for creating new deployment and its
// subscription for edge events. It also routes the given NFDeploy struct to its
// corresponding deployment, which syncs the deployment graph with NFDeploy spec.
// ReportNFDeployEvent is a synchronous method and should be called in a separate
// thread to prevent blocking on it.
func (deploymentManager *deploymentManager) ReportNFDeployEvent(
	nfdeploy v1alpha1.NfDeploy, namespacedName types.NamespacedName,
) {
	deploymentName := nfdeploy.Name
	edgewatcherSubscriberName := deploymentName + uuid.New().String()
	var isNewDeployment = false
	deploymentManager.deploymentSet.deploymentSetMu.Lock()
	if _, ok := deploymentManager.deploymentSet.deployments[deploymentName]; !ok {
		deployment := Deployment{}
		deployment.Init(deploymentManager.statusReader,
			deploymentManager.statusWriter, namespacedName, deploymentManager.log,
		)
		deploymentInfo := DeploymentInfo{
			deploymentName: deploymentName, deployment: &deployment, edgewatcherSubscriberName: edgewatcherSubscriberName,
		}
		deploymentManager.deploymentSet.deployments[deploymentName] = &deploymentInfo
		isNewDeployment = true
	}
	deployment := deploymentManager.deploymentSet.deployments[deploymentName].deployment
	deploymentManager.deploymentSet.deploymentSetMu.Unlock()
	deployment.ReportNFDeployEvent(nfdeploy)
	if isNewDeployment {
		subscribeReq := edgewatcher.SubscriptionReq{
			Ctx:   context.TODO(),
			Error: deployment.edgeErrorChan,
			EventOptions: edgewatcher.EventOptions{
				Type: edgewatcher.NfDeploySubscriber, SubscriptionName: deploymentName,
			}, SubscriberInfo: edgewatcher.SubscriberInfo{
				SubscriberName: edgewatcherSubscriberName,
				Channel:        deployment.edgeEventsChan,
			},
		}
		deploymentManager.subscriberChan <- &subscribeReq
		deployment.ListenSubscriptionStatus()

	}
}

// ReportNFDeployDeleteEvent := See DeploymentManager interface for method use
func (deploymentManager *deploymentManager) ReportNFDeployDeleteEvent(
	nfdeploy v1alpha1.NfDeploy,
) {
	deploymentManager.log.Info("ReportNFDeployDeleteEvent Enter")
	deploymentName := nfdeploy.Name
	deploymentManager.deploymentSet.deploymentSetMu.Lock()
	if _, ok := deploymentManager.deploymentSet.deployments[deploymentName]; !ok {
		deploymentManager.log.Info(
			"NFDeploy marked for deletion not found in deployment manager",
			"NFDeploy",
			nfdeploy.Name,
		)
		deploymentManager.deploymentSet.deploymentSetMu.Unlock()
	} else {
		edgewatcherSubscriberName := deploymentManager.deploymentSet.deployments[deploymentName].edgewatcherSubscriberName
		deploymentManager.deploymentSet.deployments[deploymentName].deployment.cancelCtx()
		delete(deploymentManager.deploymentSet.deployments, deploymentName)
		deploymentManager.deploymentSet.deploymentSetMu.Unlock()
		errorChan := make(chan error, 1)
		subscriptionReq := &edgewatcher.SubscriptionReq{
			Ctx:            context.Background(),
			Error:          errorChan,
			EventOptions:   edgewatcher.EventOptions{Type: edgewatcher.NfDeploySubscriber, SubscriptionName: deploymentName},
			SubscriberInfo: edgewatcher.SubscriberInfo{SubscriberName: edgewatcherSubscriberName},
		}
		deploymentManager.cancellationChan <- subscriptionReq
	}
	deploymentManager.log.Info("ReportNFDeployDeleteEvent Exit")
}
