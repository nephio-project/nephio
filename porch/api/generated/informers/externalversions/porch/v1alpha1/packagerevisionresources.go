// Copyright 2023 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	versioned "github.com/GoogleContainerTools/kpt/porch/api/generated/clientset/versioned"
	internalinterfaces "github.com/GoogleContainerTools/kpt/porch/api/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/generated/listers/porch/v1alpha1"
	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// PackageRevisionResourcesInformer provides access to a shared informer and lister for
// PackageRevisionResources.
type PackageRevisionResourcesInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.PackageRevisionResourcesLister
}

type packageRevisionResourcesInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewPackageRevisionResourcesInformer constructs a new informer for PackageRevisionResources type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewPackageRevisionResourcesInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredPackageRevisionResourcesInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredPackageRevisionResourcesInformer constructs a new informer for PackageRevisionResources type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredPackageRevisionResourcesInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PorchV1alpha1().PackageRevisionResources(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PorchV1alpha1().PackageRevisionResources(namespace).Watch(context.TODO(), options)
			},
		},
		&porchv1alpha1.PackageRevisionResources{},
		resyncPeriod,
		indexers,
	)
}

func (f *packageRevisionResourcesInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredPackageRevisionResourcesInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *packageRevisionResourcesInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&porchv1alpha1.PackageRevisionResources{}, f.defaultInformer)
}

func (f *packageRevisionResourcesInformer) Lister() v1alpha1.PackageRevisionResourcesLister {
	return v1alpha1.NewPackageRevisionResourcesLister(f.Informer().GetIndexer())
}
