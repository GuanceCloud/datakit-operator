// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// setupTestFunctions sets up the test function variables for logfwd injection tests
func setupTestFunctions() {
	logfwdImage = func() string { return "pubrepo.guance.com/datakit-operator/logfwd-testing:v1.0.1" }
	logfwdResourceRequests = func() (string, string) { return "100m", "64Mi" }
	logfwdResourceLimits = func() (string, string) { return "200m", "128Mi" }

	logfwdEnvs = func() []struct{ Key, Value string } {
		return []struct{ Key, Value string }{
			{"LOGFWD_POD_NAME", "{fieldRef:metadata.name}"},
			{"LOGFWD_POD_NAMESPACE", "{fieldRef:metadata.namespace}"},
			{"LOGFWD_GLOBAL_SERVICE", "{fieldRef:metadata.labels['app']}"},
		}
	}
}

// createTestPod creates a basic test pod with given name and annotations
func createTestPod(name string, annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "nginx", Image: "nginx:1.22"},
			},
		},
	}
}

func TestInjectLogfwd(t *testing.T) {
	t.Run("inject with instances config and volume reuse", func(t *testing.T) {
		setupTestFunctions()

		const instancesConfig = `[
    {
        "datakit_addr": "datakit-service.datakit.svc:9533",
        "loggings": [
            {
                "logfiles": ["/var/log/nginx/success/*.log"],
                "source": "nginx-success",
                "tags": {
                    "key01": "value01"
                }
            },
            {
                "logfiles": ["/var/log/nginx/error/*.log"],
                "source": "nginx-error",
                "pipeline": "nginx-error.p"
            }
        ]
    }
]`
		const instancesCompact = `[{"datakit_addr":"datakit-service.datakit.svc:9533","loggings":[{"logfiles":["/var/log/nginx/success/*.log"],"source":"nginx-success","tags":{"key01":"value01"}},{"logfiles":["/var/log/nginx/error/*.log"],"source":"nginx-error","pipeline":"nginx-error.p"}]}]`

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-instances-pod",
				Annotations: map[string]string{logfwdEnabledAnnotationKey: "true", logfwdInstancesAnnotationKey: instancesConfig},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx:1.22",
						VolumeMounts: []corev1.VolumeMount{
							{Name: "exist-mount", MountPath: "/var/log/nginx/success", ReadOnly: false},
						},
					},
				},
				Volumes: []corev1.Volume{
					{Name: "exist-mount", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				},
			},
		}

		err := InjectLogfwdToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Verify container injection
		assert.Len(t, pod.Spec.Containers, 2)
		assert.Equal(t, logfwdContainerName, pod.Spec.Containers[1].Name)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/logfwd-testing:v1.0.1", pod.Spec.Containers[1].Image)

		// Verify env vars
		assert.Len(t, pod.Spec.Containers[1].Env, 4)
		assert.Equal(t, logfwdJSONConfigKey, pod.Spec.Containers[1].Env[3].Name)
		assert.Equal(t, instancesCompact, pod.Spec.Containers[1].Env[3].Value)

		// Verify volume reuse and new volume creation (including pod info volume)
		assert.Len(t, pod.Spec.Volumes, 3) // original + new volume + pod info volume
		assert.Equal(t, "exist-mount", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "datakit-logfwd-volume-1", pod.Spec.Volumes[1].Name)
		assert.Equal(t, "datakit-pod-info", pod.Spec.Volumes[2].Name)

		// Verify volume mounts
		assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 2) // original + new mount
		assert.Len(t, pod.Spec.Containers[1].VolumeMounts, 3) // reused + new mount + pod info
		assert.Equal(t, "datakit-pod-info", pod.Spec.Containers[1].VolumeMounts[2].Name)
		assert.Equal(t, "/etc/podinfo", pod.Spec.Containers[1].VolumeMounts[2].MountPath)
	})

	t.Run("inject with log_configs", func(t *testing.T) {
		setupTestFunctions()

		const logConfigsConfig = `[
    {
        "path": "/var/log/app/*.log"
    }
]`
		const logConfigsCompact = `[{"path":"/var/log/app/*.log"}]`

		pod := createTestPod("test-log-configs-pod", map[string]string{
			logfwdEnabledAnnotationKey:    "true",
			logfwdLogConfigsAnnotationKey: logConfigsConfig,
		})

		err := InjectLogfwdToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Verify container injection
		assert.Len(t, pod.Spec.Containers, 2)
		assert.Equal(t, logfwdContainerName, pod.Spec.Containers[1].Name)

		// Verify env vars contain log configs
		assert.Len(t, pod.Spec.Containers[1].Env, 4)
		assert.Equal(t, logfwdLogConfigsKey, pod.Spec.Containers[1].Env[3].Name)
		assert.Equal(t, logConfigsCompact, pod.Spec.Containers[1].Env[3].Value)

		// Verify volume and mount creation (including pod info volume)
		assert.Len(t, pod.Spec.Volumes, 2)
		assert.Equal(t, "datakit-logfwd-volume-0", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "datakit-pod-info", pod.Spec.Volumes[1].Name)
		assert.Len(t, pod.Spec.Containers[1].VolumeMounts, 2)
		assert.Equal(t, "datakit-pod-info", pod.Spec.Containers[1].VolumeMounts[1].Name)
		assert.Equal(t, "/etc/podinfo", pod.Spec.Containers[1].VolumeMounts[1].MountPath)
	})
}

func TestInjectLogfwdEdgeCases(t *testing.T) {
	t.Run("return error for nil pod", func(t *testing.T) {
		err := InjectLogfwdToPod("", "test-pod", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot inject logfwd into nil pod")
	})

	t.Run("skip injection when no config provided", func(t *testing.T) {
		setupTestFunctions()

		originalPod := createTestPod("test-pod-no-config", map[string]string{logfwdEnabledAnnotationKey: "true"})
		expectedPod := *originalPod

		err := InjectLogfwdToPod("", originalPod.Name, originalPod)
		assert.NoError(t, err)
		assert.Equal(t, expectedPod, *originalPod)
	})
}
