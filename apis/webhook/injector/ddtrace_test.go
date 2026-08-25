// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	corev1 "k8s.io/api/core/v1"
)

func newTestDDTraceRule(language, image string) *config.DDTraceRule {
	return &config.DDTraceRule{
		InjectRule: config.InjectRule{
			Image: image,
			Resources: config.ResourceRequirements{
				Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
				Limits:   config.ResourceQuotaConfig{CPU: "500m", Memory: "512Mi"},
			},
		},
		Language: language,
	}
}

func useDDTraceRule(t *testing.T, rule *config.DDTraceRule) {
	t.Helper()
	original := ddtraceMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(string, map[string]string) (bool, []*config.DDTraceRule) {
		return true, []*config.DDTraceRule{rule}
	}
	t.Cleanup(func() { ddtraceMatchAllNamespaceOrLabelsForConfig = original })
}

func TestInjectDDTraceJava(t *testing.T) {
	rule := newTestDDTraceRule("java", "example.com/dd-java:1.0.0")
	rule.Envs = config.Envs{{Key: "DD_SERVICE", Value: "checkout"}}
	useDDTraceRule(t, rule)
	pod := createTestPod("java", nil)
	pod.Spec.Containers[0].Env = []corev1.EnvVar{{Name: javaToolOptionsKey, Value: "-Xms128m"}}

	changed, err := InjectDDTraceToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.Len(t, pod.Spec.InitContainers, 1)
	assert.Len(t, pod.Spec.Volumes, 1)
	assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 1)
	options, ok := findEnv(pod.Spec.Containers[0].Env, javaToolOptionsKey)
	if assert.True(t, ok) {
		assert.Contains(t, options.Value, "-Xms128m")
		assert.Equal(t, 1, strings.Count(options.Value, javaAgentOption))
	}
	service, ok := findEnv(pod.Spec.Containers[0].Env, "DD_SERVICE")
	if assert.True(t, ok) {
		assert.Equal(t, "checkout", service.Value)
	}

	afterFirstInjection := pod.DeepCopy()
	changed, err = InjectDDTraceToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, afterFirstInjection, pod)
}

func TestInjectDDTraceRepairsJavaReinvocation(t *testing.T) {
	useDDTraceRule(t, newTestDDTraceRule("java", "example.com/dd-java:1.0.0"))
	pod := createTestPod("reinvoked", nil)
	changed, err := InjectDDTraceToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)

	pod.Spec.Containers[0].Env = nil
	changed, err = InjectDDTraceToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, 1, countNamedEnv(pod.Spec.Containers[0].Env, javaToolOptionsKey))
	assert.Len(t, pod.Spec.InitContainers, 1)
	assert.Len(t, pod.Spec.Volumes, 1)
	assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 1)

	afterRepair := pod.DeepCopy()
	changed, err = InjectDDTraceToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, afterRepair, pod)
}

func TestInjectDDTraceRepairRequiresCompleteMountChain(t *testing.T) {
	useDDTraceRule(t, newTestDDTraceRule("java", "example.com/dd-java:1.0.0"))
	complete := createTestPod("partial", nil)
	changed, err := InjectDDTraceToPod("default", complete.Name, complete)
	if !assert.NoError(t, err) || !assert.True(t, changed) || !assert.Len(t, complete.Spec.InitContainers, 1) {
		return
	}

	tests := []struct {
		name   string
		remove func(*corev1.Pod)
	}{
		{"volume", func(p *corev1.Pod) { p.Spec.Volumes = nil }},
		{"application mount", func(p *corev1.Pod) { p.Spec.Containers[0].VolumeMounts = nil }},
		{"init mount", func(p *corev1.Pod) { p.Spec.InitContainers[0].VolumeMounts = nil }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pod := complete.DeepCopy()
			pod.Spec.Containers[0].Env = nil
			tc.remove(pod)
			before := pod.DeepCopy()
			changed, err := InjectDDTraceToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, before, pod)
		})
	}
}

func TestInjectDDTracePreservesEnvironmentOrder(t *testing.T) {
	rule := newTestDDTraceRule("java", "example.com/dd-java:1.0.0")
	rule.Envs = config.Envs{
		{Key: "NEW_FIRST", Value: "one"},
		{Key: ddtraceDDTagsKey, Value: "cluster:prod"},
		{Key: "EXISTING", Value: "replace-me"},
		{Key: "NEW_LAST", Value: "$(NEW_FIRST)"},
	}
	useDDTraceRule(t, rule)
	pod := createTestPod("ordered", nil)
	pod.Spec.Containers[0].Env = []corev1.EnvVar{
		{Name: "BEFORE", Value: "before"},
		{Name: ddtraceDDTagsKey, Value: "user:value"},
		{Name: "EXISTING", Value: "keep-me"},
		{Name: "AFTER", Value: "after"},
	}

	changed, err := InjectDDTraceToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, []string{
		"BEFORE", ddtraceDDTagsKey, "EXISTING", "AFTER",
		javaToolOptionsKey, "NEW_FIRST", "NEW_LAST",
	}, envNames(pod.Spec.Containers[0].Env))
	tags, _ := findEnv(pod.Spec.Containers[0].Env, ddtraceDDTagsKey)
	assert.Equal(t, "user:value,cluster:prod", tags.Value)
	existing, _ := findEnv(pod.Spec.Containers[0].Env, "EXISTING")
	assert.Equal(t, "keep-me", existing.Value)
}

func TestInjectDDTracePreservesValueFromDDTags(t *testing.T) {
	rule := newTestDDTraceRule("java", "example.com/dd-java:1.0.0")
	rule.Envs = config.Envs{{Key: ddtraceDDTagsKey, Value: "cluster:prod"}}
	useDDTraceRule(t, rule)
	pod := createTestPod("tags-value-from", nil)
	pod.Spec.Containers[0].Env = []corev1.EnvVar{
		{Name: "BEFORE", Value: "before"}, fieldRefEnv(ddtraceDDTagsKey), {Name: "AFTER", Value: "after"},
	}

	changed, err := InjectDDTraceToPod("default", pod.Name, pod)
	if !assert.NoError(t, err) || !assert.True(t, changed) {
		return
	}
	assert.Equal(t, []string{"BEFORE", ddtraceDDTagsKey, "AFTER", javaToolOptionsKey}, envNames(pod.Spec.Containers[0].Env))
	tags, ok := findEnv(pod.Spec.Containers[0].Env, ddtraceDDTagsKey)
	if !assert.True(t, ok) {
		return
	}
	assert.Equal(t, fieldRefEnv(ddtraceDDTagsKey), tags)
}

func TestInjectDDTraceReinvocationLeavesValueFromJavaOptionsUntouched(t *testing.T) {
	useDDTraceRule(t, newTestDDTraceRule("java", "example.com/dd-java:1.0.0"))
	pod := createTestPod("value-from", nil)
	changed, err := InjectDDTraceToPod("default", pod.Name, pod)
	if !assert.NoError(t, err) || !assert.True(t, changed) {
		return
	}
	pod.Spec.Containers[0].Env = []corev1.EnvVar{fieldRefEnv(javaToolOptionsKey)}
	before := pod.DeepCopy()

	changed, err = InjectDDTraceToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, before, pod)
}
