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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NFDeployConditionType string

const (
	// Reconciling implies that the deployment is progressing.
	// Reconciliation for a deployment is considered when a
	// 1. new version of at-least one NF is adopted,
	// 2. when new pods scale up or old pods scale down,
	// 3. when required peering is in progress, or,
	// 4. location of at-least one NF changes.
	//
	// Condition name follows Kpt guidelines.
	DeploymentReconciling NFDeployConditionType = "Reconciling"

	// Deployment is unable to make progress towards Reconciliation.
	// Reasons could be NF creation failure, Peering failure etc.
	//
	// Condition name follows Kpt guidelines.
	DeploymentStalled NFDeployConditionType = "Stalled"

	// The Deployment is considered available when following conditions hold:
	// 1. All the NFs are Available.
	// 2. The NFs are making progress towards peering on the required
	//    interfaces.
	DeploymentPeering NFDeployConditionType = "Peering"

	// The Deployment is said to be Ready when all the NFs are Ready.
	// At this stage, the deployment is ready to serve requests.
	DeploymentReady NFDeployConditionType = "Ready"
)

type NFDeployCondition struct {
	// Type of deployment condition.
	Type NFDeployConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

type NfDeployStatus struct {
	// The generation observed by the deployment controller.
	ObservedGeneration int32 `json:"observedGeneration,omitempty"`

	// Total number of NFs targeted by this deployment
	TargetedNFs int32 `json:"targetedNFs,omitempty"`

	// Total number of NFs targeted by this deployment with a Ready Condition set.
	ReadyNFs int32 `json:"readyNFs,omitempty"`

	// Total number of NFs targeted by this deployment with an Available Condition set.
	AvailableNFs int32 `json:"availableNFs,omitempty"`

	// Total number of NFs targeted by this deployment with a Stalled Condition set.
	StalledNFs int32 `json:"stalledNFs,omitempty"`

	// Current service state of the UPF.
	Conditions []NFDeployCondition `json:"conditions,omitempty"`
}
