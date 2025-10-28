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

func (e *nodeEventHandler) Create(ctx context.Context, evt event.TypedCreateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.add(ctx, evt.Object, q)
}

func (e *nodeEventHandler) Update(ctx context.Context, evt event.TypedUpdateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.add(ctx, evt.ObjectOld, q)
	e.add(ctx, evt.ObjectNew, q)
}

func (e *nodeEventHandler) Delete(ctx context.Context, evt event.TypedDeleteEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.add(ctx, evt.Object, q)
}

func (e *nodeEventHandler) Generic(ctx context.Context, evt event.TypedGenericEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.add(ctx, evt.Object, q)
}

func (e *nodeEventHandler) add(ctx context.Context, obj client.Object, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	cr, ok := obj.(*invv1alpha1.Node)
	if !ok {
		return
	}
	log := log.FromContext(ctx)
	log.Info("event", "kind", obj.GetObjectKind(), "name", cr.GetName())

	networks := &infrav1alpha1.NetworkList{}
	if err := e.client.List(ctx, networks); err != nil {
		return
	}

	for _, network := range networks.Items {
		// only enqueue if the provider and the network topology match
		if cr.Labels[invv1alpha1.NephioProviderKey] == nokiaSRLProvider &&
			cr.Labels[invv1alpha1.NephioTopologyKey] == network.Spec.Topology {
			log.Info("event requeue network", "name", network.GetName())
			queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: network.GetNamespace(),
				Name:      network.GetName()}})
		}
	}
}
