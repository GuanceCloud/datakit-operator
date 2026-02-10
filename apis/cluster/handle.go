// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package cluster

import (
	"context"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/kubernetes/client"
	"k8s.io/client-go/informers"
	corev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type Handler struct {
	Factory   informers.SharedInformerFactory
	PodLister corev1.PodLister
	// DeploymentLister appsv1.DeploymentLister
	// other...
}

func NewHandler(k8sClient client.Client) *Handler {
	informerFactory := informers.NewSharedInformerFactory(k8sClient.Clientset(), 0)
	return &Handler{
		Factory:   informerFactory,
		PodLister: informerFactory.Core().V1().Pods().Lister(),
		// DeploymentLister: informerFactory.Apps().V1().Deployments().Lister(),
	}
}

func (h *Handler) Start(ctx context.Context) error {
	log.Info("starting informers...")

	h.Factory.Start(ctx.Done())

	log.Info("waiting for cache sync...")
	if !cache.WaitForCacheSync(ctx.Done(),
		h.Factory.Core().V1().Pods().Informer().HasSynced,
		// h.Factory.Apps().V1().Deployments().Informer().HasSynced,
	) {
		log.Error("failed to sync cache")
		return context.DeadlineExceeded
	}

	log.Info("cache sync completed")
	return nil
}
