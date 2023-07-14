package manager

import (
	corev1 "k8s.io/api/core/v1"
)

type VolumeManager interface {
	AddVolume(newVolume *corev1.Volume)
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
