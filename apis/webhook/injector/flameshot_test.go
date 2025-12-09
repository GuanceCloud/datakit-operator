// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// setupFlameshotTestFunctions sets up test function variables for flameshot tests
func setupFlameshotTestFunctions() {
	// Test constants
	const testFlameshotImage = "pubrepo.guance.com/datakit-operator/flameshot-testing:v1.0.0"
	testFlameshotEnvs := []struct{ Key, Value string }{
		{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
		{"DK_AGENT_PORT", "9529"},
		{"FLAMESHOT_PROFILING_PATH", "/flameshot-data"},
		{"FLAMESHOT_HTTP_LOCAL_ADDRESS", "0.0.0.0:8089"},
	}

	flameshotImage = func() string { return testFlameshotImage }
	flameshotResourceRequests = func() (string, string) { return "100m", "64Mi" }
	flameshotResourceLimits = func() (string, string) { return "200m", "128Mi" }
	flameshotEnvs = func() []struct{ Key, Value string } {
		return testFlameshotEnvs
	}
}

func TestInjectFlameshot(t *testing.T) {
	t.Run("inject flameshot with basic configuration", func(t *testing.T) {
		setupFlameshotTestFunctions()

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-flameshot-pod",
				Annotations: map[string]string{
					"admission.datakit/flameshot.enabled":   "true",
					"admission.datakit/flameshot.processes": "nginx,app",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "nginx", Image: "nginx:1.22"},
				},
			},
		}

		err := InjectFlameshotToPod("", pod.Name, pod)
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
	})

	t.Run("verify volume creation and mounting", func(t *testing.T) {
		setupFlameshotTestFunctions()

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-volumes-pod",
				Annotations: map[string]string{
					"admission.datakit/flameshot.enabled":   "true",
					"admission.datakit/flameshot.processes": "app",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "app:1.0"},
				},
			},
		}

		err := InjectFlameshotToPod("", pod.Name, pod)
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
}

func TestInjectFlameshotEdgeCases(t *testing.T) {
	t.Run("return error for nil pod", func(t *testing.T) {
		err := InjectFlameshotToPod("", "test-pod", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot inject flameshot into nil pod")
	})
}
