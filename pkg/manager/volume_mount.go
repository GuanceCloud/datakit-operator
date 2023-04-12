package manager

import (
	corev1 "k8s.io/api/core/v1"
)

type VolumeMountManager interface {
	AddVolumeMount(newVolumeMount *corev1.VolumeMount)
	AddVolumeMountToContainer(containerName string, newVolumeMount *corev1.VolumeMount)
	AddVolumeMountToInitContainer(containerName string, newVolumeMount *corev1.VolumeMount)
}

func NewVolumeMountManager(podTemplate *corev1.PodTemplateSpec) VolumeMountManager {
	return &volumeMountManager{podTemplate}
}

type volumeMountManager struct {
	podTemplate *corev1.PodTemplateSpec
}

func (m *volumeMountManager) AddVolumeMount(newVolumeMount *corev1.VolumeMount) {
	for idx := range m.podTemplate.Spec.Containers {
		_ = AddVolumeMountToContainer(&m.podTemplate.Spec.Containers[idx], newVolumeMount)
	}
}

func (m *volumeMountManager) AddVolumeMountToContainer(containerName string, newVolumeMount *corev1.VolumeMount) {
	for idx := range m.podTemplate.Spec.Containers {
		if m.podTemplate.Spec.Containers[idx].Name == containerName {
			_ = AddVolumeMountToContainer(&m.podTemplate.Spec.Containers[idx], newVolumeMount)
		}
	}
}

func (m *volumeMountManager) AddVolumeMountToInitContainer(containerName string, newVolumeMount *corev1.VolumeMount) {
	for idx := range m.podTemplate.Spec.InitContainers {
		if m.podTemplate.Spec.InitContainers[idx].Name == containerName {
			_ = AddVolumeMountToContainer(&m.podTemplate.Spec.InitContainers[idx], newVolumeMount)
		}
	}
}

func AddVolumeMountToContainer(container *corev1.Container, newVolumeMount *corev1.VolumeMount) []corev1.VolumeMount {
	found := false
	for idx := range container.VolumeMounts {
		if container.VolumeMounts[idx].Name == newVolumeMount.Name {
			found = true
		}
	}
	if !found {
		container.VolumeMounts = append(container.VolumeMounts, *newVolumeMount)
	}
	return container.VolumeMounts
}
