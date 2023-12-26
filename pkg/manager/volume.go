package manager

import (
	corev1 "k8s.io/api/core/v1"
)

type VolumeManager interface {
	AddVolume(newVolume *corev1.Volume)
	IsEmptyDirVolume(name string) bool
}

func NewVolumeManager(pod *corev1.Pod) VolumeManager {
	return &volumeMangaer{pod}
}

type volumeMangaer struct {
	pod *corev1.Pod
}

func (m *volumeMangaer) AddVolume(newVolume *corev1.Volume) {
	for idx := range m.pod.Spec.Volumes {
		if m.pod.Spec.Volumes[idx].Name == newVolume.Name {
			return
		}
	}
	m.pod.Spec.Volumes = append(m.pod.Spec.Volumes, *newVolume)
}

func (m *volumeMangaer) IsEmptyDirVolume(name string) bool {
	for idx := range m.pod.Spec.Volumes {
		if m.pod.Spec.Volumes[idx].Name == name {
			return m.pod.Spec.Volumes[idx].EmptyDir != nil
		}
	}
	return false
}
