// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package manager

import (
	corev1 "k8s.io/api/core/v1"
)

type VolumeManager interface {
	AddVolume(newVolume *corev1.Volume)
	IsEmptyDirVolume(name string) bool
}

func NewVolumeManager(pod *corev1.Pod) VolumeManager {
	return &volumeManagerImpl{pod}
}

type volumeManagerImpl struct {
	pod *corev1.Pod
}

func (m *volumeManagerImpl) AddVolume(newVolume *corev1.Volume) {
	for idx := range m.pod.Spec.Volumes {
		if m.pod.Spec.Volumes[idx].Name == newVolume.Name {
			return
		}
	}
	m.pod.Spec.Volumes = append(m.pod.Spec.Volumes, *newVolume)
}

func (m *volumeManagerImpl) IsEmptyDirVolume(name string) bool {
	for idx := range m.pod.Spec.Volumes {
		if m.pod.Spec.Volumes[idx].Name == name {
			return m.pod.Spec.Volumes[idx].EmptyDir != nil
		}
	}
	return false
}
