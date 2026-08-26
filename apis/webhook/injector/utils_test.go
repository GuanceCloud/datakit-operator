// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createTestPod(name string, annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Annotations: annotations},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: "nginx", Image: "nginx:1.22",
		}}},
	}
}

func fieldRefEnv(name string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{
			FieldPath: "metadata.name",
		}},
	}
}

func TestAppendKVPairs(t *testing.T) {
	cases := []struct {
		inOrigin string
		inNew    string
		output   string
	}{
		{
			inOrigin: "host:node01,name:nginx",
			inNew:    "app:system",
			output:   "host:node01,name:nginx,app:system",
		},
		{
			inOrigin: "host:node01,name:nginx",
			inNew:    "host:node02",
			output:   "host:node01,name:nginx",
		},
		{
			inOrigin: "",
			inNew:    "app:system",
			output:   "app:system",
		},
		{
			inOrigin: "host:node01,name:nginx",
			inNew:    "app::system",
			output:   "host:node01,name:nginx",
		},
	}

	for _, tc := range cases {
		res := appendKVPairs(tc.inOrigin, tc.inNew)
		assert.Equal(t, tc.output, res)
	}
}

func TestGetMountPath(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "path with wildcard",
			input:    "/data/logs/**/123",
			expected: "/data/logs",
		},
		{
			name:     "path without wildcard",
			input:    "/data/logs",
			expected: "/data/logs",
		},
		{
			name:     "root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "single level path",
			input:    "/data",
			expected: "/data",
		},
		{
			name:     "wildcard at beginning",
			input:    "*/logs",
			expected: ".",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := getMountPath(tc.input)
			assert.Equal(t, tc.expected, res)
		})
	}
}

func findEnv(envs []corev1.EnvVar, name string) (corev1.EnvVar, bool) {
	for _, env := range envs {
		if env.Name == name {
			return env, true
		}
	}
	return corev1.EnvVar{}, false
}

func envValues(envs []corev1.EnvVar) map[string]string {
	values := make(map[string]string, len(envs))
	for _, env := range envs {
		values[env.Name] = env.Value
	}
	return values
}

func envNames(envs []corev1.EnvVar) []string {
	names := make([]string, 0, len(envs))
	for _, env := range envs {
		names = append(names, env.Name)
	}
	return names
}

func countNamedEnv(envs []corev1.EnvVar, name string) int {
	count := 0
	for _, env := range envs {
		if env.Name == name {
			count++
		}
	}
	return count
}
