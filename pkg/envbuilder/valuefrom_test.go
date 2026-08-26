// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package envbuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestConvertFieldRefPath(t *testing.T) {
	tests := map[string]string{
		"{fieldRef:metadata.name}":             "metadata.name",
		"{fieldRef:metadata.labels['app']}":    "metadata.labels['app']",
		"prefix{fieldRef:metadata.name}suffix": "",
		"{fieldRef:metadataXlabels['app']}":    "",
		"unmatched":                            "",
	}
	for input, expected := range tests {
		assert.Equal(t, expected, convertFieldRefPath(input))
	}
}

func TestConvertResourceFieldRefPath(t *testing.T) {
	tests := map[string]string{
		"{resourceFieldRef:limits.cpu}":      "limits.cpu",
		"{resourceFieldRef:requests.memory}": "requests.memory",
		"unmatched":                          "",
	}
	for input, expected := range tests {
		assert.Equal(t, expected, convertResourceFieldRefPath(input))
	}
}

func TestConvertSecretKeyRef(t *testing.T) {
	name, key, ok := convertSecretKeyRef("{secretKeyRef:flameshot-oss.access_key_id}")
	assert.True(t, ok)
	assert.Equal(t, "flameshot-oss", name)
	assert.Equal(t, "access_key_id", key)

	for _, invalid := range []string{
		"{secretKeyRef:flameshot-oss}",
		"{secretKeyRef:FLAMESHOT-OSS.access_key_id}",
		"{secretKeyRef:flameshot-oss.access/key}",
		"prefix{secretKeyRef:flameshot-oss.access_key_id}suffix",
	} {
		name, key, ok := convertSecretKeyRef(invalid)
		assert.False(t, ok, invalid)
		assert.Empty(t, name, invalid)
		assert.Empty(t, key, invalid)
	}
}

func TestBuildEnvWithSecretKeyRef(t *testing.T) {
	got := BuildEnv("ACCESS_KEY", "{secretKeyRef:flameshot-oss.access_key_id}", true)
	assert.Equal(t, corev1.EnvVar{
		Name: "ACCESS_KEY",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "flameshot-oss"},
				Key:                  "access_key_id",
			},
		},
	}, got)

	invalid := "{secretKeyRef:INVALID-NAME.access/key}"
	assert.Equal(t, corev1.EnvVar{Name: "ACCESS_KEY", Value: invalid}, BuildEnv("ACCESS_KEY", invalid, true))
}

func TestFilterAndSetResourceFieldRefEnvVars(t *testing.T) {
	pod := &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{
		Name: "app",
		Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		}},
	}}}}
	envs := []corev1.EnvVar{
		{
			Name: "LIMITS_CPU",
			ValueFrom: &corev1.EnvVarSource{ResourceFieldRef: &corev1.ResourceFieldSelector{
				Resource: "limits.cpu",
			}},
		},
		{
			Name: "LIMITS_MEMORY",
			ValueFrom: &corev1.EnvVarSource{ResourceFieldRef: &corev1.ResourceFieldSelector{
				Resource: "limits.memory",
			}},
		},
		{Name: "NORMAL_ENV", Value: "normal-value"},
	}

	result := FilterAndSetResourceFieldRefEnvVars(envs, pod)
	if assert.Len(t, result, 2) {
		assert.Equal(t, "LIMITS_MEMORY", result[0].Name)
		assert.Equal(t, "app", result[0].ValueFrom.ResourceFieldRef.ContainerName)
		assert.Equal(t, "NORMAL_ENV", result[1].Name)
	}
}
