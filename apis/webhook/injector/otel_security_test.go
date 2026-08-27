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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
)

func TestInjectOTelUsesEffectiveSecurityContext(t *testing.T) {
	nonRoot, rootAllowed := true, false
	nonRootUID, rootUID := int64(1000), int64(0)
	tests := []struct {
		name      string
		pod       *corev1.PodSecurityContext
		container *corev1.SecurityContext
		want      bool
	}{
		{"pod requires non-root without UID", &corev1.PodSecurityContext{RunAsNonRoot: &nonRoot}, nil, false},
		{"pod supplies non-root UID", &corev1.PodSecurityContext{RunAsNonRoot: &nonRoot, RunAsUser: &nonRootUID}, nil, true},
		{"container disables pod requirement", &corev1.PodSecurityContext{RunAsNonRoot: &nonRoot}, &corev1.SecurityContext{RunAsNonRoot: &rootAllowed}, true},
		{"container inherits pod UID", &corev1.PodSecurityContext{RunAsUser: &nonRootUID}, &corev1.SecurityContext{RunAsNonRoot: &nonRoot}, true},
		{"container root UID overrides pod", &corev1.PodSecurityContext{RunAsNonRoot: &nonRoot, RunAsUser: &nonRootUID}, &corev1.SecurityContext{RunAsUser: &rootUID}, false},
		{"container supplies non-root UID", &corev1.PodSecurityContext{RunAsNonRoot: &nonRoot}, &corev1.SecurityContext{RunAsUser: &nonRootUID}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			useOTelRules(t, newTestOTelRule("example.com/otel-java:1"))
			pod := createTestPod("security-context", nil)
			pod.Spec.SecurityContext = tc.pod.DeepCopy()
			if tc.container != nil {
				pod.Spec.Containers[0].SecurityContext = tc.container.DeepCopy()
			}
			before := pod.DeepCopy()

			changed, err := InjectOTelToPod("default", pod.Name, pod)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, tc.want, changed)
			if tc.want {
				assert.Len(t, pod.Spec.InitContainers, 1)
			} else {
				assert.Equal(t, before, pod)
			}
		})
	}
}

func TestInjectOTelRunAsNonRootWarningIncludesTraceFields(t *testing.T) {
	var logs bytes.Buffer
	original := log
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), zapcore.AddSync(&logs), zap.WarnLevel)
	log = &logger.Logger{SugaredLogger: zap.New(core).Sugar()}
	t.Cleanup(func() { log = original })
	useOTelRules(t, newTestOTelRule("example.com/otel-java:1"))

	nonRoot := true
	pod := createTestPod("unsafe", nil)
	pod.Spec.SecurityContext = &corev1.PodSecurityContext{RunAsNonRoot: &nonRoot}
	changed, err := InjectOTelToPod("default", pod.Name, pod)
	if !assert.NoError(t, err) {
		return
	}
	assert.False(t, changed)

	for _, fragment := range []string{
		"pod=unsafe", "namespace=default", "rule=otel-java", "language=java",
		"image=example.com/otel-java:1", "reason=run_as_non_root_without_run_as_user",
	} {
		assert.Contains(t, logs.String(), fragment)
	}
}
