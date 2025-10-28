// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package manager

import (
	corev1 "k8s.io/api/core/v1"
)

type EnvVarManager interface {
	AddEnvVar(newEnvVar *corev1.EnvVar)
	AddEnvVarToContainer(containerName string, newEnvVar *corev1.EnvVar)
	AddEnvVarToInitContainer(containerName string, newEnvVar *corev1.EnvVar)
}

func NewEnvVarManager(pod *corev1.Pod) EnvVarManager {
	return &envVarManagerImpl{pod}
}

type envVarManagerImpl struct {
	pod *corev1.Pod
}

func (m *envVarManagerImpl) AddEnvVar(newEnvVar *corev1.EnvVar) {
	for idx := range m.pod.Spec.Containers {
		_ = AddEnvVarToContainer(&m.pod.Spec.Containers[idx], newEnvVar)
	}
}

func (m *envVarManagerImpl) AddEnvVarToContainer(containerName string, newEnvVar *corev1.EnvVar) {
	for idx := range m.pod.Spec.Containers {
		if m.pod.Spec.Containers[idx].Name == containerName {
			_ = AddEnvVarToContainer(&m.pod.Spec.Containers[idx], newEnvVar)
		}
	}
}

func (m *envVarManagerImpl) AddEnvVarToInitContainer(initContainerName string, newEnvVar *corev1.EnvVar) {
	for idx := range m.pod.Spec.InitContainers {
		if m.pod.Spec.InitContainers[idx].Name == initContainerName {
			_ = AddEnvVarToContainer(&m.pod.Spec.InitContainers[idx], newEnvVar)
		}
	}
}

func AddEnvVarToContainer(container *corev1.Container, newEnvVar *corev1.EnvVar) []corev1.EnvVar {
	found := false
	for idx := range container.Env {
		if container.Env[idx].Name == newEnvVar.Name {
			found = true
		}
	}
	if !found {
		container.Env = append(container.Env, *newEnvVar)
	}
	return container.Env
}
