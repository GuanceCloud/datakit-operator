// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package logging

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/logging/config"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/kubernetes/client"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Response struct {
	Name            string   `json:"name,omitempty"`
	PodTargetLabels []string `json:"pod_target_labels,omitempty"`
	Configs         string   `json:"configs,omitempty"`
	Error           string   `json:"error,omitempty"`
}

// CheckClusterLoggingConfigRBAC 检查是否有 ClusterLoggingConfig 的 RBAC 权限
func CheckClusterLoggingConfigRBAC(k8sClient client.Client) error {
	ctx := context.Background()
	if clientset := k8sClient.Logging(); clientset != nil {
		_, err := clientset.LoggingV1alpha1().ClusterLoggingConfigs().List(ctx, metav1.ListOptions{Limit: 1})
		if errors.IsForbidden(err) || errors.IsUnauthorized(err) {
			return fmt.Errorf("missing RBAC permissions to access ClusterLoggingConfig: %w", err)
		}
		if errors.IsNotFound(err) {
			return fmt.Errorf("ClusterLoggingConfig CRD resource type not found: %w", err)
		}
		if err != nil {
			return fmt.Errorf("cannot list ClusterLoggingConfig: %w", err)
		}
		return nil
	}
	return fmt.Errorf("logging client is nil")
}

// StartLoggingConfigWatcher 启动 logging config watcher
func StartLoggingConfigWatcher(ctx context.Context, k8sClient client.Client) {
	config.StartLoggingConfigWatcher(ctx, k8sClient)
}

func HandleConfigs(c *gin.Context) {
	namespace := c.Query("namespace")
	podName := c.Query("pod_name")
	podLabelsStr := c.Query("pod_labels")

	log.Debugf("config request received: method=%s, namespace=%s, pod_name=%s", c.Request.Method, namespace, podName)

	if namespace == "" || podName == "" {
		c.JSON(http.StatusBadRequest, Response{Error: "namespace, pod_name parameters are required"})
		return
	}

	// 从 cache 中查找匹配的配置
	name, configs, podTargetLabels := config.FindMatchingConfigs(namespace, podName, podLabelsStr)

	if configs == "" {
		log.Infof("no matching config found: namespace=%s, pod_name=%s", namespace, podName)
		c.JSON(http.StatusNotFound, Response{Error: "no matching config found"})
		return
	}

	log.Infof("matching config found: name=%s", name)
	c.JSON(http.StatusOK, Response{Name: name, PodTargetLabels: podTargetLabels, Configs: configs})
}
