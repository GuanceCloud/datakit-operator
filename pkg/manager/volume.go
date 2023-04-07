package manager

import (
	corev1 "k8s.io/api/core/v1"
)

type VolumeManager interface {
	AddVolume(newVolume *corev1.Volume)
}

func NewVolumeManager(podTemplate *corev1.PodTemplateSpec) VolumeManager {
	return &volumeMangaer{podTemplate}
}

type volumeMangaer struct {
	podTemplate *corev1.PodTemplateSpec
}

func (m *volumeMangaer) AddVolume(newVolume *corev1.Volume) {
	for idx := range m.podTemplate.Spec.Volumes {
		if m.podTemplate.Spec.Volumes[idx].Name == newVolume.Name {
			return
		}
	}
	m.podTemplate.Spec.Volumes = append(m.podTemplate.Spec.Volumes, *newVolume)
}
