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
	cases := []struct {
		in     string
		output string
	}{
		{
			in:     "{fieldRef:metadata.name}",
			output: "metadata.name",
		},
		{
			in:     "{fieldRef:metadata.labels['app']}",
			output: "metadata.labels['app']",
		},
		{
			in:     "unmatched",
			output: "",
		},
	}

	for _, tc := range cases {
		res := converFieldRefPath(tc.in)
		assert.Equal(t, tc.output, res)
	}
}

func TestConvertResourceFieldRefPath(t *testing.T) {
	cases := []struct {
		in     string
		output string
	}{
		{
			in:     "{resourceFieldRef:limits.cpu}",
			output: "limits.cpu",
		},
		{
			in:     "{resourceFieldRef:requests.memory}",
			output: "requests.memory",
		},
		{
			in:     "unmatched",
			output: "",
		},
	}

	for _, tc := range cases {
		res := converResourceFieldRefPath(tc.in)
		assert.Equal(t, tc.output, res)
	}
}

func TestFilterAndSetResourceFieldRefEnvVars(t *testing.T) {
	t.Run("resourceFieldRef with matching resources", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "app-container",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1"),
								corev1.ResourceMemory: resource.MustParse("512Mi"),
							},
						},
					},
				},
			},
		}

		envs := []corev1.EnvVar{
			{
				Name:  "LIMITS_CPU",
				Value: "test",
				ValueFrom: &corev1.EnvVarSource{
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						Resource: "limits.cpu",
					},
				},
			},
			{
				Name:  "NORMAL_ENV",
				Value: "normal-value",
			},
		}

		result := FilterAndSetResourceFieldRefEnvVars(envs, pod)
		assert.Len(t, result, 2)
		assert.Equal(t, "app-container", result[0].ValueFrom.ResourceFieldRef.ContainerName)
		assert.Equal(t, "NORMAL_ENV", result[1].Name)
	})

	t.Run("resourceFieldRef without matching resources", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "app-container",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("512Mi"),
							},
						},
					},
				},
			},
		}

		envs := []corev1.EnvVar{
			{
				Name:  "LIMITS_CPU",
				Value: "test",
				ValueFrom: &corev1.EnvVarSource{
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						Resource: "limits.cpu",
					},
				},
			},
			{
				Name:  "LIMITS_MEMORY",
				Value: "test",
				ValueFrom: &corev1.EnvVarSource{
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						Resource: "limits.memory",
					},
				},
			},
		}

		result := FilterAndSetResourceFieldRefEnvVars(envs, pod)
		assert.Len(t, result, 1)
		assert.Equal(t, "LIMITS_MEMORY", result[0].Name)
		assert.Equal(t, "app-container", result[0].ValueFrom.ResourceFieldRef.ContainerName)
	})
}
