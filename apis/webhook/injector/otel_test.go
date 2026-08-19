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

func newTestOTelRule(image string) *config.OTelRule {
	return &config.OTelRule{
		InjectRule: config.InjectRule{
			Image: image,
			Resources: config.ResourceRequirements{
				Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
				Limits:   config.ResourceQuotaConfig{CPU: "500m", Memory: "512Mi"},
			},
		},
		Language: "java",
	}
}

func TestInjectOTelJavaToPod(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		rule := newTestOTelRule("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0")
		rule.Envs = []struct{ Key, Value string }{
			{Key: "OTEL_SERVICE_NAME", Value: "checkout"},
		}
		return true, []*config.OTelRule{rule}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	runAsNonRoot := true
	pod := createTestPod("test-otel-java", map[string]string{otelEnabledAnnotationKey: "true"})
	pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{RunAsNonRoot: &runAsNonRoot}
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: "sidecar", Image: "busybox:1.36"})

	changed, err := InjectOTelToPod("default", pod.Name, pod)
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, changed)

	if assert.Len(t, pod.Spec.InitContainers, 1) {
		initContainer := pod.Spec.InitContainers[0]
		assert.Equal(t, otelInitContainerName, initContainer.Name)
		assert.Equal(t, "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0", initContainer.Image)
		assert.Equal(t, []string{"cp", otelJavaAgentSourcePath, otelJavaAgentPath}, initContainer.Command)
		assert.Equal(t, corev1.PullAlways, initContainer.ImagePullPolicy)
		assert.Equal(t, pod.Spec.Containers[0].SecurityContext, initContainer.SecurityContext)
		assert.Equal(t, "100m", initContainer.Resources.Requests.Cpu().String())
		assert.Equal(t, "64Mi", initContainer.Resources.Requests.Memory().String())
		assert.Equal(t, "500m", initContainer.Resources.Limits.Cpu().String())
		assert.Equal(t, "512Mi", initContainer.Resources.Limits.Memory().String())
	}

	if assert.Len(t, pod.Spec.Volumes, 1) {
		assert.Equal(t, otelVolumeName, pod.Spec.Volumes[0].Name)
		assert.NotNil(t, pod.Spec.Volumes[0].EmptyDir)
	}

	for _, container := range pod.Spec.Containers {
		if assert.Len(t, container.VolumeMounts, 1, container.Name) {
			assert.Equal(t, otelVolumeName, container.VolumeMounts[0].Name)
			assert.Equal(t, otelJavaMountPath, container.VolumeMounts[0].MountPath)
		}
		envs := envValues(container.Env)
		assert.Equal(t, otelJavaAgentOption, envs[otelJavaToolOptionsKey], container.Name)
		assert.Equal(t, "checkout", envs["OTEL_SERVICE_NAME"], container.Name)
	}
}

func TestInjectOTelRequiresVersionAnnotationWhenConfigured(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		rule := newTestOTelRule("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0")
		rule.CheckAnnotation = true
		return true, []*config.OTelRule{rule}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-otel-java-no-version", map[string]string{otelEnabledAnnotationKey: "true"})
	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Empty(t, pod.Spec.InitContainers)
	assert.Empty(t, pod.Spec.Volumes)
}

func TestInjectOTelUsesVersionAnnotation(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		rule := newTestOTelRule("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0")
		rule.CheckAnnotation = true
		return true, []*config.OTelRule{rule}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-otel-java-version", map[string]string{
		otelEnabledAnnotationKey:                  "true",
		"admission.datakit/otel-java-lib.version": "2.28.0",
	})
	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	if assert.Len(t, pod.Spec.InitContainers, 1) {
		assert.Equal(t, "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.28.0", pod.Spec.InitContainers[0].Image)
	}
}

func TestInjectOTelSkipsPodWhenJavaToolOptionsUsesValueFrom(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		return true, []*config.OTelRule{newTestOTelRule("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0")}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-otel-java-value-from", nil)
	pod.Spec.Containers[0].Env = []corev1.EnvVar{
		{
			Name: otelJavaToolOptionsKey,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{Key: "java-tool-options"},
			},
		},
	}
	originalPod := pod.DeepCopy()

	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, originalPod, pod)
}

func TestInjectOTelSkipsPodWithDatadogJavaAgent(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		return true, []*config.OTelRule{newTestOTelRule("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0")}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-otel-java-datadog", nil)
	pod.Spec.Containers[0].Env = []corev1.EnvVar{
		{Name: otelJavaToolOptionsKey, Value: "-Xmx512m -javaagent:/datadog-lib/dd-java-agent.jar"},
	}
	originalPod := pod.DeepCopy()

	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, originalPod, pod)
}

func TestInjectOTelMergesJavaToolOptionsAndIsIdempotent(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		return true, []*config.OTelRule{newTestOTelRule("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0")}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-otel-java-existing-options", nil)
	existingJavaToolOptions := "  -Xmx512m -javaagent:/profiler/profiler.jar  "
	pod.Spec.Containers[0].Env = []corev1.EnvVar{
		{Name: otelJavaToolOptionsKey, Value: existingJavaToolOptions},
	}

	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t,
		existingJavaToolOptions+" "+otelJavaAgentOption,
		envValues(pod.Spec.Containers[0].Env)[otelJavaToolOptionsKey],
	)

	injectedPod := pod.DeepCopy()
	changed, err = InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, injectedPod, pod)
}

func TestInjectOTelSkipsWhenDisabled(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		return true, []*config.OTelRule{newTestOTelRule("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0")}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-otel-disabled", map[string]string{otelEnabledAnnotationKey: "false"})
	originalPod := pod.DeepCopy()

	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, originalPod, pod)
}

func TestInjectOTelSkipsIncompatiblePod(t *testing.T) {
	tests := []struct {
		name   string
		image  string
		mutate func(*corev1.Pod)
	}{
		{
			name:  "without application containers",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.Containers = nil
			},
		},
		{
			name:  "with empty image",
			image: "",
		},
		{
			name:  "with datadog init container",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.InitContainers = []corev1.Container{{Name: ddtraceInitContainerName, Image: "dd-java-agent"}}
			},
		},
		{
			name:  "with conflicting otel init container",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.InitContainers = []corev1.Container{{Name: otelInitContainerName, Image: "other-image"}}
			},
		},
		{
			name:  "with conflicting otel volume",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.Volumes = []corev1.Volume{{
					Name: otelVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{SecretName: "existing"},
					},
				}}
			},
		},
		{
			name:  "with conflicting otel mount name",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{Name: otelVolumeName, MountPath: "/other"}}
			},
		},
		{
			name:  "with conflicting otel mount path",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{Name: "existing", MountPath: otelJavaMountPath}}
			},
		},
		{
			name:  "with otel mount using subpath",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{
					Name:      otelVolumeName,
					MountPath: otelJavaMountPath,
					SubPath:   "agent",
				}}
			},
		},
		{
			name:  "with duplicate java tool options",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.Containers[0].Env = []corev1.EnvVar{
					{Name: otelJavaToolOptionsKey, Value: "-Xmx512m"},
					{Name: otelJavaToolOptionsKey, Value: "-javaagent:/datadog-lib/dd-java-agent.jar"},
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalFunc := otelMatchAllNamespaceOrLabelsForConfig
			otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
				return true, []*config.OTelRule{newTestOTelRule(tc.image)}
			}
			defer func() {
				otelMatchAllNamespaceOrLabelsForConfig = originalFunc
			}()

			pod := createTestPod("test-otel-java-conflict", nil)
			if tc.mutate != nil {
				tc.mutate(pod)
			}
			originalPod := pod.DeepCopy()

			changed, err := InjectOTelToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, originalPod, pod)
		})
	}
}

func envValues(envs []corev1.EnvVar) map[string]string {
	values := make(map[string]string, len(envs))
	for _, env := range envs {
		values[env.Name] = env.Value
	}
	return values
}
