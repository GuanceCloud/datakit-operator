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

// setupProfilerTestFunctions sets up test function variables for profiler tests
func setupProfilerTestFunctions() {
	// Test constants
	const testProfilerImage = "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1"
	testProfilerEnvs := []struct{ Key, Value string }{
		{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
		{"DK_AGENT_PORT", "9529"},
		{"POD_NAME", "{fieldRef:metadata.name}"},
	}

	profilerJavaImage = func() string { return testProfilerImage }
	profilerResourceLimits = func() (string, string) { return "200m", "128Mi" }
	profilerEnvs = func() []struct{ Key, Value string } {
		return testProfilerEnvs
	}
}

func TestInjectProfiler(t *testing.T) {
	t.Run("inject profiler with basic configuration", func(t *testing.T) {
		setupProfilerTestFunctions()

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-profiler-pod",
				Annotations: map[string]string{"admission.datakit/java-profiler.version": "latest"},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "nginx", Image: "nginx:1.22"},
				},
			},
		}

		err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Verify pod spec modifications
		assert.True(t, *pod.Spec.ShareProcessNamespace)
		assert.Equal(t, corev1.RestartPolicyAlways, pod.Spec.RestartPolicy)

		// Verify containers
		assert.Len(t, pod.Spec.Containers, 2)

		// Check main container volume mounts
		mainContainer := pod.Spec.Containers[0]
		assert.Len(t, mainContainer.VolumeMounts, 3)

		expectedMounts := []string{"datakit-profiler-volume", "tmp", "timezone"}
		for i, mount := range mainContainer.VolumeMounts {
			assert.Equal(t, expectedMounts[i], mount.Name)
		}

		// Check profiler container
		profilerContainer := pod.Spec.Containers[1]
		assert.Equal(t, "datakit-profiler", profilerContainer.Name)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/java-profiler-testing:latest", profilerContainer.Image)
		assert.Equal(t, "/app/datakit-profiler", profilerContainer.WorkingDir)
		assert.Equal(t, []string{"bash", "cmd.sh"}, profilerContainer.Command)

		// Check profiler container env vars
		assert.Len(t, profilerContainer.Env, 3)

		// Check profiler container security context
		assert.NotNil(t, profilerContainer.SecurityContext)
		assert.NotNil(t, profilerContainer.SecurityContext.Capabilities)
		assert.Contains(t, profilerContainer.SecurityContext.Capabilities.Add, corev1.Capability("SYS_PTRACE"))
		assert.Contains(t, profilerContainer.SecurityContext.Capabilities.Add, corev1.Capability("SYS_ADMIN"))

		// Check resources
		assert.Equal(t, resource.MustParse("200m"), profilerContainer.Resources.Limits[corev1.ResourceCPU])
		assert.Equal(t, resource.MustParse("128Mi"), profilerContainer.Resources.Limits[corev1.ResourceMemory])
	})

	t.Run("verify volume creation and mounting", func(t *testing.T) {
		setupProfilerTestFunctions()

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-volumes-pod",
				Annotations: map[string]string{"admission.datakit/java-profiler.version": "latest"},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "app:1.0"},
				},
			},
		}

		err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Check volumes are created correctly
		assert.Len(t, pod.Spec.Volumes, 3)

		// Check volume names and types
		expectedVolumes := map[string]bool{
			"datakit-profiler-volume": false,
			"tmp":                     false,
			"timezone":                false,
		}

		for _, volume := range pod.Spec.Volumes {
			assert.Contains(t, expectedVolumes, volume.Name)

			switch volume.Name {
			case "datakit-profiler-volume", "tmp":
				assert.NotNil(t, volume.VolumeSource.EmptyDir)
			case "timezone":
				assert.NotNil(t, volume.VolumeSource.HostPath)
				assert.Equal(t, "/etc/localtime", volume.VolumeSource.HostPath.Path)
			}
		}

		// Check volume mounts in profiler container
		profilerContainer := pod.Spec.Containers[1]
		assert.Len(t, profilerContainer.VolumeMounts, 3)

		expectedMountPaths := map[string]string{
			"datakit-profiler-volume": "/app/datakit-profiler",
			"tmp":                     "/tmp",
			"timezone":                "/etc/localtime",
		}

		for _, mount := range profilerContainer.VolumeMounts {
			assert.Contains(t, expectedVolumes, mount.Name)
			expectedPath, exists := expectedMountPaths[mount.Name]
			assert.True(t, exists, "mount path should be defined for volume %s", mount.Name)
			assert.Equal(t, expectedPath, mount.MountPath)
		}
	})
}

func TestInjectProfilerEdgeCases(t *testing.T) {
	t.Run("return error for nil pod", func(t *testing.T) {
		err := InjectProfilerToPod("", "test-pod", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot inject profiler into nil pod")
	})
}
