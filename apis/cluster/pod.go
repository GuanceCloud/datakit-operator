// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package cluster

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/kubernetes/client"
)

func CheckPodRBAC(k8sClient client.Client) error {
	ctx := context.Background()
	_, err := k8sClient.Clientset().CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{Limit: 1})
	if errors.IsForbidden(err) || errors.IsUnauthorized(err) {
		return fmt.Errorf("missing RBAC permissions to access Pod: %w", err)
	}
	if err != nil {
		return fmt.Errorf("cannot list Pod: %w", err)
	}
	return nil
}

// ListAllPods 对应 GET /api/v1/pods
func (h *Handler) ListAllPods(c *gin.Context) {
	pods, err := h.PodLister.Pods("").List(labels.Everything())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pods)
}

// ListPods 对应 GET /api/v1/namespaces/:ns/pods
func (h *Handler) ListPods(c *gin.Context) {
	ns := c.Param("namespace")

	pods, err := h.PodLister.Pods(ns).List(labels.Everything())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pods)
}

// GetPod 对应 GET /api/v1/namespaces/:ns/pods/:name
func (h *Handler) GetPod(c *gin.Context) {
	ns := c.Param("namespace")
	name := c.Param("name")

	pod, err := h.PodLister.Pods(ns).Get(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "not found"})
		return
	}
	c.JSON(http.StatusOK, pod)
}
