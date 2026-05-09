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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInjectFlameshot(t *testing.T) {
	t.Run("inject flameshot with basic configuration", func(t *testing.T) {
		originalFunc := flameshotMatchNamespaceOrLabelsForConfig
		flameshotMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				Image: "pubrepo.guance.com/datakit-operator/flameshot-testing:v1.0.0",
				Envs: []struct{ Key, Value string }{
					{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
					{"DK_AGENT_PORT", "9529"},
					{"FLAMESHOT_PROFILING_PATH", "/flameshot-data"},
					{"FLAMESHOT_HTTP_LOCAL_PORT", "8089"},
				},
				Processes: "[{\"service\":\"jfr-parser\"}]",
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}
		}
		defer func() {
			flameshotMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-flameshot-pod",
				Annotations: map[string]string{
					flameshotEnabledAnnotationKey: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "nginx", Image: "nginx:1.22"},
				},
			},
		}

		_, err := InjectFlameshotToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Verify pod spec modifications
		assert.True(t, *pod.Spec.ShareProcessNamespace)
		assert.Equal(t, corev1.RestartPolicyAlways, pod.Spec.RestartPolicy)

		// Verify containers
		assert.Len(t, pod.Spec.Containers, 2)

		// Check main container volume mounts
		mainContainer := pod.Spec.Containers[0]
		assert.Len(t, mainContainer.VolumeMounts, 1)
		assert.Equal(t, flameshotProfilingVolumeName, mainContainer.VolumeMounts[0].Name)

		// Check flameshot container
		flameshotContainer := pod.Spec.Containers[1]
		assert.Equal(t, flameshotContainerName, flameshotContainer.Name)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/flameshot-testing:v1.0.0", flameshotContainer.Image)
		assert.Equal(t, []string{"/flameshot/flameshot"}, flameshotContainer.Command)

		// Check flameshot container ports
		assert.Len(t, flameshotContainer.Ports, 1)
		assert.Equal(t, flameshotHTTPPortName, flameshotContainer.Ports[0].Name)
		assert.Equal(t, int32(8089), flameshotContainer.Ports[0].ContainerPort)
		assert.Equal(t, corev1.ProtocolTCP, flameshotContainer.Ports[0].Protocol)

		// Check flameshot container security context
		assert.NotNil(t, flameshotContainer.SecurityContext)
		assert.NotNil(t, flameshotContainer.SecurityContext.Capabilities)
		assert.Contains(t, flameshotContainer.SecurityContext.Capabilities.Add, corev1.Capability("SYS_PTRACE"))

		// Check resources
		assert.Equal(t, resource.MustParse("200m"), flameshotContainer.Resources.Limits[corev1.ResourceCPU])
		assert.Equal(t, resource.MustParse("128Mi"), flameshotContainer.Resources.Limits[corev1.ResourceMemory])
		assert.Equal(t, resource.MustParse("100m"), flameshotContainer.Resources.Requests[corev1.ResourceCPU])
		assert.Equal(t, resource.MustParse("64Mi"), flameshotContainer.Resources.Requests[corev1.ResourceMemory])

		// Check environment variables
		assert.GreaterOrEqual(t, len(flameshotContainer.Env), 4)
		envMap := make(map[string]string)
		for _, env := range flameshotContainer.Env {
			envMap[env.Name] = env.Value
		}
		assert.Equal(t, "datakit-service.datakit.svc", envMap["DK_AGENT_HOST"])
		assert.Equal(t, "9529", envMap["DK_AGENT_PORT"])
		assert.Equal(t, "/flameshot-data", envMap["FLAMESHOT_PROFILING_PATH"])
		assert.Equal(t, "8089", envMap["FLAMESHOT_HTTP_LOCAL_PORT"])
	})

	t.Run("verify volume creation and mounting", func(t *testing.T) {
		originalFunc := flameshotMatchNamespaceOrLabelsForConfig
		flameshotMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				Image: "pubrepo.guance.com/datakit-operator/flameshot-testing:v1.0.0",
				Envs: []struct{ Key, Value string }{
					{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
					{"DK_AGENT_PORT", "9529"},
					{"FLAMESHOT_PROFILING_PATH", "/flameshot-data"},
					{"FLAMESHOT_HTTP_LOCAL_PORT", "8089"},
				},
				Processes: "[{\"service\":\"jfr-parser\"}]",
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}
		}
		defer func() {
			flameshotMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-volumes-pod",
				Annotations: map[string]string{
					flameshotEnabledAnnotationKey: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "app:1.0"},
				},
			},
		}

		_, err := InjectFlameshotToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Check volumes are created correctly
		assert.Len(t, pod.Spec.Volumes, 1)
		assert.Equal(t, flameshotProfilingVolumeName, pod.Spec.Volumes[0].Name)
		assert.NotNil(t, pod.Spec.Volumes[0].VolumeSource.EmptyDir)

		// Check volume mounts in flameshot container
		flameshotContainer := pod.Spec.Containers[1]
		assert.Len(t, flameshotContainer.VolumeMounts, 1)
		assert.Equal(t, flameshotProfilingVolumeName, flameshotContainer.VolumeMounts[0].Name)
		assert.Equal(t, "/flameshot-data", flameshotContainer.VolumeMounts[0].MountPath)
	})

	t.Run("add prometheus annotations when enabled", func(t *testing.T) {
		originalFunc := flameshotMatchNamespaceOrLabelsForConfig
		flameshotMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				Image: "pubrepo.guance.com/datakit-operator/flameshot-testing:v1.0.0",
				Envs: []struct{ Key, Value string }{
					{"FLAMESHOT_PROFILING_PATH", "/flameshot-data"},
					{"FLAMESHOT_HTTP_LOCAL_PORT", "8089"},
				},
				Processes:                   "[{\"service\":\"jfr-parser\"}]",
				EnablePrometheusAnnotations: true,
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}
		}
		defer func() { flameshotMatchNamespaceOrLabelsForConfig = originalFunc }()

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-prometheus-pod",
				Annotations: map[string]string{flameshotEnabledAnnotationKey: "true"},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "app", Image: "app:1.0"}},
			},
		}

		_, err := InjectFlameshotToPod("", pod.Name, pod)
		assert.NoError(t, err)
		assert.Equal(t, "true", pod.Annotations["prometheus.io/scrape"])
		assert.Equal(t, "8089", pod.Annotations["prometheus.io/port"])
		assert.Equal(t, "http", pod.Annotations["prometheus.io/scheme"])
		assert.Equal(t, "/metrics", pod.Annotations["prometheus.io/path"])
		assert.Equal(t, "flameshot", pod.Annotations["prometheus.io/param_measurement"])
	})
}

func TestInjectFlameshotEdgeCases(t *testing.T) {
	t.Run("return error for nil pod", func(t *testing.T) {
		_, err := InjectFlameshotToPod("", "test-pod", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot inject flameshot into nil pod")
	})
}
