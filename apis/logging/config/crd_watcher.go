// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package config

import (
	"context"
	"time"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/kubernetes/client"
	loggingv1alpha1 "gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/kubernetes/pkg/apis/datakits/v1alpha1"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/kubernetes/pkg/client/informers/externalversions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type loggingConfigWatcher struct {
	client client.Client

	queue    workqueue.DelayingInterface
	informer cache.SharedIndexInformer
	store    cache.Store
}

func newLoggingConfigWatcher(client client.Client) *loggingConfigWatcher {
	return &loggingConfigWatcher{
		client: client,
		queue:  workqueue.NewDelayingQueue(),
	}
}

func (w *loggingConfigWatcher) start(ctx context.Context) {
	log.Info("starting logging config watcher")

	// RBAC 预检：尝试进行一次最小化的 List 调用，若无权限或 CRD 不存在则退出
	if clientset := w.client.Logging(); clientset != nil {
		_, err := clientset.LoggingV1alpha1().ClusterLoggingConfigs().List(ctx, metav1.ListOptions{Limit: 1})
		if err != nil {
			if apierrors.IsForbidden(err) {
				log.Warnf("missing RBAC permission to access ClusterLoggingConfig: %v; exit logging config watcher", err)
			} else if apierrors.IsNotFound(err) {
				log.Warnf("ClusterLoggingConfig CRD resource type not found: %v; exit logging config watcher", err)
			} else {
				log.Warnf("failed to access ClusterLoggingConfig: %v; exit logging config watcher", err)
			}
			return
		}
	}

	log.Info("starting CRD informer for ClusterLoggingConfig")

	w.setupInformer()

	go func() {
		w.processQueue(ctx)
	}()

	go func() {
		w.informer.Run(ctx.Done())
	}()

	if !cache.WaitForCacheSync(ctx.Done(), w.informer.HasSynced) {
		log.Error("failed to sync informer cache")
		return
	}

	log.Info("logging config watcher started successfully")

	<-ctx.Done()
	w.queue.ShutDown()
	log.Info("logging config watcher stopped")
}

func (w *loggingConfigWatcher) setupInformer() {
	clientset := w.client.Logging()
	informerFactory := externalversions.NewSharedInformerFactoryWithOptions(
		clientset, 0,
		externalversions.WithTweakListOptions(func(v *metav1.ListOptions) { v.Limit = 50 }),
	)

	w.informer = informerFactory.Logging().V1alpha1().ClusterLoggingConfigs().Informer()
	w.store = w.informer.GetStore()

	w.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			w.enqueue(obj, "add")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			w.enqueue(newObj, "update")
		},
		DeleteFunc: func(obj interface{}) {
			w.enqueue(obj, "delete")
		},
	})
}

func (w *loggingConfigWatcher) enqueue(obj interface{}, action string) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("failed to get key for object: %v", err)
		return
	}

	w.queue.AddAfter(key, time.Second)
	log.Debugf("enqueued %s event for key: %s", action, key)
}

func (w *loggingConfigWatcher) processQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !w.processNextItem() {
				return
			}
		}
	}
}

func (w *loggingConfigWatcher) processNextItem() bool {
	keyObj, quit := w.queue.Get()
	if quit {
		return false
	}
	defer w.queue.Done(keyObj)

	key := keyObj.(string)

	obj, exists, err := w.store.GetByKey(key)
	if err != nil {
		log.Errorf("failed to get object by key %s: %v", key, err)
		return true
	}

	if !exists {
		RemoveConfig(key)
		log.Infof("logging config deleted: %s", key)
		return true
	}

	logging, ok := obj.(*loggingv1alpha1.ClusterLoggingConfig)
	if !ok {
		log.Warnf("failed to convert object to ClusterLoggingConfig: %v", obj)
		return true
	}

	log.Debugf("processNextItem: key=%s, object=%+v", key, logging)

	// 更新 cache
	AddOrUpdateConfig(key, logging)
	return true
}

func startLoggingConfigWatcher(ctx context.Context) {
	client, err := client.NewClientInCluster()
	if err != nil {
		log.Error(err)
		return
	}

	watcher := newLoggingConfigWatcher(client)
	go func() {
		watcher.start(ctx)
	}()
}
