package manager

import (
	corev1 "k8s.io/api/core/v1"
)

type ContainerManager interface {
	AddContainer(newContainer *corev1.Container)
	AddInitContainer(newContainer *corev1.Container)
	ContainsContainer(containerName string) bool
	ContainsInitContainer(containerName string) bool
}

func NewContainerManager(podTemplate *corev1.PodTemplateSpec) ContainerManager {
	return &containerManager{podTemplate}
}

type containerManager struct {
	podTemplate *corev1.PodTemplateSpec
}

func (m *containerManager) AddContainer(newContainer *corev1.Container) {
	for idx := range m.podTemplate.Spec.Containers {
		if m.podTemplate.Spec.Containers[idx].Name == newContainer.Name {
			return
		}
	}
	m.podTemplate.Spec.Containers = append(m.podTemplate.Spec.Containers, *newContainer)
}

func (m *containerManager) AddInitContainer(newContainer *corev1.Container) {
	for idx := range m.podTemplate.Spec.InitContainers {
		if m.podTemplate.Spec.InitContainers[idx].Name == newContainer.Name {
			return
		}
	}
	m.podTemplate.Spec.InitContainers = append(m.podTemplate.Spec.InitContainers, *newContainer)
}

func (m *containerManager) ContainsContainer(containerName string) bool {
	for idx := range m.podTemplate.Spec.Containers {
		if m.podTemplate.Spec.Containers[idx].Name == containerName {
			return true
		}
	}
	return false
}

func (m *containerManager) ContainsInitContainer(containerName string) bool {
	for idx := range m.podTemplate.Spec.InitContainers {
		if m.podTemplate.Spec.InitContainers[idx].Name == containerName {
			return true
		}
	}
	return false
}
