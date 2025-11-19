// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package mutator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// setupLoggingTestFunctions sets up test function variables for logging tests
func setupLoggingTestFunctions(configStr string) {
	loggingMatchNamespaceOrLabelsForConfig = func(_ string, _ map[string]string) string { return configStr }
}

// createTestPod creates a basic test pod with given name and optional existing mounts
func createTestPod(name string, existingMounts []corev1.VolumeMount, existingVolumes []corev1.Volume) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:         "app",
					Image:        "nginx:1.22",
					VolumeMounts: existingMounts,
				},
			},
			Volumes: existingVolumes,
		},
	}
	return pod
}

func TestMutateLogging(t *testing.T) {
	t.Run("mutate pod with logging configuration and volume reuse", func(t *testing.T) {
		const configStr = `[{"disable":false,"type":"file","path":"/var/log/opt/**/*log","source":"logging-var"}, {"disable":false,"type":"file","path":"/tmp/opt/log","source":"logging-tmp"}]`
		setupLoggingTestFunctions(configStr)

		existingMounts := []corev1.VolumeMount{
			{Name: "shared-logs", MountPath: "/var/log/opt", ReadOnly: false},
		}
		existingVolumes := []corev1.Volume{
			{Name: "shared-logs", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		}

		pod := createTestPod("test-logging-basic", existingMounts, existingVolumes)

		err := MutateLoggingToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Verify annotation is added
		assert.Contains(t, pod.Annotations, "datakit/logs")
		assert.Equal(t, configStr, pod.Annotations["datakit/logs"])

		// Verify volumes are created correctly
		assert.Len(t, pod.Spec.Volumes, 2) // existing + new volume
		assert.Equal(t, "shared-logs", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "datakit-logs-volume-1", pod.Spec.Volumes[1].Name)

		// Verify volume mounts
		container := pod.Spec.Containers[0]
		assert.Len(t, container.VolumeMounts, 2) // existing + new mount
		assert.Equal(t, "shared-logs", container.VolumeMounts[0].Name)
		assert.Equal(t, "datakit-logs-volume-1", container.VolumeMounts[1].Name)
		assert.Equal(t, "/tmp/opt", container.VolumeMounts[1].MountPath)
	})

	t.Run("mutate pod with single logging configuration", func(t *testing.T) {
		const configStr = `[{"disable":false,"type":"file","path":"/var/log/app.log","source":"app-log"}]`
		setupLoggingTestFunctions(configStr)

		pod := createTestPod("test-single-config", nil, nil)

		err := MutateLoggingToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Verify annotation is added
		assert.Contains(t, pod.Annotations, "datakit/logs")
		assert.Equal(t, configStr, pod.Annotations["datakit/logs"])

		// Verify single volume is created
		assert.Len(t, pod.Spec.Volumes, 1)
		assert.Equal(t, "datakit-logs-volume-0", pod.Spec.Volumes[0].Name)

		// Verify single volume mount
		container := pod.Spec.Containers[0]
		assert.Len(t, container.VolumeMounts, 1)
		assert.Equal(t, "datakit-logs-volume-0", container.VolumeMounts[0].Name)
		assert.Equal(t, "/var/log", container.VolumeMounts[0].MountPath)
	})
}

func TestMutateLoggingEdgeCases(t *testing.T) {
	t.Run("skip mutation when datakit/logs annotation already exists", func(t *testing.T) {
		const configStr = `[{"disable":false,"type":"file","path":"/var/log/opt/**/*log","source":"logging-var"}]`
		setupLoggingTestFunctions(configStr)

		existingAnnotations := map[string]string{"datakit/logs": "existing-config"}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-existing-annotation",
				Annotations: existingAnnotations,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "nginx:1.22"},
				},
			},
		}

		err := MutateLoggingToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Should not modify existing annotation
		assert.Equal(t, existingAnnotations, pod.Annotations)
		// Should not add any volumes or mounts
		assert.Empty(t, pod.Spec.Volumes)
		assert.Empty(t, pod.Spec.Containers[0].VolumeMounts)
	})

	t.Run("skip mutation when no config is returned", func(t *testing.T) {
		loggingMatchNamespaceOrLabelsForConfig = func(_ string, _ map[string]string) string { return "" }

		pod := createTestPod("test-no-config", nil, nil)

		err := MutateLoggingToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Should not add annotation or volumes
		assert.Empty(t, pod.Annotations)
		assert.Empty(t, pod.Spec.Volumes)
		assert.Empty(t, pod.Spec.Containers[0].VolumeMounts)
	})

	t.Run("return error for nil pod", func(t *testing.T) {
		err := MutateLoggingToPod("", "test-pod", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot inject ddtrace-lib into nil pod")
	})
}
