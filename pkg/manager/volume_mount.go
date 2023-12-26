package manager

import (
	corev1 "k8s.io/api/core/v1"
)

type VolumeMountManager interface {
	AddVolumeMount(newVolumeMount *corev1.VolumeMount)
	AddVolumeMountToContainer(containerName string, newVolumeMount *corev1.VolumeMount)
	AddVolumeMountToInitContainer(containerName string, newVolumeMount *corev1.VolumeMount)
	ContainsVolumeMountInContainer(mountPath string) (name string)
}

func NewVolumeMountManager(pod *corev1.Pod) VolumeMountManager {
	return &volumeMountManager{pod}
}

type volumeMountManager struct {
	pod *corev1.Pod
}

func (m *volumeMountManager) AddVolumeMount(newVolumeMount *corev1.VolumeMount) {
	for idx := range m.pod.Spec.Containers {
		_ = AddVolumeMountToContainer(&m.pod.Spec.Containers[idx], newVolumeMount)
	}
}

func (m *volumeMountManager) AddVolumeMountToContainer(containerName string, newVolumeMount *corev1.VolumeMount) {
	for idx := range m.pod.Spec.Containers {
		if m.pod.Spec.Containers[idx].Name == containerName {
			_ = AddVolumeMountToContainer(&m.pod.Spec.Containers[idx], newVolumeMount)
		}
	}
}

func (m *volumeMountManager) AddVolumeMountToInitContainer(containerName string, newVolumeMount *corev1.VolumeMount) {
	for idx := range m.pod.Spec.InitContainers {
		if m.pod.Spec.InitContainers[idx].Name == containerName {
			_ = AddVolumeMountToContainer(&m.pod.Spec.InitContainers[idx], newVolumeMount)
		}
	}
}

func (m *volumeMountManager) ContainsVolumeMountInContainer(mountPath string) (name string) {
	for idx := range m.pod.Spec.Containers {
		for _, mount := range m.pod.Spec.Containers[idx].VolumeMounts {
			if mount.MountPath == mountPath {
				return mount.Name
			}
		}
	}
	return ""
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
