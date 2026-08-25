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

func useUnmatchedDDTrace(t *testing.T) {
	t.Helper()
	original := ddtraceMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(string, map[string]string) (bool, []*config.DDTraceRule) {
		return false, nil
	}
	t.Cleanup(func() { ddtraceMatchAllNamespaceOrLabelsForConfig = original })
}

func TestInjectTracingPrefersDDTrace(t *testing.T) {
	useDDTraceRule(t, newTestDDTraceRule("java", "example.com/dd-java:1"))
	useOTelRules(t, newTestOTelRule("example.com/otel-java:1"))
	pod := createTestPod("ddtrace-wins", nil)

	changed, err := InjectTracingToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	if assert.Len(t, pod.Spec.InitContainers, 1) {
		assert.Equal(t, ddtraceInitContainerName, pod.Spec.InitContainers[0].Name)
	}
	options, ok := findEnv(pod.Spec.Containers[0].Env, javaToolOptionsKey)
	if assert.True(t, ok) {
		assert.Contains(t, options.Value, javaAgentOption)
		assert.NotContains(t, options.Value, otelJavaAgentOption)
	}
}

func TestInjectTracingDoesNotFallbackAfterDDTraceSelection(t *testing.T) {
	useDDTraceRule(t, newTestDDTraceRule("unsupported", "example.com/dd:1"))
	useOTelRules(t, newTestOTelRule("example.com/otel-java:1"))
	pod := createTestPod("no-fallback", nil)
	before := pod.DeepCopy()

	changed, err := InjectTracingToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, before, pod)
}

func TestInjectTracingUsesOTelWhenDDTraceDoesNotMatch(t *testing.T) {
	useUnmatchedDDTrace(t)
	useOTelRules(t, newTestOTelRuleForLanguage("nodejs", "example.com/otel-node:1"))
	pod := createTestPod("otel", nil)

	changed, err := InjectTracingToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	if assert.Len(t, pod.Spec.InitContainers, 1) {
		assert.Equal(t, otelInitContainerName, pod.Spec.InitContainers[0].Name)
	}
	options, ok := findEnv(pod.Spec.Containers[0].Env, otelNodeJSOptionsKey)
	if assert.True(t, ok) {
		assert.Contains(t, options.Value, otelNodeJSRequireOption)
	}
}

func TestInjectTracingKeepsExistingOTelInjection(t *testing.T) {
	useUnmatchedDDTrace(t)
	useOTelRules(t, newTestOTelRule("example.com/otel-java:1"))
	pod := createTestPod("existing-otel", nil)
	changed, err := InjectTracingToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	afterInjection := pod.DeepCopy()
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(string, map[string]string) (bool, []*config.DDTraceRule) {
		return true, []*config.DDTraceRule{newTestDDTraceRule("java", "example.com/dd-java:1")}
	}

	changed, err = InjectTracingToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, afterInjection, pod)
}

func TestInjectTracingSkipsDatadogNodePreload(t *testing.T) {
	for _, options := range []string{
		"--require dd-trace/init",
		"-r dd-trace/init",
		"--require=/app/node_modules/dd-trace/init",
	} {
		t.Run(options, func(t *testing.T) {
			useUnmatchedDDTrace(t)
			useOTelRules(t, newTestOTelRuleForLanguage("nodejs", "example.com/otel-node:1"))
			pod := createTestPod("datadog-node", nil)
			pod.Spec.Containers[0].Env = []corev1.EnvVar{{Name: otelNodeJSOptionsKey, Value: options}}
			before := pod.DeepCopy()

			changed, err := InjectTracingToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, before, pod)
		})
	}
}
