// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestInjectOTelRejectsConflictingEnvironments(t *testing.T) {
	tests := []struct {
		name, language string
		envs           []corev1.EnvVar
	}{
		{"java valueFrom", "java", []corev1.EnvVar{fieldRefEnv(otelJavaToolOptionsKey)}},
		{"java Datadog agent", "java", []corev1.EnvVar{{Name: otelJavaToolOptionsKey, Value: "-javaagent:/datadog-lib/dd-java-agent.jar"}}},
		{"python valueFrom", "python", []corev1.EnvVar{fieldRefEnv(otelPythonPathKey)}},
		{"python Datadog path", "python", []corev1.EnvVar{{Name: otelPythonPathKey, Value: "/app:/datadog-lib"}}},
		{"python duplicate", "python", []corev1.EnvVar{{Name: otelPythonPathKey, Value: "/app"}, {Name: otelPythonPathKey, Value: "/vendor"}}},
		{"node valueFrom", "nodejs", []corev1.EnvVar{fieldRefEnv(otelNodeJSOptionsKey)}},
		{"node duplicate", "nodejs", []corev1.EnvVar{{Name: otelNodeJSOptionsKey, Value: "--trace-warnings"}, {Name: otelNodeJSOptionsKey, Value: "--max-old-space-size=512"}}},
		{"node incompatible require", "nodejs", []corev1.EnvVar{{Name: otelNodeJSOptionsKey, Value: "--require=/otel-auto-instrumentation-nodejs/autoinstrumentation.js"}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			useOTelRules(t, newTestOTelRuleForLanguage(tc.language, "example.com/otel:1"))
			pod := createTestPod("conflict", nil)
			pod.Spec.Containers[0].Env = tc.envs
			before := pod.DeepCopy()

			changed, err := InjectOTelToPod("default", pod.Name, pod)
			if !assert.NoError(t, err) {
				return
			}
			assert.False(t, changed)
			assert.Equal(t, before, pod)
		})
	}
}

func TestInjectOTelReinvocationHandlesDefaultsAndLateSidecar(t *testing.T) {
	tests := []struct {
		language, envName, expected string
	}{
		{"java", otelJavaToolOptionsKey, otelJavaAgentOption},
		{"python", otelPythonPathKey, otelPythonPathPrefix + ":" + otelPythonMountPath},
		{"nodejs", otelNodeJSOptionsKey, " " + otelNodeJSRequireOption},
	}

	for _, tc := range tests {
		t.Run(tc.language, func(t *testing.T) {
			useOTelRules(t, newTestOTelRuleForLanguage(tc.language, "example.com/otel:1"))
			pod := createTestPod("reinvocation", nil)
			nonRoot, uid := true, int64(1000)
			pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{RunAsNonRoot: &nonRoot, RunAsUser: &uid}
			changed, err := InjectOTelToPod("default", pod.Name, pod)
			if !assert.NoError(t, err) || !assert.True(t, changed) || !assert.Len(t, pod.Spec.InitContainers, 1) {
				return
			}

			allowPrivilegeEscalation := false
			init := &pod.Spec.InitContainers[0]
			if !assert.NotNil(t, init.SecurityContext) {
				return
			}
			init.TerminationMessagePath = "/dev/termination-log"
			init.SecurityContext.AllowPrivilegeEscalation = &allowPrivilegeEscalation
			init.Env = append(init.Env, corev1.EnvVar{Name: "WEBHOOK_ADDED", Value: "true"})
			init.VolumeMounts = append(init.VolumeMounts, corev1.VolumeMount{Name: "api-access", MountPath: "/var/run/secrets"})
			pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: "late-sidecar", Image: "sidecar:1"})

			changed, err = InjectOTelToPod("default", pod.Name, pod)
			if !assert.NoError(t, err) {
				return
			}
			assert.True(t, changed)
			assert.Len(t, pod.Spec.InitContainers, 1)
			late := pod.Spec.Containers[len(pod.Spec.Containers)-1]
			assert.Equal(t, tc.expected, envValues(late.Env)[tc.envName])
			assert.Len(t, late.VolumeMounts, 1)
		})
	}
}
