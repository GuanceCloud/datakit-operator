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

func TestInjectLogfwd(t *testing.T) {
	t.Run("inject with instances config and volume reuse", func(t *testing.T) {
		// 设置 rule 配置（用于 instances 测试）
		originalFunc := logfwdMatchNamespaceOrLabelsForConfig
		logfwdMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.LogfwdRule) {
			return true, &config.LogfwdRule{
				InjectRule: config.InjectRule{
					Image: "pubrepo.guance.com/datakit-operator/logfwd-testing:v1.0.1",
					Envs: []struct{ Key, Value string }{
						{"LOGFWD_POD_NAME", "{fieldRef:metadata.name}"},
						{"LOGFWD_POD_NAMESPACE", "{fieldRef:metadata.namespace}"},
						{"LOGFWD_GLOBAL_SERVICE", "{fieldRef:metadata.labels['app']}"},
					},
					Resources: config.ResourceRequirements{
						Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
						Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
					},
				},
			}
		}
		defer func() {
			logfwdMatchNamespaceOrLabelsForConfig = originalFunc
		}()

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

		changed, err := InjectLogfwdToPod("", pod.Name, pod)
		if !assert.NoError(t, err) || !assert.True(t, changed) {
			return
		}

		// Verify container injection
		if !assert.Len(t, pod.Spec.Containers, 2) {
			return
		}
		assert.Equal(t, logfwdContainerName, pod.Spec.Containers[1].Name)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/logfwd-testing:v1.0.1", pod.Spec.Containers[1].Image)

		// Verify env vars
		if !assert.Len(t, pod.Spec.Containers[1].Env, 4) {
			return
		}
		assert.Equal(t, logfwdJSONConfigKey, pod.Spec.Containers[1].Env[3].Name)
		assert.Equal(t, instancesCompact, pod.Spec.Containers[1].Env[3].Value)

		// Verify volume reuse and new volume creation (including pod info volume)
		if !assert.Len(t, pod.Spec.Volumes, 3) { // original + new volume + pod info volume
			return
		}
		assert.Equal(t, "exist-mount", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "datakit-logfwd-volume-1", pod.Spec.Volumes[1].Name)
		assert.Equal(t, "datakit-pod-info", pod.Spec.Volumes[2].Name)

		// Verify volume mounts
		assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 2)       // original + new mount
		if !assert.Len(t, pod.Spec.Containers[1].VolumeMounts, 3) { // reused + new mount + pod info
			return
		}
		assert.Equal(t, "datakit-pod-info", pod.Spec.Containers[1].VolumeMounts[2].Name)
		assert.Equal(t, "/etc/podinfo", pod.Spec.Containers[1].VolumeMounts[2].MountPath)

		afterFirstInjection := pod.DeepCopy()
		changed, err = InjectLogfwdToPod("", pod.Name, pod)
		assert.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, afterFirstInjection, pod)
	})

	t.Run("inject with log_configs from rule", func(t *testing.T) {
		const logConfigsConfig = `[
    {
        "path": "/var/log/app/*.log"
    }
]`
		const logConfigsCompact = `[{"path":"/var/log/app/*.log"}]`

		// 设置 rule 配置
		originalFunc := logfwdMatchNamespaceOrLabelsForConfig
		logfwdMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.LogfwdRule) {
			return true, &config.LogfwdRule{
				InjectRule: config.InjectRule{
					Image: "pubrepo.guance.com/datakit-operator/logfwd-testing:v1.0.1",
					Envs: []struct{ Key, Value string }{
						{Key: "FIRST", Value: "first"},
						{Key: logfwdLogConfigsKey, Value: "stale"},
						{Key: "LAST", Value: "$(FIRST)"},
					},
					Resources: config.ResourceRequirements{
						Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
						Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
					},
				},
				LogConfigs: logConfigsConfig,
			}
		}
		defer func() {
			logfwdMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-log-configs-pod", map[string]string{
			logfwdEnabledAnnotationKey: "true",
		})

		changed, err := InjectLogfwdToPod("", pod.Name, pod)
		if !assert.NoError(t, err) || !assert.True(t, changed) {
			return
		}

		// Verify container injection
		if !assert.Len(t, pod.Spec.Containers, 2) {
			return
		}
		assert.Equal(t, logfwdContainerName, pod.Spec.Containers[1].Name)

		// Verify env vars contain log configs
		assert.Equal(t, []string{"FIRST", logfwdLogConfigsKey, "LAST"}, envNames(pod.Spec.Containers[1].Env))
		logConfigs, ok := findEnv(pod.Spec.Containers[1].Env, logfwdLogConfigsKey)
		if !assert.True(t, ok) {
			return
		}
		assert.Equal(t, logConfigsCompact, logConfigs.Value)

		// Verify volume and mount creation (including pod info volume)
		if !assert.Len(t, pod.Spec.Volumes, 2) {
			return
		}
		assert.Equal(t, "datakit-logfwd-volume-0", pod.Spec.Volumes[0].Name)
		assert.Equal(t, "datakit-pod-info", pod.Spec.Volumes[1].Name)
		if !assert.Len(t, pod.Spec.Containers[1].VolumeMounts, 2) {
			return
		}
		assert.Equal(t, "datakit-pod-info", pod.Spec.Containers[1].VolumeMounts[1].Name)
		assert.Equal(t, "/etc/podinfo", pod.Spec.Containers[1].VolumeMounts[1].MountPath)
	})

}
