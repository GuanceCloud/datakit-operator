// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package election

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	typedauthorizationv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	typedcoordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	coordinationlisters "k8s.io/client-go/listers/coordination/v1"
	"k8s.io/client-go/tools/cache"
)

type informerLeaseCache struct {
	coordinationlisters.LeaseNamespaceLister
	informer   cache.SharedIndexInformer
	ctx        context.Context
	startOnce  sync.Once
	watchReady atomic.Bool
}

// leaseWatch invalidates readiness whenever the reflector stops a watch,
// including stream errors that require a new LIST before WATCH can resume.
type leaseWatch struct {
	watch.Interface
	ready *atomic.Bool
}

func (w *leaseWatch) Stop() {
	w.ready.Store(false)
	w.Interface.Stop()
}

func NewCoordinator(client typedcoordinationv1.LeaseInterface, accessReviews typedauthorizationv1.SelfSubjectAccessReviewInterface, namespace string) *Coordinator {
	started := time.Now()
	coordinator := &Coordinator{
		store: client, accessReviews: accessReviews, namespace: namespace, now: time.Now,
		elapsed: func() time.Duration { return time.Since(started) },
	}
	leaseCache := &informerLeaseCache{}
	selector := labels.Set{managedLeaseLabel: "true"}.AsSelector().String()
	listWatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			leaseCache.watchReady.Store(false)
			options.LabelSelector = selector
			leases, err := client.List(leaseCache.ctx, options)
			if err != nil && leaseCache.ctx.Err() == nil {
				_ = coordinator.storageError(operationInformer, "list", namespace, err)
			}
			return leases, err
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			leaseCache.watchReady.Store(false)
			options.LabelSelector = selector
			watcher, err := client.Watch(leaseCache.ctx, options)
			if err != nil {
				if leaseCache.ctx.Err() == nil {
					_ = coordinator.storageError(operationInformer, "watch", namespace, err)
				}
				return nil, err
			}
			leaseCache.watchReady.Store(true)
			return &leaseWatch{Interface: watcher, ready: &leaseCache.watchReady}, nil
		},
	}
	leaseCache.informer = cache.NewSharedIndexInformer(
		listWatch, &coordinationv1.Lease{}, 0,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	leaseCache.LeaseNamespaceLister = coordinationlisters.NewLeaseLister(leaseCache.informer.GetIndexer()).Leases(namespace)
	coordinator.cache = leaseCache
	return coordinator
}

func (c *informerLeaseCache) Start(ctx context.Context) {
	c.startOnce.Do(func() {
		c.ctx = ctx
		go func() {
			defer c.watchReady.Store(false)
			c.informer.Run(ctx.Done())
		}()
	})
}

func (c *informerLeaseCache) Ready() bool {
	return c.informer.HasSynced() && c.watchReady.Load()
}

func (c *Coordinator) Start(ctx context.Context) {
	// Older deployments may lack the namespace projection; keep their other APIs available.
	if c.namespace != "" {
		c.cache.Start(ctx)
	}
}
