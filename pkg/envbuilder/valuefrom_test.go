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

func TestConvertFieldPath(t *testing.T) {
	cases := []struct {
		in      string
		output  string
		refType refType
	}{
		{
			in:      "{fieldRef:metadata.name}",
			output:  "metadata.name",
			refType: refTypeFieldRef,
		},
		{
			in:      "{fieldRef:metadata.labels['app']}",
			output:  "metadata.labels['app']",
			refType: refTypeFieldRef,
		},
		{
			in:      "{fieldRef:metadata.annotations['app']}",
			output:  "metadata.annotations['app']",
			refType: refTypeFieldRef,
		},
		{
			in:      "{resourceFieldRef:limits.cpu}",
			output:  "limits.cpu",
			refType: refTypeResourceFieldRef,
		},
		{
			in:      "unmatched",
			output:  "",
			refType: "",
		},
	}

	for _, tc := range cases {
		res, rt := converFieldPath(tc.in)
		assert.Equal(t, tc.output, res)
		assert.Equal(t, tc.refType, rt)
	}
}

func TestFilterAndSetResourceFieldRefEnvVars(t *testing.T) {
	t.Run("pod is nil", func(t *testing.T) {
		envs := []corev1.EnvVar{
			{Name: "TEST", Value: "value"},
		}
		result := FilterAndSetResourceFieldRefEnvVars(envs, nil)
		assert.Equal(t, envs, result)
	})

	t.Run("pod has no containers", func(t *testing.T) {
		pod := &corev1.Pod{}
		envs := []corev1.EnvVar{
			{Name: "TEST", Value: "value"},
		}
		result := FilterAndSetResourceFieldRefEnvVars(envs, pod)
		assert.Equal(t, envs, result)
	})

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
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("256Mi"),
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
			{
				Name:  "NORMAL_ENV",
				Value: "normal-value",
			},
		}

		result := FilterAndSetResourceFieldRefEnvVars(envs, pod)
		assert.Len(t, result, 3)
		assert.Equal(t, "app-container", result[0].ValueFrom.ResourceFieldRef.ContainerName)
		assert.Equal(t, "app-container", result[1].ValueFrom.ResourceFieldRef.ContainerName)
		assert.Equal(t, "NORMAL_ENV", result[2].Name)
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

	t.Run("requests resourceFieldRef", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "app-container",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("256Mi"),
							},
						},
					},
				},
			},
		}

		envs := []corev1.EnvVar{
			{
				Name:  "REQUESTS_CPU",
				Value: "test",
				ValueFrom: &corev1.EnvVarSource{
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						Resource: "requests.cpu",
					},
				},
			},
			{
				Name:  "REQUESTS_MEMORY",
				Value: "test",
				ValueFrom: &corev1.EnvVarSource{
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						Resource: "requests.memory",
					},
				},
			},
		}

		result := FilterAndSetResourceFieldRefEnvVars(envs, pod)
		assert.Len(t, result, 2)
		assert.Equal(t, "app-container", result[0].ValueFrom.ResourceFieldRef.ContainerName)
		assert.Equal(t, "app-container", result[1].ValueFrom.ResourceFieldRef.ContainerName)
	})

	t.Run("container with no resources", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "app-container",
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
		assert.Len(t, result, 1)
		assert.Equal(t, "NORMAL_ENV", result[0].Name)
	})
}
