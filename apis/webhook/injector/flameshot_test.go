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

func testFlameshotRule() *config.FlameshotRule {
	return &config.FlameshotRule{
		InjectRule: config.InjectRule{
			Image: "example.com/flameshot:1",
			Envs: config.Envs{
				{Key: flameshotProfilingPathKey, Value: "/flameshot-data"},
				{Key: flameshotProcessesKey, Value: "stale"},
				{Key: flameshotHTTPLocalPortKey, Value: "8089"},
			},
			Resources: config.ResourceRequirements{
				Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
				Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
			},
		},
		Processes:                   `[{"service":"app"}]`,
		EnablePrometheusAnnotations: true,
	}
}

func useFlameshotRule(t *testing.T, rule *config.FlameshotRule) {
	t.Helper()
	original := flameshotMatchNamespaceOrLabelsForConfig
	flameshotMatchNamespaceOrLabelsForConfig = func(string, map[string]string) (bool, *config.FlameshotRule) {
		return true, rule
	}
	t.Cleanup(func() { flameshotMatchNamespaceOrLabelsForConfig = original })
}

func TestInjectFlameshot(t *testing.T) {
	useFlameshotRule(t, testFlameshotRule())
	pod := createTestPod("flameshot", nil)

	changed, err := InjectFlameshotToPod("default", pod.Name, pod)
	if !assert.NoError(t, err) || !assert.True(t, changed) || !assert.NotNil(t, pod.Spec.ShareProcessNamespace) {
		return
	}
	assert.True(t, *pod.Spec.ShareProcessNamespace)
	assert.Equal(t, corev1.RestartPolicyAlways, pod.Spec.RestartPolicy)
	assert.Len(t, pod.Spec.Volumes, 1)
	if !assert.Len(t, pod.Spec.Containers, 2) {
		return
	}
	container := pod.Spec.Containers[1]
	assert.Equal(t, flameshotContainerName, container.Name)
	assert.Equal(t, "example.com/flameshot:1", container.Image)
	if !assert.Len(t, container.Ports, 1) {
		return
	}
	assert.Equal(t, int32(8089), container.Ports[0].ContainerPort)
	if !assert.NotNil(t, container.SecurityContext) || !assert.NotNil(t, container.SecurityContext.Capabilities) {
		return
	}
	assert.Contains(t, container.SecurityContext.Capabilities.Add, corev1.Capability("SYS_PTRACE"))
	assert.Equal(t, []string{flameshotProfilingPathKey, flameshotProcessesKey, flameshotHTTPLocalPortKey}, envNames(container.Env))
	assert.Equal(t, `[{"service":"app"}]`, envValues(container.Env)[flameshotProcessesKey])
	assert.Equal(t, "true", pod.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "8089", pod.Annotations["prometheus.io/port"])

	afterFirstInjection := pod.DeepCopy()
	changed, err = InjectFlameshotToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, afterFirstInjection, pod)
}

func TestInjectFlameshotRequiresCompleteConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.FlameshotRule)
	}{
		{"processes", func(rule *config.FlameshotRule) { rule.Processes = "" }},
		{"profiling path", func(rule *config.FlameshotRule) { rule.Envs = rule.Envs[1:] }},
		{"port", func(rule *config.FlameshotRule) { rule.Envs[2].Value = "invalid" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rule := testFlameshotRule()
			tc.mutate(rule)
			useFlameshotRule(t, rule)
			pod := createTestPod("incomplete", nil)
			before := pod.DeepCopy()

			changed, err := InjectFlameshotToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, before, pod)
		})
	}
}

func TestInjectFlameshotCanBeDisabled(t *testing.T) {
	useFlameshotRule(t, testFlameshotRule())
	pod := createTestPod("disabled", map[string]string{flameshotEnabledAnnotationKey: "false"})
	before := pod.DeepCopy()

	changed, err := InjectFlameshotToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, before, pod)
}

func TestInjectFlameshotRejectsNilPod(t *testing.T) {
	changed, err := InjectFlameshotToPod("default", "nil", nil)
	assert.False(t, changed)
	assert.Error(t, err)
}
