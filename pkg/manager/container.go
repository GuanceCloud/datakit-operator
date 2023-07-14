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

func NewContainerManager(pod *corev1.Pod) ContainerManager {
	return &containerManager{pod}
}

type containerManager struct {
	pod *corev1.Pod
}

func (m *containerManager) AddContainer(newContainer *corev1.Container) {
	for idx := range m.pod.Spec.Containers {
		if m.pod.Spec.Containers[idx].Name == newContainer.Name {
			return
		}
	}
	m.pod.Spec.Containers = append(m.pod.Spec.Containers, *newContainer)
}

func (m *containerManager) AddInitContainer(newContainer *corev1.Container) {
	for idx := range m.pod.Spec.InitContainers {
		if m.pod.Spec.InitContainers[idx].Name == newContainer.Name {
			return
		}
	}
	m.pod.Spec.InitContainers = append(m.pod.Spec.InitContainers, *newContainer)
}

func (m *containerManager) ContainsContainer(containerName string) bool {
	for idx := range m.pod.Spec.Containers {
		if m.pod.Spec.Containers[idx].Name == containerName {
			return true
		}
	}
	return false
}

func (m *containerManager) ContainsInitContainer(containerName string) bool {
	for idx := range m.pod.Spec.InitContainers {
		if m.pod.Spec.InitContainers[idx].Name == containerName {
			return true
		}
	}
	return false
}
