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

func testProfilerRule() *config.ProfilerRule {
	return &config.ProfilerRule{
		InjectRule: config.InjectRule{
			Envs: config.Envs{
				{Key: "FIRST", Value: "first"},
				{Key: "DUPLICATE", Value: "old"},
				{Key: "DK_AGENT_HOST", Value: "datakit-service.datakit.svc"},
				{Key: "LAST", Value: "$(FIRST)"},
				{Key: "DUPLICATE", Value: "new"},
			},
			Resources: config.ResourceRequirements{
				Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
				Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
			},
		},
		Language: "java",
		Images: map[string]string{
			config.DeprecatedProfilerJavaImageKey: "example.com/java-profiler:1",
		},
	}
}

func useProfilerRule(t *testing.T, rule *config.ProfilerRule) {
	t.Helper()
	original := profilerMatchNamespaceOrLabelsForConfig
	profilerMatchNamespaceOrLabelsForConfig = func(string, map[string]string) (bool, *config.ProfilerRule) {
		return true, rule
	}
	t.Cleanup(func() { profilerMatchNamespaceOrLabelsForConfig = original })
}

func TestInjectProfiler(t *testing.T) {
	useProfilerRule(t, testProfilerRule())
	pod := createTestPod("profiler", nil)

	changed, err := InjectProfilerToPod("default", pod.Name, pod)
	if !assert.NoError(t, err) || !assert.True(t, changed) || !assert.NotNil(t, pod.Spec.ShareProcessNamespace) {
		return
	}
	assert.True(t, *pod.Spec.ShareProcessNamespace)
	assert.Len(t, pod.Spec.Volumes, 3)
	if !assert.Len(t, pod.Spec.Containers, 2) {
		return
	}
	profiler := pod.Spec.Containers[1]
	assert.Equal(t, profilerContainerName, profiler.Name)
	assert.Equal(t, "example.com/java-profiler:1", profiler.Image)
	assert.Equal(t, []string{"FIRST", "DUPLICATE", "DK_AGENT_HOST", "LAST"}, envNames(profiler.Env))
	assert.Equal(t, "new", envValues(profiler.Env)["DUPLICATE"])
	assert.Equal(t, "datakit-service.datakit.svc", envValues(profiler.Env)["DK_AGENT_HOST"])

	afterFirstInjection := pod.DeepCopy()
	changed, err = InjectProfilerToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, afterFirstInjection, pod)
}

func TestInjectProfilerHonorsVersionAnnotation(t *testing.T) {
	rule := testProfilerRule()
	rule.CheckAnnotation = true
	useProfilerRule(t, rule)

	withoutVersion := createTestPod("without-version", nil)
	changed, err := InjectProfilerToPod("default", withoutVersion.Name, withoutVersion)
	assert.NoError(t, err)
	assert.False(t, changed)

	withVersion := createTestPod("with-version", map[string]string{
		"admission.datakit/java-profiler.version": "2",
	})
	changed, err = InjectProfilerToPod("default", withVersion.Name, withVersion)
	if !assert.NoError(t, err) || !assert.True(t, changed) || !assert.Len(t, withVersion.Spec.Containers, 2) {
		return
	}
	assert.Equal(t, "example.com/java-profiler:2", withVersion.Spec.Containers[1].Image)
}

func TestInjectProfilerCanBeDisabled(t *testing.T) {
	useProfilerRule(t, testProfilerRule())
	pod := createTestPod("disabled", map[string]string{profilerEnabledAnnotationKey: "false"})
	before := pod.DeepCopy()

	changed, err := InjectProfilerToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, before, pod)
}

func TestInjectProfilerRejectsNilPod(t *testing.T) {
	changed, err := InjectProfilerToPod("default", "nil", nil)
	assert.False(t, changed)
	assert.Error(t, err)
}
