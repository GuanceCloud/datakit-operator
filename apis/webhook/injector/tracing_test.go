// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
)

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
