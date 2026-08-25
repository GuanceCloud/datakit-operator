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
	return newTestOTelRuleForLanguage("java", image)
}

func newTestOTelRuleForLanguage(language, image string) *config.OTelRule {
	return &config.OTelRule{
		InjectRule: config.InjectRule{
			Name:  "otel-" + language,
			Image: image,
			Resources: config.ResourceRequirements{
				Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
				Limits:   config.ResourceQuotaConfig{CPU: "500m", Memory: "512Mi"},
			},
		},
		Language: language,
	}
}

func useOTelRules(t *testing.T, rules ...*config.OTelRule) {
	t.Helper()
	original := otelMatchAllNamespaceOrLabelsForConfig
	otelMatchAllNamespaceOrLabelsForConfig = func(string, map[string]string) (bool, []*config.OTelRule) {
		return true, rules
	}
	t.Cleanup(func() { otelMatchAllNamespaceOrLabelsForConfig = original })
}

func TestInjectOTelLanguages(t *testing.T) {
	tests := []struct {
		language string
		image    string
		envName  string
		existing string
		marker   string
		mount    string
	}{
		{"java", "example.com/otel-java:1", otelJavaToolOptionsKey, "-Xmx256m", otelJavaAgentOption, otelJavaMountPath},
		{"python", "example.com/otel-python:1", otelPythonPathKey, "/app", otelPythonPathPrefix, otelPythonMountPath},
		{"nodejs", "example.com/otel-node:1", otelNodeJSOptionsKey, "--max-old-space-size=256", otelNodeJSRequireOption, otelNodeJSMountPath},
	}

	for _, tc := range tests {
		t.Run(tc.language, func(t *testing.T) {
			rule := newTestOTelRuleForLanguage(tc.language, tc.image)
			rule.Envs = config.Envs{{Key: "OTEL_SERVICE_NAME", Value: "checkout"}}
			useOTelRules(t, rule)
			pod := createTestPod("otel-"+tc.language, nil)
			pod.Spec.Containers[0].Env = []corev1.EnvVar{{Name: tc.envName, Value: tc.existing}}

			changed, err := InjectOTelToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.True(t, changed)
			if assert.Len(t, pod.Spec.InitContainers, 1) {
				assert.Equal(t, tc.image, pod.Spec.InitContainers[0].Image)
			}
			assert.Len(t, pod.Spec.Volumes, 1)
			if assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 1) {
				assert.Equal(t, tc.mount, pod.Spec.Containers[0].VolumeMounts[0].MountPath)
			}
			envs := envValues(pod.Spec.Containers[0].Env)
			assert.Contains(t, envs[tc.envName], tc.existing)
			assert.Contains(t, envs[tc.envName], tc.marker)
			assert.Equal(t, "checkout", envs["OTEL_SERVICE_NAME"])

			afterFirstInjection := pod.DeepCopy()
			changed, err = InjectOTelToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, afterFirstInjection, pod)
		})
	}
}

func TestInjectOTelDoesNotFallbackPastFirstSelectedRule(t *testing.T) {
	useOTelRules(t,
		newTestOTelRuleForLanguage("unsupported", "example.com/unsupported:1"),
		newTestOTelRuleForLanguage("python", "example.com/otel-python:1"),
	)
	pod := createTestPod("first-rule", nil)
	before := pod.DeepCopy()

	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, before, pod)
}

func TestInjectOTelUsesVersionAnnotation(t *testing.T) {
	rule := newTestOTelRule("example.com/otel-java:1")
	rule.CheckAnnotation = true
	useOTelRules(t, rule)
	pod := createTestPod("versioned", map[string]string{
		"admission.datakit/otel-java-lib.version": "2",
	})

	changed, err := InjectOTelToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	if assert.Len(t, pod.Spec.InitContainers, 1) {
		assert.Equal(t, "example.com/otel-java:2", pod.Spec.InitContainers[0].Image)
	}
}
