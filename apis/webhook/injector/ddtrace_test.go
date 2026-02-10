// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	corev1 "k8s.io/api/core/v1"
)

func TestInjectDDTrace(t *testing.T) {
	t.Run("basic injection", func(t *testing.T) {
		originalFunc := ddtraceMatchNamespaceOrLabelsForConfig
		ddtraceMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				Language: "java",
				Image:    "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
				Envs: []struct{ Key, Value string }{
					{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
					{"DD_TAGS", "host:node-02,system:linux"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}
		}
		defer func() {
			ddtraceMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})

		err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "datakit-lib-init", pod.Spec.InitContainers[0].Name)
		assert.Len(t, pod.Spec.Volumes, 1)
		assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 1)
	})

	t.Run("CheckAnnotation=true with annotation", func(t *testing.T) {
		originalFunc := ddtraceMatchNamespaceOrLabelsForConfig
		ddtraceMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				Language:        "java",
				CheckAnnotation: true,
				Image:           "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
				Envs: []struct{ Key, Value string }{
					{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}
		}
		defer func() {
			ddtraceMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-check-annotation", map[string]string{
			ddtraceEnabledAnnotationKey:          "true",
			"admission.datakit/java-lib.version": "v2.0.0",
		})

		err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/java-lib-testing:v2.0.0", pod.Spec.InitContainers[0].Image)
	})

	t.Run("CheckAnnotation=true without annotation", func(t *testing.T) {
		originalFunc := ddtraceMatchNamespaceOrLabelsForConfig
		ddtraceMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				Language:        "java",
				CheckAnnotation: true,
				Image:           "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
				Envs: []struct{ Key, Value string }{
					{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}
		}
		defer func() {
			ddtraceMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-no-annotation", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})

		err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 0)
		assert.Len(t, pod.Spec.Volumes, 0)
	})

	t.Run("CheckAnnotation=false", func(t *testing.T) {
		originalFunc := ddtraceMatchNamespaceOrLabelsForConfig
		ddtraceMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				Language:        "java",
				CheckAnnotation: false,
				Image:           "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
				Envs: []struct{ Key, Value string }{
					{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}
		}
		defer func() {
			ddtraceMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-no-check", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})

		err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1", pod.Spec.InitContainers[0].Image)
	})

	t.Run("skip when annotation is false", func(t *testing.T) {
		originalFunc := ddtraceMatchNamespaceOrLabelsForConfig
		ddtraceMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				Language: "java",
				Image:    "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
			}
		}
		defer func() {
			ddtraceMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-disabled", map[string]string{
			ddtraceEnabledAnnotationKey: "false",
		})

		err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 0)
		assert.Len(t, pod.Spec.Volumes, 0)
	})

	t.Run("skip when init container already exists", func(t *testing.T) {
		originalFunc := ddtraceMatchNamespaceOrLabelsForConfig
		ddtraceMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				Language: "java",
				Image:    "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
			}
		}
		defer func() {
			ddtraceMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-exists", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})
		pod.Spec.InitContainers = []corev1.Container{
			{Name: ddtraceInitContainerName, Image: "existing-image"},
		}

		err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "existing-image", pod.Spec.InitContainers[0].Image)
	})

	t.Run("nil pod error", func(t *testing.T) {
		err := InjectDDTraceToPod("", "test-pod", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot inject ddtrace-lib into nil pod")
	})
}
