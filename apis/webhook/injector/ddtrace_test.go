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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// createTestPodWithContainers creates a test pod with specified containers
func createTestPodWithContainers(name string, annotations map[string]string, containers []corev1.Container) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: containers,
		},
	}
}

func TestInjectDDTrace(t *testing.T) {
	t.Run("inject ddtrace to single container", func(t *testing.T) {
		originalFunc := ddtraceMatchNamespaceOrLabelsForConfig
		ddtraceMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				Language: "java",
				Image:    "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
				Envs: []struct{ Key, Value string }{
					{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
					{"DD_TAGS", "host:node-02,system:linux"},
					{"POD_NAME", "{fieldRef:metadata.name}"},
					{"SERVICE_NOT", "{fieldRef:metadata.annotations['hello-$$$']}"},
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

		pod := createTestPodWithContainers("test-ddtrace-single", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		}, []corev1.Container{
			{Name: "app", Image: "nginx:1.22"},
		})

		err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Verify container modifications
		assert.Len(t, pod.Spec.Containers, 1)
		container := pod.Spec.Containers[0]

		// Check volume mount
		assert.Len(t, container.VolumeMounts, 1)
		assert.Equal(t, "datakit-auto-instrument", container.VolumeMounts[0].Name)
		assert.Equal(t, "/datadog-lib", container.VolumeMounts[0].MountPath)

		// Check JAVA_TOOL_OPTIONS env var
		found := false
		for _, env := range container.Env {
			if env.Name == "JAVA_TOOL_OPTIONS" {
				assert.Equal(t, " -javaagent:/datadog-lib/dd-java-agent.jar", env.Value)
				found = true
				break
			}
		}
		assert.True(t, found, "JAVA_TOOL_OPTIONS should be set")

		// Check init container
		assert.Len(t, pod.Spec.InitContainers, 1)
		initContainer := pod.Spec.InitContainers[0]
		assert.Equal(t, "datakit-lib-init", initContainer.Name)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1", initContainer.Image)
		assert.NotEmpty(t, initContainer.Resources.Requests)
		assert.NotEmpty(t, initContainer.Resources.Limits)

		// Check volumes
		assert.Len(t, pod.Spec.Volumes, 1)
		assert.Equal(t, "datakit-auto-instrument", pod.Spec.Volumes[0].Name)
	})

	t.Run("inject ddtrace to multiple containers with merged DD_TAGS", func(t *testing.T) {
		originalFunc := ddtraceMatchNamespaceOrLabelsForConfig
		ddtraceMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				Language: "java",
				Image:    "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
				Envs: []struct{ Key, Value string }{
					{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
					{"DD_TAGS", "host:node-02,system:linux"},
					{"POD_NAME", "{fieldRef:metadata.name}"},
					{"SERVICE_NOT", "{fieldRef:metadata.annotations['hello-$$$']}"},
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

		pod := createTestPodWithContainers("test-ddtrace-multi", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		}, []corev1.Container{
			{
				Name:  "app1",
				Image: "nginx:1.22",
				Env: []corev1.EnvVar{
					{Name: "DD_TAGS", Value: "host:node-01"},
				},
			},
			{
				Name:  "app2",
				Image: "nginx:1.22",
				Env:   []corev1.EnvVar{},
			},
		})

		err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.Containers, 2)

		// Check first container - existing DD_TAGS should be merged
		container1 := pod.Spec.Containers[0]
		found := false
		for _, env := range container1.Env {
			if env.Name == "DD_TAGS" {
				assert.Equal(t, "host:node-01,system:linux", env.Value)
				found = true
				break
			}
		}
		assert.True(t, found, "DD_TAGS should be merged for container1")

		// Check second container - should get default DD_TAGS
		container2 := pod.Spec.Containers[1]
		found = false
		for _, env := range container2.Env {
			if env.Name == "DD_TAGS" {
				assert.Equal(t, "host:node-02,system:linux", env.Value)
				found = true
				break
			}
		}
		assert.True(t, found, "DD_TAGS should be set for container2")
	})
}

func TestInjectDDTraceEdgeCases(t *testing.T) {
	t.Run("return error for nil pod", func(t *testing.T) {
		err := InjectDDTraceToPod("", "test-pod", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot inject ddtrace-lib into nil pod")
	})
}
