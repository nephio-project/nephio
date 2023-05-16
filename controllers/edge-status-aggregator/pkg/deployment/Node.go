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
	"time"

	nfdeployments "github.com/nephio-project/api/nf_deployments/v1alpha1"
)

type void struct{}

type Node struct {
	Id          string
	NFType      NFType
	Connections map[string]void
}

type UPFNode struct {
	Node
	Spec   UPFSpec
	Status NFStatus
}

type SMFNode struct {
	Node
	Spec   SMFSpec
	Status NFStatus
}

type NFStatus struct {
	// all the NFConditions that were true in the last observed edge event
	activeConditions   map[nfdeployments.NFDeploymentConditionType]string
	state              nfdeployments.NFDeploymentConditionType
	stateMessage       string
	lastEventTimestamp time.Time
}

type AMFNode struct {
	Node
}

type AUSFNode struct {
	Node
	Status NFStatus
}

type UDMNode struct {
	Node
	Status NFStatus
}

// UPFSpec : Stores the spec related to UPF
type UPFSpec struct {
	throughput  string
	clusterName string
}

// SMFSpec : Stores the spec related to SMF
type SMFSpec struct {
	maxSessions string
	clusterName string
}

// AMFSpec : Stores the spec related to AMF
type AMFSpec struct {
	maxSubscribers string
}

type NFType string

const (
	UPF               NFType = "upf"
	SMF               NFType = "smf"
	AMF               NFType = "amf"
	UnspecifiedNFType NFType = "unspecified"
)
