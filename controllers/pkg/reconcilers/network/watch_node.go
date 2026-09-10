/*
Copyright 2023-2025 The Nephio Authors.

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

package network

import (
	"context"

	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	invv1alpha1 "github.com/nokia/k8s-ipam/apis/inv/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type nodeEventHandler struct {
	client client.Client
}

func (e *nodeEventHandler) Create(ctx context.Context, evt event.TypedCreateEvent[*invv1alpha1.Node], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.enqueue(ctx, q, evt.Object)
}

func (e *nodeEventHandler) Update(ctx context.Context, evt event.TypedUpdateEvent[*invv1alpha1.Node], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// Both sides: a Node that moves between topologies has to wake the Network
	// it left as well as the one it joined.
	e.enqueue(ctx, q, evt.ObjectOld, evt.ObjectNew)
}

func (e *nodeEventHandler) Delete(ctx context.Context, evt event.TypedDeleteEvent[*invv1alpha1.Node], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.enqueue(ctx, q, evt.Object)
}

func (e *nodeEventHandler) Generic(ctx context.Context, evt event.TypedGenericEvent[*invv1alpha1.Node], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.enqueue(ctx, q, evt.Object)
}

// enqueue wakes every Network whose topology one of these Nodes belongs to.
//
// The Nodes are taken together rather than one at a time. An update carries
// two, and listing per Node would scan the cache twice and add each matching
// Network twice: the queue only folds those into one reconcile while no worker
// picks the first up in between, which is not something a handler can rely on.
func (e *nodeEventHandler) enqueue(ctx context.Context, queue workqueue.TypedRateLimitingInterface[reconcile.Request], nodes ...*invv1alpha1.Node) {
	log := log.FromContext(ctx)

	topologies := map[string]struct{}{}
	for _, node := range nodes {
		if node == nil {
			continue
		}
		log.Info("event", "kind", node.GetObjectKind(), "name", node.GetName())

		labels := node.GetLabels()
		if labels[invv1alpha1.NephioProviderKey] != nokiaSRLProvider {
			continue
		}
		// Absent, not empty. A map lookup gives "" for both, but a label set
		// to "" is a topology as far as the reconciler is concerned: it lists
		// Nodes with MatchingLabels, which selects that value and not the
		// missing key. Treating the two alike here would leave a Network whose
		// topology is empty asleep through changes to its own Nodes.
		topology, present := labels[invv1alpha1.NephioTopologyKey]
		if !present {
			continue
		}
		topologies[topology] = struct{}{}
	}
	if len(topologies) == 0 {
		return
	}

	networks := &infrav1alpha1.NetworkList{}
	if err := e.client.List(ctx, networks); err != nil {
		log.Error(err, "cannot list networks, event dropped")
		return
	}

	for i := range networks.Items {
		network := &networks.Items[i]
		if _, ok := topologies[network.Spec.Topology]; !ok {
			continue
		}
		log.Info("event requeue network", "name", network.GetName())
		queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: network.GetNamespace(),
			Name:      network.GetName()}})
	}
}
