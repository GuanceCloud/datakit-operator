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

func TestInjectDDTraceLanguages(t *testing.T) {
	tests := []struct {
		language, envName, existing, expected string
		phpLoaderFlavor                       string
	}{
		{
			language: "python", envName: "PYTHONPATH", existing: "/app",
			expected: "/datadog-lib/:/app",
		},
		{
			language: "php", envName: "PHP_INI_SCAN_DIR", existing: "/app/php",
			expected: "/app/php:/datadog-lib", phpLoaderFlavor: phpLoaderFlavorMusl,
		},
		{
			language: "nodejs", envName: "NODE_OPTIONS", existing: "--max-old-space-size=512",
			expected: "--max-old-space-size=512 --require=/datadog-lib/node_modules/dd-trace/init",
		},
	}

	for _, tc := range tests {
		t.Run(tc.language, func(t *testing.T) {
			rule := newTestDDTraceRule(tc.language, "example.com/dd-"+tc.language+":1")
			rule.PHPLoaderFlavor = tc.phpLoaderFlavor
			useDDTraceRule(t, rule)
			pod := createTestPod("ddtrace-"+tc.language, nil)
			pod.Spec.Containers[0].Env = []corev1.EnvVar{{Name: tc.envName, Value: tc.existing}}

			changed, err := InjectDDTraceToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.True(t, changed)
			if assert.Len(t, pod.Spec.InitContainers, 1) {
				assert.Equal(t, rule.Image, pod.Spec.InitContainers[0].Image)
			}
			assert.Len(t, pod.Spec.Volumes, 1)
			if assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 1) {
				assert.Equal(t, ddtraceMountPath, pod.Spec.Containers[0].VolumeMounts[0].MountPath)
			}
			env, ok := findEnv(pod.Spec.Containers[0].Env, tc.envName)
			if assert.True(t, ok) {
				assert.Equal(t, tc.expected, env.Value)
			}

			if tc.language == "php" {
				assert.Contains(t, pod.Spec.InitContainers[0].Command[2], "/linux-musl/loader/")
				loader, ok := findEnv(pod.Spec.Containers[0].Env, "DD_LOADER_PACKAGE_PATH")
				if assert.True(t, ok) {
					assert.Equal(t, ddtraceMountPath, loader.Value)
				}
			}
		})
	}
}

func TestInjectDDTraceConfigErrorIsAtomic(t *testing.T) {
	for _, tc := range []struct {
		language, envName string
	}{
		{"java", javaToolOptionsKey},
		{"python", "PYTHONPATH"},
		{"php", "PHP_INI_SCAN_DIR"},
		{"nodejs", "NODE_OPTIONS"},
	} {
		t.Run(tc.language, func(t *testing.T) {
			useDDTraceRule(t, newTestDDTraceRule(tc.language, "example.com/dd-"+tc.language+":1"))
			pod := createTestPod("atomic-"+tc.language, nil)
			pod.Spec.Containers[0].Env = []corev1.EnvVar{fieldRefEnv(tc.envName)}
			before := pod.DeepCopy()

			changed, err := InjectDDTraceToPod("default", pod.Name, pod)
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, before, pod)
		})
	}
}

func TestInjectDDTraceVersionAnnotationSelectsLaterRule(t *testing.T) {
	javaRule := newTestDDTraceRule("java", "example.com/dd-java:1")
	javaRule.CheckAnnotation = true
	pythonRule := newTestDDTraceRule("python", "example.com/dd-python:1")
	pythonRule.CheckAnnotation = true
	original := ddtraceMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(string, map[string]string) (bool, []*config.DDTraceRule) {
		return true, []*config.DDTraceRule{javaRule, pythonRule}
	}
	t.Cleanup(func() { ddtraceMatchAllNamespaceOrLabelsForConfig = original })
	pod := createTestPod("versioned-python", map[string]string{
		"admission.datakit/python-lib.version": "2.0.0",
	})

	changed, err := InjectDDTraceToPod("default", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	if assert.Len(t, pod.Spec.InitContainers, 1) {
		assert.Equal(t, "example.com/dd-python:2.0.0", pod.Spec.InitContainers[0].Image)
	}
	_, hasPythonPath := findEnv(pod.Spec.Containers[0].Env, "PYTHONPATH")
	_, hasJavaOptions := findEnv(pod.Spec.Containers[0].Env, javaToolOptionsKey)
	assert.True(t, hasPythonPath)
	assert.False(t, hasJavaOptions)
}
