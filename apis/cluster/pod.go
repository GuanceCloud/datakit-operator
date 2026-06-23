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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/kubernetes/client"
)

const podViewEBPFV1 = "ebpf-v1"

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

	if c.Query("view") == podViewEBPFV1 {
		c.JSON(http.StatusOK, trimPodsForEBPFV1(pods))
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

	if c.Query("view") == podViewEBPFV1 {
		c.JSON(http.StatusOK, trimPodsForEBPFV1(pods))
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
	if c.Query("view") == podViewEBPFV1 {
		c.JSON(http.StatusOK, trimPodForEBPFV1(pod))
		return
	}
	c.JSON(http.StatusOK, pod)
}

func trimPodsForEBPFV1(pods []*corev1.Pod) []corev1.Pod {
	trimmed := make([]corev1.Pod, 0, len(pods))
	for _, pod := range pods {
		trimmed = append(trimmed, trimPodForEBPFV1(pod))
	}
	return trimmed
}

func trimPodForEBPFV1(pod *corev1.Pod) corev1.Pod {
	if pod == nil {
		return corev1.Pod{}
	}

	containers := make([]corev1.Container, 0, len(pod.Spec.Containers))
	for _, container := range pod.Spec.Containers {
		containers = append(containers, corev1.Container{
			Name:  container.Name,
			Ports: container.Ports,
		})
	}

	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID:             pod.UID,
			Name:            pod.Name,
			Namespace:       pod.Namespace,
			Labels:          pod.Labels,
			OwnerReferences: pod.OwnerReferences,
		},
		Spec: corev1.PodSpec{
			Containers:  containers,
			HostNetwork: pod.Spec.HostNetwork,
			HostPID:     pod.Spec.HostPID,
			HostIPC:     pod.Spec.HostIPC,
		},
		Status: corev1.PodStatus{
			HostIP:    pod.Status.HostIP,
			PodIP:     pod.Status.PodIP,
			PodIPs:    pod.Status.PodIPs,
			StartTime: pod.Status.StartTime,
		},
	}
}
