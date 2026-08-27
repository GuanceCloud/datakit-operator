// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package manager

import (
	corev1 "k8s.io/api/core/v1"
)

// EnvVarConflictResolver decides how to combine an incoming environment
// variable with an existing variable of the same name.
type EnvVarConflictResolver func(existing, incoming corev1.EnvVar) corev1.EnvVar

// KeepExistingEnvVar leaves an existing environment variable unchanged.
func KeepExistingEnvVar(existing, _ corev1.EnvVar) corev1.EnvVar {
	return existing
}

// ReplaceExistingEnvVar replaces an existing variable without changing its
// position in the environment list.
func ReplaceExistingEnvVar(_, incoming corev1.EnvVar) corev1.EnvVar {
	return incoming
}

// AddOrUpdateEnvVar preserves an existing environment variable's position. If
// the variable does not exist, it is appended to the end of the slice.
func AddOrUpdateEnvVar(envs []corev1.EnvVar, incoming corev1.EnvVar, resolve EnvVarConflictResolver) []corev1.EnvVar {
	for idx := range envs {
		if envs[idx].Name != incoming.Name {
			continue
		}

		if resolve == nil {
			return envs
		}
		envs[idx] = resolve(envs[idx], incoming)
		return envs
	}

	return append(envs, incoming)
}

// AddOrUpdateEnvVars applies incoming variables in order. Missing variables
// are therefore appended in the same order in which they are provided.
func AddOrUpdateEnvVars(envs, incoming []corev1.EnvVar, resolve EnvVarConflictResolver) []corev1.EnvVar {
	for idx := range incoming {
		envs = AddOrUpdateEnvVar(envs, incoming[idx], resolve)
	}
	return envs
}
