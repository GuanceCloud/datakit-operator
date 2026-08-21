// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
)

func newTestOTelRule(image string) *config.OTelRule {
	return newTestOTelRuleForLanguage("java", image)
}

func newTestOTelRuleForLanguage(lang, image string) *config.OTelRule {
	return &config.OTelRule{
		InjectRule: config.InjectRule{
			Image: image,
			Resources: config.ResourceRequirements{
				Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
				Limits:   config.ResourceQuotaConfig{CPU: "500m", Memory: "512Mi"},
			},
		},
		Language: lang,
	}
}

func TestInjectOTelSkipsRunAsNonRootWithoutRunAsUser(t *testing.T) {
	tests := []struct {
		name     string
		language string
		image    string
	}{
		{
			name:     "java",
			language: "java",
			image:    "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0",
		},
		{
			name:     "python",
			language: "python",
			image:    "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
		},
		{
			name:     "nodejs",
			language: "nodejs",
			image:    "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:0.78.0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalFunc := otelMatchAllNamespaceOrLabelsForConfig
			otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
				return true, []*config.OTelRule{newTestOTelRuleForLanguage(tc.language, tc.image)}
			}
			defer func() {
				otelMatchAllNamespaceOrLabelsForConfig = originalFunc
			}()

			runAsNonRoot := true
			pod := createTestPod("test-otel-nonroot-"+tc.name, nil)
			pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{RunAsNonRoot: &runAsNonRoot}
			originalPod := pod.DeepCopy()

			changed, err := InjectOTelToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, originalPod, pod)
		})
	}
}

func TestInjectOTelRunAsNonRootWarningIncludesTraceFields(t *testing.T) {
	const image = "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0"
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		rule := newTestOTelRule(image)
		rule.Name = "otel-java"
		return true, []*config.OTelRule{rule}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	var logs bytes.Buffer
	originalLog := log
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), zapcore.AddSync(&logs), zap.WarnLevel)
	log = &logger.Logger{SugaredLogger: zap.New(core).Sugar()}
	defer func() {
		log = originalLog
	}()

	runAsNonRoot := true
	pod := createTestPod("test-otel-run-as-non-root", nil)
	pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{RunAsNonRoot: &runAsNonRoot}

	changed, err := InjectOTelToPod("default", "<no-one>", pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Contains(t, logs.String(),
		"otel inject skipped: pod=test-otel-run-as-non-root, namespace=default, rule=otel-java, language=java, image="+image+
			", reason=run_as_non_root_without_run_as_user",
	)
}

func TestInjectOTelRunAsNonRootGuardUsesEffectiveSecurityContext(t *testing.T) {
	runAsNonRoot := true
	doNotRunAsNonRoot := false
	nonRootUser := int64(1000)
	rootUser := int64(0)
	tests := []struct {
		name                     string
		podSecurityContext       *corev1.PodSecurityContext
		containerSecurityContext *corev1.SecurityContext
		wantChanged              bool
	}{
		{
			name:               "pod requires non-root without user",
			podSecurityContext: &corev1.PodSecurityContext{RunAsNonRoot: &runAsNonRoot},
		},
		{
			name: "pod provides non-root user",
			podSecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: &runAsNonRoot,
				RunAsUser:    &nonRootUser,
			},
			wantChanged: true,
		},
		{
			name:               "container disables pod non-root requirement",
			podSecurityContext: &corev1.PodSecurityContext{RunAsNonRoot: &runAsNonRoot},
			containerSecurityContext: &corev1.SecurityContext{
				RunAsNonRoot: &doNotRunAsNonRoot,
			},
			wantChanged: true,
		},
		{
			name:                     "container inherits pod non-root user",
			podSecurityContext:       &corev1.PodSecurityContext{RunAsUser: &nonRootUser},
			containerSecurityContext: &corev1.SecurityContext{RunAsNonRoot: &runAsNonRoot},
			wantChanged:              true,
		},
		{
			name: "container root user overrides pod non-root user",
			podSecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: &runAsNonRoot,
				RunAsUser:    &nonRootUser,
			},
			containerSecurityContext: &corev1.SecurityContext{RunAsUser: &rootUser},
		},
		{
			name:                     "container provides non-root user for pod requirement",
			podSecurityContext:       &corev1.PodSecurityContext{RunAsNonRoot: &runAsNonRoot},
			containerSecurityContext: &corev1.SecurityContext{RunAsUser: &nonRootUser},
			wantChanged:              true,
		},
		{
			name:                     "root user without non-root requirement",
			containerSecurityContext: &corev1.SecurityContext{RunAsUser: &rootUser},
			wantChanged:              true,
		},
	}

	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		return true, []*config.OTelRule{newTestOTelRule("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0")}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pod := createTestPod("test-otel-security-context", nil)
			if tc.podSecurityContext != nil {
				pod.Spec.SecurityContext = tc.podSecurityContext.DeepCopy()
			}
			if tc.containerSecurityContext != nil {
				pod.Spec.Containers[0].SecurityContext = tc.containerSecurityContext.DeepCopy()
			}
			originalPod := pod.DeepCopy()

			changed, err := InjectOTelToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantChanged, changed)
			if tc.wantChanged {
				assert.Len(t, pod.Spec.InitContainers, 1)
			} else {
				assert.Equal(t, originalPod, pod)
			}
		})
	}
}

func TestInjectOTelPythonToPod(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		rule := newTestOTelRuleForLanguage("python", "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0")
		rule.Envs = []struct{ Key, Value string }{
			{Key: "OTEL_SERVICE_NAME", Value: "checkout-python"},
		}
		return true, []*config.OTelRule{rule}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	runAsNonRoot := true
	runAsUser := int64(1000)
	pod := createTestPod("test-otel-python", map[string]string{otelEnabledAnnotationKey: "true"})
	pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{RunAsNonRoot: &runAsNonRoot, RunAsUser: &runAsUser}
	pod.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "PYTHONPATH", Value: "/app:/vendor"}}
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: "sidecar", Image: "python:3.14-slim"})

	changed, err := InjectOTelToPod("default", pod.Name, pod)
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, changed)

	if assert.Len(t, pod.Spec.InitContainers, 1) {
		initContainer := pod.Spec.InitContainers[0]
		assert.Equal(t, "datakit-otel-lib-init", initContainer.Name)
		assert.Equal(t, "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0", initContainer.Image)
		assert.Equal(t, []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-python"}, initContainer.Command)
		assert.Equal(t, corev1.PullAlways, initContainer.ImagePullPolicy)
		assert.Equal(t, pod.Spec.Containers[0].SecurityContext, initContainer.SecurityContext)
		assert.Equal(t, "100m", initContainer.Resources.Requests.Cpu().String())
		assert.Equal(t, "64Mi", initContainer.Resources.Requests.Memory().String())
		assert.Equal(t, "500m", initContainer.Resources.Limits.Cpu().String())
		assert.Equal(t, "512Mi", initContainer.Resources.Limits.Memory().String())
	}

	if assert.Len(t, pod.Spec.Volumes, 1) {
		assert.Equal(t, "datakit-otel-auto-instrument", pod.Spec.Volumes[0].Name)
		assert.NotNil(t, pod.Spec.Volumes[0].EmptyDir)
	}

	expectedPythonPaths := map[string]string{
		"nginx":   "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation:/app:/vendor:/otel-auto-instrumentation-python",
		"sidecar": "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation:/otel-auto-instrumentation-python",
	}
	for _, container := range pod.Spec.Containers {
		if assert.Len(t, container.VolumeMounts, 1, container.Name) {
			assert.Equal(t, "datakit-otel-auto-instrument", container.VolumeMounts[0].Name)
			assert.Equal(t, "/otel-auto-instrumentation-python", container.VolumeMounts[0].MountPath)
		}
		envs := envValues(container.Env)
		assert.Equal(t, expectedPythonPaths[container.Name], envs["PYTHONPATH"], container.Name)
		assert.Equal(t, "checkout-python", envs["OTEL_SERVICE_NAME"], container.Name)
	}
}

func TestInjectOTelPythonSkipsWholePodWhenPythonPathUsesValueFrom(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		return true, []*config.OTelRule{newTestOTelRuleForLanguage(
			"python",
			"ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
		)}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-otel-python-value-from", nil)
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
		Name:  "python-sidecar",
		Image: "python:3.14-slim",
		Env: []corev1.EnvVar{{
			Name: otelPythonPathKey,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{Key: "python-path"},
			},
		}},
	})
	originalPod := pod.DeepCopy()

	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, originalPod, pod)
}

func TestInjectOTelPythonSkipsWholePodWithDatadogPythonPath(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		return true, []*config.OTelRule{newTestOTelRuleForLanguage(
			"python",
			"ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
		)}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-otel-python-datadog", nil)
	pod.Spec.Containers[0].Env = []corev1.EnvVar{{
		Name:  otelPythonPathKey,
		Value: "/app:/datadog-lib/:/vendor",
	}}
	originalPod := pod.DeepCopy()

	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, originalPod, pod)
}

func TestInjectOTelPythonIsIdempotent(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		return true, []*config.OTelRule{newTestOTelRuleForLanguage(
			"python",
			"ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
		)}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-otel-python-idempotent", nil)
	pod.Spec.Containers[0].Env = []corev1.EnvVar{{Name: otelPythonPathKey, Value: "/app"}}

	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)

	injectedPod := pod.DeepCopy()
	changed, err = InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, injectedPod, pod)
}

func TestInjectOTelPythonSkipsWholePodWithPartialOrMisplacedInjectionPath(t *testing.T) {
	tests := []struct {
		name       string
		pythonPath string
	}{
		{
			name:       "prefix only",
			pythonPath: "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation:/app",
		},
		{
			name:       "suffix only",
			pythonPath: "/app:/otel-auto-instrumentation-python",
		},
		{
			name:       "prefix in the middle",
			pythonPath: "/app:/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation:/otel-auto-instrumentation-python",
		},
		{
			name:       "duplicate suffix",
			pythonPath: "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation:/app:/otel-auto-instrumentation-python:/otel-auto-instrumentation-python",
		},
		{
			name:       "otel paths with trailing slash",
			pythonPath: "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation/:/app:/otel-auto-instrumentation-python/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalFunc := otelMatchAllNamespaceOrLabelsForConfig
			otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
				return true, []*config.OTelRule{newTestOTelRuleForLanguage(
					"python",
					"ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
				)}
			}
			defer func() {
				otelMatchAllNamespaceOrLabelsForConfig = originalFunc
			}()

			pod := createTestPod("test-otel-python-partial", nil)
			pod.Spec.Containers[0].Env = []corev1.EnvVar{{Name: otelPythonPathKey, Value: tc.pythonPath}}
			originalPod := pod.DeepCopy()

			changed, err := InjectOTelToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, originalPod, pod)
		})
	}
}

func TestInjectOTelPythonSkipsIncompatiblePod(t *testing.T) {
	tests := []struct {
		name   string
		image  string
		mutate func(*corev1.Pod)
	}{
		{
			name:  "without application containers",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
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
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.InitContainers = []corev1.Container{{Name: ddtraceInitContainerName, Image: "dd-python-lib"}}
			},
		},
		{
			name:  "with datadog library mount path",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{Name: "existing", MountPath: ddtraceMountPath}}
			},
		},
		{
			name:  "with conflicting otel init container",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.InitContainers = []corev1.Container{{
					Name:    otelInitContainerName,
					Image:   "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
					Command: []string{"other"},
				}}
			},
		},
		{
			name:  "with conflicting owned init mount path",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
			mutate: func(pod *corev1.Pod) {
				rule := newTestOTelRuleForLanguage(
					"python",
					"ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
				)
				resource := &otelResource{pod: pod}
				library, _ := resource.libraryForLanguage(python)
				initContainer := resource.initContainer(rule, rule.Image, library)
				initContainer.VolumeMounts = append(initContainer.VolumeMounts, corev1.VolumeMount{
					Name:      "conflicting",
					MountPath: otelPythonMountPath,
				})
				pod.Spec.InitContainers = []corev1.Container{initContainer}
			},
		},
		{
			name:  "with conflicting otel volume",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
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
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{Name: otelVolumeName, MountPath: "/other"}}
			},
		},
		{
			name:  "with conflicting otel mount path",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{Name: "existing", MountPath: otelPythonMountPath}}
			},
		},
		{
			name:  "with otel mount using subpath",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{
					Name:      otelVolumeName,
					MountPath: otelPythonMountPath,
					SubPath:   "agent",
				}}
			},
		},
		{
			name:  "with duplicate python path",
			image: "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
			mutate: func(pod *corev1.Pod) {
				pod.Spec.Containers[0].Env = []corev1.EnvVar{
					{Name: otelPythonPathKey, Value: "/app"},
					{Name: otelPythonPathKey, Value: "/vendor"},
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalFunc := otelMatchAllNamespaceOrLabelsForConfig
			otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
				return true, []*config.OTelRule{newTestOTelRuleForLanguage("python", tc.image)}
			}
			defer func() {
				otelMatchAllNamespaceOrLabelsForConfig = originalFunc
			}()

			pod := createTestPod("test-otel-python-conflict", nil)
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

func TestInjectOTelReinvocationToleratesInitContainerDefaults(t *testing.T) {
	tests := []struct {
		name         string
		language     string
		image        string
		expectedEnv  string
		expectedPath string
	}{
		{
			name:         "java",
			language:     "java",
			image:        "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0",
			expectedEnv:  otelJavaToolOptionsKey,
			expectedPath: otelJavaAgentOption,
		},
		{
			name:         "python",
			language:     "python",
			image:        "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
			expectedEnv:  otelPythonPathKey,
			expectedPath: otelPythonPathPrefix + ":" + otelPythonMountPath,
		},
		{
			name:         "nodejs",
			language:     "nodejs",
			image:        "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:0.78.0",
			expectedEnv:  otelNodeJSOptionsKey,
			expectedPath: " " + otelNodeJSRequireOption,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalFunc := otelMatchAllNamespaceOrLabelsForConfig
			otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
				return true, []*config.OTelRule{newTestOTelRuleForLanguage(tc.language, tc.image)}
			}
			defer func() {
				otelMatchAllNamespaceOrLabelsForConfig = originalFunc
			}()

			pod := createTestPod("test-otel-reinvocation-"+tc.name, nil)
			runAsNonRoot := true
			runAsUser := int64(1000)
			pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{RunAsNonRoot: &runAsNonRoot, RunAsUser: &runAsUser}
			changed, err := InjectOTelToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.True(t, changed)
			if !assert.Len(t, pod.Spec.InitContainers, 1) {
				return
			}

			allowPrivilegeEscalation := false
			initContainer := &pod.Spec.InitContainers[0]
			initContainer.TerminationMessagePath = "/dev/termination-log"
			initContainer.TerminationMessagePolicy = corev1.TerminationMessageReadFile
			initContainer.SecurityContext.AllowPrivilegeEscalation = &allowPrivilegeEscalation
			initContainer.Env = append(initContainer.Env, corev1.EnvVar{Name: "WEBHOOK_ADDED", Value: "true"})
			initContainer.VolumeMounts = append(initContainer.VolumeMounts, corev1.VolumeMount{
				Name:      "kube-api-access-test",
				MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
				ReadOnly:  true,
			})
			pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: "late-sidecar", Image: "example.com/sidecar:1.0.0"})

			changed, err = InjectOTelToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.True(t, changed)
			assert.Len(t, pod.Spec.InitContainers, 1)
			assert.Len(t, pod.Spec.Volumes, 1)
			lateContainer := pod.Spec.Containers[len(pod.Spec.Containers)-1]
			assert.Equal(t, tc.expectedPath, envValues(lateContainer.Env)[tc.expectedEnv])
			assert.Len(t, lateContainer.VolumeMounts, 1)
		})
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
	runAsUser := int64(1000)
	pod := createTestPod("test-otel-java", map[string]string{otelEnabledAnnotationKey: "true"})
	pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{RunAsNonRoot: &runAsNonRoot, RunAsUser: &runAsUser}
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

func TestInjectOTelPythonUsesVersionAnnotation(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		rule := newTestOTelRuleForLanguage(
			"python",
			"ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
		)
		rule.CheckAnnotation = true
		return true, []*config.OTelRule{rule}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-otel-python-version", map[string]string{
		otelEnabledAnnotationKey:                    "true",
		"admission.datakit/otel-python-lib.version": "0.63b1",
	})
	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	if assert.Len(t, pod.Spec.InitContainers, 1) {
		assert.Equal(t,
			"ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.63b1",
			pod.Spec.InitContainers[0].Image,
		)
	}
}

func TestInjectOTelNodeJSUsesVersionAnnotation(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		rule := newTestOTelRuleForLanguage(
			"nodejs",
			"ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:0.78.0",
		)
		rule.CheckAnnotation = true
		return true, []*config.OTelRule{rule}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-otel-nodejs-version", map[string]string{
		otelEnabledAnnotationKey:                    "true",
		"admission.datakit/otel-nodejs-lib.version": "0.77.0",
	})
	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	if assert.Len(t, pod.Spec.InitContainers, 1) {
		assert.Equal(t,
			"ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:0.77.0",
			pod.Spec.InitContainers[0].Image,
		)
	}
}

func TestInjectOTelDoesNotFallbackPastFirstSelectedRule(t *testing.T) {
	originalFunc := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		return true, []*config.OTelRule{
			newTestOTelRuleForLanguage("unsupported", "example.com/unsupported:1.0.0"),
			newTestOTelRuleForLanguage("python", "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0"),
		}
	}
	defer func() {
		otelMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-otel-first-rule-wins", nil)
	originalPod := pod.DeepCopy()
	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, originalPod, pod)
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
