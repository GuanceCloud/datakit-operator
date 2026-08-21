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

func useTestOTelNodeJSRule(t *testing.T) {
	t.Helper()
	originalConfig := config.Cfg
	rule := newTestOTelRuleForLanguage(
		"nodejs",
		"ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:0.78.0",
	)
	rule.Selector = config.Selector{Namespaces: []string{"default"}}
	config.Cfg = &config.Configuration{
		AdmissionInject: config.AdmissionInjectConfig{OTels: config.OTelRules{rule}},
	}
	t.Cleanup(func() {
		config.Cfg = originalConfig
	})
	if !assert.NoError(t, config.Cfg.Setup()) {
		t.FailNow()
	}
}

func TestInjectTracingInjectsOTelNodeJS(t *testing.T) {
	originalConfig := config.Cfg
	rule := newTestOTelRuleForLanguage(
		"nodejs",
		"ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:0.78.0",
	)
	rule.Selector = config.Selector{Namespaces: []string{"default"}}
	rule.Envs = config.Envs{{Key: "OTEL_SERVICE_NAME", Value: "checkout-nodejs"}}
	config.Cfg = &config.Configuration{
		AdmissionInject: config.AdmissionInjectConfig{OTels: config.OTelRules{rule}},
	}
	defer func() {
		config.Cfg = originalConfig
	}()
	if !assert.NoError(t, config.Cfg.Setup()) {
		return
	}

	runAsNonRoot := true
	runAsUser := int64(1000)
	pod := createTestPod("test-tracing-otel-nodejs", map[string]string{ddtraceEnabledAnnotationKey: "false"})
	pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{RunAsNonRoot: &runAsNonRoot, RunAsUser: &runAsUser}
	pod.Spec.Containers[0].Env = []corev1.EnvVar{{
		Name:  "NODE_OPTIONS",
		Value: "--require /app/preload.js --max-old-space-size=512",
	}}
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: "sidecar", Image: "node:24-alpine"})

	changed, err := InjectTracingToPod("default", pod.Name, pod)
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, changed)

	if assert.Len(t, pod.Spec.InitContainers, 1) {
		initContainer := pod.Spec.InitContainers[0]
		assert.Equal(t, "datakit-otel-lib-init", initContainer.Name)
		assert.Equal(t, "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:0.78.0", initContainer.Image)
		assert.Equal(t, []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-nodejs"}, initContainer.Command)
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

	expectedNodeOptions := map[string]string{
		"nginx":   "--require /app/preload.js --max-old-space-size=512 --require /otel-auto-instrumentation-nodejs/autoinstrumentation.js",
		"sidecar": " --require /otel-auto-instrumentation-nodejs/autoinstrumentation.js",
	}
	for _, container := range pod.Spec.Containers {
		if assert.Len(t, container.VolumeMounts, 1, container.Name) {
			assert.Equal(t, "datakit-otel-auto-instrument", container.VolumeMounts[0].Name)
			assert.Equal(t, "/otel-auto-instrumentation-nodejs", container.VolumeMounts[0].MountPath)
		}
		envs := envValues(container.Env)
		assert.Equal(t, expectedNodeOptions[container.Name], envs["NODE_OPTIONS"], container.Name)
		assert.Equal(t, "checkout-nodejs", envs["OTEL_SERVICE_NAME"], container.Name)
	}

	injectedPod := pod.DeepCopy()
	changed, err = InjectTracingToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, injectedPod, pod)
}

func TestInjectTracingSkipsOTelNodeJSAtomicallyWhenNodeOptionsUsesValueFrom(t *testing.T) {
	originalConfig := config.Cfg
	rule := newTestOTelRuleForLanguage(
		"nodejs",
		"ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:0.78.0",
	)
	rule.Selector = config.Selector{Namespaces: []string{"default"}}
	config.Cfg = &config.Configuration{
		AdmissionInject: config.AdmissionInjectConfig{OTels: config.OTelRules{rule}},
	}
	defer func() {
		config.Cfg = originalConfig
	}()
	if !assert.NoError(t, config.Cfg.Setup()) {
		return
	}

	pod := createTestPod("test-tracing-otel-nodejs-value-from", map[string]string{ddtraceEnabledAnnotationKey: "false"})
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
		Name:  "sidecar",
		Image: "node:24-alpine",
		Env: []corev1.EnvVar{{
			Name: otelNodeJSOptionsKey,
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{Key: "node-options"},
			},
		}},
	})
	originalPod := pod.DeepCopy()

	changed, err := InjectTracingToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, originalPod, pod)
}

func TestInjectTracingSkipsOTelNodeJSAtomicallyWithDuplicateNodeOptions(t *testing.T) {
	originalConfig := config.Cfg
	rule := newTestOTelRuleForLanguage(
		"nodejs",
		"ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:0.78.0",
	)
	rule.Selector = config.Selector{Namespaces: []string{"default"}}
	config.Cfg = &config.Configuration{
		AdmissionInject: config.AdmissionInjectConfig{OTels: config.OTelRules{rule}},
	}
	defer func() {
		config.Cfg = originalConfig
	}()
	if !assert.NoError(t, config.Cfg.Setup()) {
		return
	}

	pod := createTestPod("test-tracing-otel-nodejs-duplicate-options", map[string]string{ddtraceEnabledAnnotationKey: "false"})
	pod.Spec.Containers[0].Env = []corev1.EnvVar{
		{Name: otelNodeJSOptionsKey, Value: "--max-old-space-size=512"},
		{Name: otelNodeJSOptionsKey, Value: "--trace-warnings"},
	}
	originalPod := pod.DeepCopy()

	changed, err := InjectTracingToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, originalPod, pod)
}

func TestInjectTracingSkipsOTelNodeJSWithDatadogRequire(t *testing.T) {
	tests := []struct {
		name        string
		nodeOptions string
	}{
		{
			name:        "operator path",
			nodeOptions: "--trace-warnings --require=/datadog-lib/node_modules/dd-trace/init",
		},
		{
			name:        "package name",
			nodeOptions: "--require dd-trace/init",
		},
		{
			name:        "application node_modules path",
			nodeOptions: "--require=/app/node_modules/dd-trace/init",
		},
		{
			name:        "short require flag",
			nodeOptions: "-r dd-trace/init",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			useTestOTelNodeJSRule(t)
			pod := createTestPod("test-tracing-otel-nodejs-datadog", map[string]string{
				ddtraceEnabledAnnotationKey: "false",
			})
			pod.Spec.Containers[0].Env = []corev1.EnvVar{{Name: otelNodeJSOptionsKey, Value: tc.nodeOptions}}
			originalPod := pod.DeepCopy()

			changed, err := InjectTracingToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, originalPod, pod)
		})
	}
}

func TestInjectTracingSkipsOTelNodeJSWithIncompatibleOTelRequire(t *testing.T) {
	tests := []struct {
		name        string
		nodeOptions string
	}{
		{
			name:        "equals form",
			nodeOptions: "--require=/otel-auto-instrumentation-nodejs/autoinstrumentation.js",
		},
		{
			name:        "different entrypoint",
			nodeOptions: "--require /otel-auto-instrumentation-nodejs/register.js",
		},
		{
			name:        "partial path",
			nodeOptions: "--require otel-auto-instrumentation-nodejs/autoinstrumentation.js",
		},
		{
			name: "duplicate official option",
			nodeOptions: "--require /otel-auto-instrumentation-nodejs/autoinstrumentation.js " +
				"--require /otel-auto-instrumentation-nodejs/autoinstrumentation.js",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			useTestOTelNodeJSRule(t)
			pod := createTestPod("test-tracing-otel-nodejs-incompatible-require", map[string]string{
				ddtraceEnabledAnnotationKey: "false",
			})
			pod.Spec.Containers[0].Env = []corev1.EnvVar{{Name: otelNodeJSOptionsKey, Value: tc.nodeOptions}}
			originalPod := pod.DeepCopy()

			changed, err := InjectTracingToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, originalPod, pod)
		})
	}
}

func TestInjectTracingPrefersDDTrace(t *testing.T) {
	originalDDTraceFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
	originalOTelFunc := otelMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
		return true, []*config.DDTraceRule{
			newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/dd-lib-java-init:v1.0.0"),
		}
	}
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		return true, []*config.OTelRule{
			newTestOTelRule("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0"),
		}
	}
	defer func() {
		ddtraceMatchAllNamespaceOrLabelsForConfig = originalDDTraceFunc
		otelMatchAllNamespaceOrLabelsForConfig = originalOTelFunc
	}()

	pod := createTestPod("test-tracing-ddtrace-wins", nil)
	changed, err := InjectTracingToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	if assert.Len(t, pod.Spec.InitContainers, 1) {
		assert.Equal(t, ddtraceInitContainerName, pod.Spec.InitContainers[0].Name)
	}
	javaToolOptions := envValues(pod.Spec.Containers[0].Env)[otelJavaToolOptionsKey]
	assert.Contains(t, javaToolOptions, ddJavaAgentFileName)
	assert.NotContains(t, javaToolOptions, otelJavaAgentOption)
}

func TestInjectTracingDoesNotFallbackWhenSelectedDDTraceCannotInject(t *testing.T) {
	originalDDTraceFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
	originalOTelFunc := otelMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
		return true, []*config.DDTraceRule{
			newTestDDTraceRule("unsupported", "example.com/ddtrace:1.0.0"),
		}
	}
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		return true, []*config.OTelRule{
			newTestOTelRule("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0"),
		}
	}
	defer func() {
		ddtraceMatchAllNamespaceOrLabelsForConfig = originalDDTraceFunc
		otelMatchAllNamespaceOrLabelsForConfig = originalOTelFunc
	}()

	pod := createTestPod("test-tracing-no-fallback", nil)
	originalPod := pod.DeepCopy()
	changed, err := InjectTracingToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, originalPod, pod)
}

func TestInjectTracingUsesOTelWhenDDTraceIsDisabled(t *testing.T) {
	originalDDTraceFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
	originalOTelFunc := otelMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
		return true, []*config.DDTraceRule{
			newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/dd-lib-java-init:v1.0.0"),
		}
	}
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		return true, []*config.OTelRule{
			newTestOTelRule("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0"),
		}
	}
	defer func() {
		ddtraceMatchAllNamespaceOrLabelsForConfig = originalDDTraceFunc
		otelMatchAllNamespaceOrLabelsForConfig = originalOTelFunc
	}()

	pod := createTestPod("test-tracing-otel", map[string]string{ddtraceEnabledAnnotationKey: "false"})
	changed, err := InjectTracingToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	if assert.Len(t, pod.Spec.InitContainers, 1) {
		assert.Equal(t, otelInitContainerName, pod.Spec.InitContainers[0].Name)
	}
}

func TestInjectTracingKeepsExistingOTelInjection(t *testing.T) {
	originalDDTraceFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
	originalOTelFunc := otelMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
		return true, []*config.DDTraceRule{
			newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/dd-lib-java-init:v1.0.0"),
		}
	}
	otelMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.OTelRule) {
		return true, []*config.OTelRule{
			newTestOTelRule("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0"),
		}
	}
	defer func() {
		ddtraceMatchAllNamespaceOrLabelsForConfig = originalDDTraceFunc
		otelMatchAllNamespaceOrLabelsForConfig = originalOTelFunc
	}()

	pod := createTestPod("test-tracing-existing-otel", map[string]string{ddtraceEnabledAnnotationKey: "false"})
	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	injectedPod := pod.DeepCopy()
	pod.Annotations[ddtraceEnabledAnnotationKey] = "true"
	injectedPod.Annotations[ddtraceEnabledAnnotationKey] = "true"

	changed, err = InjectTracingToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, injectedPod, pod)
}
