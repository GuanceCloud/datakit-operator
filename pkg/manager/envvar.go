package manager

import (
	corev1 "k8s.io/api/core/v1"
)

type EnvVarManager interface {
	AddEnvVar(newEnvVar *corev1.EnvVar)
	AddEnvVarToContainer(containerName string, newEnvVar *corev1.EnvVar)
	AddEnvVarToInitContainer(containerName string, newEnvVar *corev1.EnvVar)
}

func NewEnvVarManager(podTemplate *corev1.PodTemplateSpec) EnvVarManager {
	return &envVarMangaer{podTemplate}
}

type envVarMangaer struct {
	podTemplate *corev1.PodTemplateSpec
}

func (m *envVarMangaer) AddEnvVar(newEnvVar *corev1.EnvVar) {
	for idx := range m.podTemplate.Spec.Containers {
		_ = AddEnvVarToContainer(&m.podTemplate.Spec.Containers[idx], newEnvVar)
	}
}

func (m *envVarMangaer) AddEnvVarToContainer(containerName string, newEnvVar *corev1.EnvVar) {
	for idx := range m.podTemplate.Spec.Containers {
		if m.podTemplate.Spec.Containers[idx].Name == containerName {
			_ = AddEnvVarToContainer(&m.podTemplate.Spec.Containers[idx], newEnvVar)
		}
	}
}

func (m *envVarMangaer) AddEnvVarToInitContainer(initContainerName string, newEnvVar *corev1.EnvVar) {
	for idx := range m.podTemplate.Spec.InitContainers {
		if m.podTemplate.Spec.InitContainers[idx].Name == initContainerName {
			_ = AddEnvVarToContainer(&m.podTemplate.Spec.InitContainers[idx], newEnvVar)
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
