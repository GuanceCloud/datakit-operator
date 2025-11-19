// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package envbuilder

import (
	corev1 "k8s.io/api/core/v1"
)

func BuildEnvs(envs []struct{ Key, Value string }, useFieldRef bool) []corev1.EnvVar {
	var res []corev1.EnvVar

	for _, in := range envs {
		res = append(res, BuildEnv(in.Key, in.Value, useFieldRef))
	}

	return res
}

func BuildEnv(key, value string, useFieldRef bool) corev1.EnvVar {
	if !useFieldRef {
		return corev1.EnvVar{Name: key, Value: value}
	}

	envvarSource := newEnvVarSource(value)
	if envvarSource != nil {
		return corev1.EnvVar{Name: key, ValueFrom: envvarSource}
	}

	return corev1.EnvVar{Name: key, Value: value}
}
