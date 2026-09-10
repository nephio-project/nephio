/*
Copyright 2026 The Nephio Authors.

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
	"testing"

	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	invv1alpha1 "github.com/nokia/k8s-ipam/apis/inv/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testTopology  = "topology-a"
	otherTopology = "topology-b"
)

// testNetwork returns a Network bound to the given topology.
func testNetwork(namespace, name, topology string) *infrav1alpha1.Network {
	return &infrav1alpha1.Network{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec:       infrav1alpha1.NetworkSpec{Topology: topology},
	}
}

// testNode returns a Node carrying the labels the handler matches on.
func withoutTopologyLabel(node *invv1alpha1.Node) *invv1alpha1.Node {
	delete(node.Labels, invv1alpha1.NephioTopologyKey)
	return node
}

func testNode(provider, topology string) *invv1alpha1.Node {
	return &invv1alpha1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "inventory",
			Name:      "node-1",
			Labels: map[string]string{
				invv1alpha1.NephioProviderKey: provider,
				invv1alpha1.NephioTopologyKey: topology,
			},
		},
	}
}

// testRequest returns the reconcile.Request a Network is expected to produce.
func testRequest(namespace, name string) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: namespace, Name: name}}
}

// drainQueue returns everything the handler enqueued, without blocking on an
// empty queue.
func drainQueue(q workqueue.TypedRateLimitingInterface[reconcile.Request]) []reconcile.Request {
	requests := make([]reconcile.Request, 0, q.Len())
	for q.Len() > 0 {
		item, shutdown := q.Get()
		if shutdown {
			break
		}
		requests = append(requests, item)
		q.Done(item)
	}
	return requests
}

// TestNodeEventHandler covers which Node changes enqueue which Networks.
//
// There is no case for an object of another kind: the handler is typed on
// *invv1alpha1.Node and registered through source.Kind, so the mispairing that
// caused this bug no longer compiles and cannot be reached at runtime.
// countingClient records how many times the handler scans the cache. A per-Node
// list would be invisible in the requests alone, since the queue folds the
// duplicates away while nothing is consuming it.
type countingClient struct {
	client.Client
	lists int
}

func (c *countingClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	c.lists++
	return c.Client.List(ctx, list, opts...)
}

func TestNodeEventHandler(t *testing.T) {
	// bothTopologies is used by the migration cases, which need a Network on
	// each side of the move.
	bothTopologies := []*infrav1alpha1.Network{
		testNetwork("ns-a", "network-a", testTopology),
		testNetwork("ns-b", "network-b", otherTopology),
	}

	cases := map[string]struct {
		node     *invv1alpha1.Node
		previous *invv1alpha1.Node // when set, delivered as an update previous -> node
		deleted  bool              // when set, delivered as a delete of node
		networks []*infrav1alpha1.Network
		expected []reconcile.Request
		// expectedLists is 0 when no Node carries an eligible topology: there
		// is nothing to match, so the cache is never scanned.
		expectedLists int
	}{
		"matching provider and topology": {
			node:          testNode(nokiaSRLProvider, testTopology),
			expected:      []reconcile.Request{testRequest("ns-a", "network-a")},
			expectedLists: 1,
		},
		"other provider": {
			node: testNode("other.vendor.com", testTopology),
		},
		"other topology": {
			node:          testNode(nokiaSRLProvider, otherTopology),
			expectedLists: 1,
		},
		// Absent and empty are different labels. testNode always sets the key,
		// so the absent case has to remove it.
		"provider but the topology label is absent": {
			node:     withoutTopologyLabel(testNode(nokiaSRLProvider, testTopology)),
			networks: []*infrav1alpha1.Network{testNetwork("ns-a", "network-a", "")},
		},
		"an explicitly empty topology matches an empty-topology Network": {
			node:          testNode(nokiaSRLProvider, ""),
			networks:      []*infrav1alpha1.Network{testNetwork("ns-a", "network-a", "")},
			expected:      []reconcile.Request{testRequest("ns-a", "network-a")},
			expectedLists: 1,
		},
		"moving out of the empty topology wakes both sides": {
			previous:      testNode(nokiaSRLProvider, ""),
			node:          testNode(nokiaSRLProvider, testTopology),
			networks:      []*infrav1alpha1.Network{testNetwork("ns-a", "network-a", ""), testNetwork("ns-b", "network-b", testTopology)},
			expected:      []reconcile.Request{testRequest("ns-a", "network-a"), testRequest("ns-b", "network-b")},
			expectedLists: 1,
		},
		"moving into the empty topology wakes both sides": {
			previous:      testNode(nokiaSRLProvider, testTopology),
			node:          testNode(nokiaSRLProvider, ""),
			networks:      []*infrav1alpha1.Network{testNetwork("ns-a", "network-a", ""), testNetwork("ns-b", "network-b", testTopology)},
			expected:      []reconcile.Request{testRequest("ns-a", "network-a"), testRequest("ns-b", "network-b")},
			expectedLists: 1,
		},
		"no labels": {
			node: &invv1alpha1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}},
		},
		"every matching network, namespace and name intact": {
			node: testNode(nokiaSRLProvider, testTopology),
			networks: []*infrav1alpha1.Network{
				testNetwork("ns-a", "network-a", testTopology),
				testNetwork("ns-b", "network-b", testTopology),
				testNetwork("ns-c", "network-c", otherTopology),
			},
			expected: []reconcile.Request{
				testRequest("ns-a", "network-a"),
				testRequest("ns-b", "network-b"),
			},
			expectedLists: 1,
		},
		// A Node leaving the inventory has to wake its Network so the config
		// rendered from it stops referring to a node that is gone.
		"a deleted Node wakes its Network": {
			node:          testNode(nokiaSRLProvider, testTopology),
			deleted:       true,
			expected:      []reconcile.Request{testRequest("ns-a", "network-a")},
			expectedLists: 1,
		},
		"a deleted Node of another provider wakes nothing": {
			node:    testNode("other.vendor.com", testTopology),
			deleted: true,
		},
		"a deleted Node in the empty topology wakes its Network": {
			node:          testNode(nokiaSRLProvider, ""),
			deleted:       true,
			networks:      []*infrav1alpha1.Network{testNetwork("ns-a", "network-a", "")},
			expected:      []reconcile.Request{testRequest("ns-a", "network-a")},
			expectedLists: 1,
		},
		"an unchanged update enqueues each Network once": {
			node:          testNode(nokiaSRLProvider, testTopology),
			previous:      testNode(nokiaSRLProvider, testTopology),
			expected:      []reconcile.Request{testRequest("ns-a", "network-a")},
			expectedLists: 1,
		},
		"moving between topologies wakes the Network left and the one joined": {
			previous: testNode(nokiaSRLProvider, testTopology),
			node:     testNode(nokiaSRLProvider, otherTopology),
			networks: bothTopologies,
			expected: []reconcile.Request{
				testRequest("ns-a", "network-a"),
				testRequest("ns-b", "network-b"),
			},
			expectedLists: 1,
		},
		"losing the provider label still wakes the Network left behind": {
			previous:      testNode(nokiaSRLProvider, testTopology),
			node:          testNode("other.vendor.com", testTopology),
			expected:      []reconcile.Request{testRequest("ns-a", "network-a")},
			expectedLists: 1,
		},
		"gaining the provider label wakes the Network joined": {
			previous:      testNode("other.vendor.com", testTopology),
			node:          testNode(nokiaSRLProvider, testTopology),
			expected:      []reconcile.Request{testRequest("ns-a", "network-a")},
			expectedLists: 1,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, infrav1alpha1.AddToScheme(scheme))
			require.NoError(t, invv1alpha1.AddToScheme(scheme))

			networks := tc.networks
			if networks == nil {
				networks = []*infrav1alpha1.Network{
					testNetwork("ns-a", "network-a", testTopology)}
			}
			objects := make([]client.Object, 0, len(networks))
			for _, network := range networks {
				objects = append(objects, network)
			}

			counting := &countingClient{Client: fake.NewClientBuilder().
				WithScheme(scheme).WithObjects(objects...).Build()}
			handler := &nodeEventHandler{client: counting}
			queue := workqueue.NewTypedRateLimitingQueue(
				workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())

			switch {
			case tc.previous != nil:
				handler.Update(context.Background(),
					event.TypedUpdateEvent[*invv1alpha1.Node]{
						ObjectOld: tc.previous, ObjectNew: tc.node}, queue)
			case tc.deleted:
				handler.Delete(context.Background(),
					event.TypedDeleteEvent[*invv1alpha1.Node]{Object: tc.node}, queue)
			default:
				handler.Create(context.Background(),
					event.TypedCreateEvent[*invv1alpha1.Node]{Object: tc.node}, queue)
			}

			assert.ElementsMatch(t, tc.expected, drainQueue(queue))
			assert.Equal(t, tc.expectedLists, counting.lists, "cache scans")
		})
	}
}
