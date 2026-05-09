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

func TestInjectProfiler(t *testing.T) {
	t.Run("basic injection with CheckAnnotation=true and annotation", func(t *testing.T) {
		originalFunc := profilerMatchNamespaceOrLabelsForConfig
		profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				CheckAnnotation: true,
				Images: map[string]string{
					config.DeprecatedProfilerJavaImageKey: "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1",
				},
				Envs: []struct{ Key, Value string }{
					{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
					{"DK_AGENT_PORT", "9529"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}
		}
		defer func() {
			profilerMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-profiler-java", map[string]string{
			profilerEnabledAnnotationKey:              "true",
			"admission.datakit/java-profiler.version": "v2.0.0",
		})

		_, err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.Containers, 2)
		assert.Equal(t, profilerContainerName, pod.Spec.Containers[1].Name)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/java-profiler-testing:v2.0.0", pod.Spec.Containers[1].Image)
		assert.Len(t, pod.Spec.Volumes, 3)
		assert.NotNil(t, pod.Spec.ShareProcessNamespace)
		assert.True(t, *pod.Spec.ShareProcessNamespace)
	})

	t.Run("CheckAnnotation=true without annotation", func(t *testing.T) {
		originalFunc := profilerMatchNamespaceOrLabelsForConfig
		profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				CheckAnnotation: true,
				Images: map[string]string{
					config.DeprecatedProfilerJavaImageKey: "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1",
				},
			}
		}
		defer func() {
			profilerMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-profiler-no-annotation", map[string]string{
			profilerEnabledAnnotationKey: "true",
		})

		_, err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.Containers, 1)
		assert.Len(t, pod.Spec.Volumes, 0)
	})

	t.Run("CheckAnnotation=false with language in rule", func(t *testing.T) {
		originalFunc := profilerMatchNamespaceOrLabelsForConfig
		profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				Language:        "java",
				CheckAnnotation: false,
				Images: map[string]string{
					config.DeprecatedProfilerJavaImageKey: "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1",
				},
				Envs: []struct{ Key, Value string }{
					{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}
		}
		defer func() {
			profilerMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-profiler-no-check", map[string]string{
			profilerEnabledAnnotationKey: "true",
		})

		_, err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.Containers, 2)
		assert.Equal(t, profilerContainerName, pod.Spec.Containers[1].Name)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1", pod.Spec.Containers[1].Image)
	})

	t.Run("python profiler with annotation", func(t *testing.T) {
		originalFunc := profilerMatchNamespaceOrLabelsForConfig
		profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				CheckAnnotation: true,
				Images: map[string]string{
					config.DeprecatedProfilerPythonImageKey: "pubrepo.guance.com/datakit-operator/python-profiler-testing:v1.0.1",
				},
				Envs: []struct{ Key, Value string }{
					{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}
		}
		defer func() {
			profilerMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-profiler-python", map[string]string{
			profilerEnabledAnnotationKey:                "true",
			"admission.datakit/python-profiler.version": "latest",
		})

		_, err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.Containers, 2)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/python-profiler-testing:latest", pod.Spec.Containers[1].Image)
	})

	t.Run("skip when profiler.enabled is false", func(t *testing.T) {
		originalFunc := profilerMatchNamespaceOrLabelsForConfig
		profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				CheckAnnotation: true,
				Images: map[string]string{
					config.DeprecatedProfilerJavaImageKey: "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1",
				},
			}
		}
		defer func() {
			profilerMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-profiler-disabled", map[string]string{
			profilerEnabledAnnotationKey: "false",
		})

		_, err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.Containers, 1)
		assert.Len(t, pod.Spec.Volumes, 0)
	})

	t.Run("skip when container already exists", func(t *testing.T) {
		originalFunc := profilerMatchNamespaceOrLabelsForConfig
		profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				CheckAnnotation: true,
				Images: map[string]string{
					config.DeprecatedProfilerJavaImageKey: "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1",
				},
			}
		}
		defer func() {
			profilerMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-profiler-exists", map[string]string{
			profilerEnabledAnnotationKey: "true",
		})
		pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
			Name:  profilerContainerName,
			Image: "existing-profiler-image",
		})

		_, err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.Containers, 2)
		assert.Equal(t, "existing-profiler-image", pod.Spec.Containers[1].Image)
	})

	t.Run("CheckAnnotation=false without language", func(t *testing.T) {
		originalFunc := profilerMatchNamespaceOrLabelsForConfig
		profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				Language:        "",
				CheckAnnotation: false,
				Images: map[string]string{
					config.DeprecatedProfilerJavaImageKey: "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1",
				},
			}
		}
		defer func() {
			profilerMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-profiler-no-lang", map[string]string{
			profilerEnabledAnnotationKey: "true",
		})

		_, err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.Containers, 1)
		assert.Len(t, pod.Spec.Volumes, 0)
	})

	t.Run("nil pod error", func(t *testing.T) {
		_, err := InjectProfilerToPod("", "test-pod", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot inject profiler into nil pod")
	})
}
