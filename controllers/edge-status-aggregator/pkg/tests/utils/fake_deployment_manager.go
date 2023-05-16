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
	"github.com/go-logr/logr"
	nfdeployv1alpha1 "github.com/nephio-project/edge-status-aggregator/api/v1alpha1"
	"github.com/nephio-project/edge-status-aggregator/deployment"
	edgewatcher "github.com/nephio-project/edge-watcher"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FakeDeploymentManager : Implements Deployment Manager interface
type FakeDeploymentManager struct {
	SignalChan          chan error
	DeploymentManager   deployment.DeploymentManager
	SubscriptionReqChan chan *edgewatcher.SubscriptionReq
}

// NewFakeDeploymentManager : Returns new FakeDeploymentManager
func NewFakeDeploymentManager(
	reader client.Reader, writer client.StatusWriter, log logr.Logger,
) FakeDeploymentManager {

	subscriptionReq := make(chan *edgewatcher.SubscriptionReq)
	cancellationReq := make(chan *edgewatcher.SubscriptionReq, 10)
	return FakeDeploymentManager{
		SignalChan: make(chan error),
		DeploymentManager: deployment.NewDeploymentManager(
			subscriptionReq, cancellationReq, reader, writer, log,
		), SubscriptionReqChan: subscriptionReq,
	}
}

var _ deployment.DeploymentManager = &FakeDeploymentManager{}

// ReportNFDeployEvent : Fake implementation. Currently, it checks if the given
// function is called
func (fakeDeploymentManager *FakeDeploymentManager) ReportNFDeployEvent(
	deploy nfdeployv1alpha1.NfDeploy, namespacedName types.NamespacedName,
) {

	go fakeDeploymentManager.DeploymentManager.ReportNFDeployEvent(
		deploy, namespacedName,
	)
	fakeDeploymentManager.SignalChan <- nil
}

func (fakeDeploymentManager *FakeDeploymentManager) ReportNFDeployDeleteEvent(
	deploy nfdeployv1alpha1.NfDeploy,
) {
	fakeDeploymentManager.DeploymentManager.ReportNFDeployDeleteEvent(
		deploy,
	)
	fakeDeploymentManager.SignalChan <- nil
}
