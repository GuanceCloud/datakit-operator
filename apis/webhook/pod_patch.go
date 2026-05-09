// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package webhook

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/mattbaird/jsonpatch"
	corev1 "k8s.io/api/core/v1"
)

func buildPodInjectionPatch(oldPod, newPod *corev1.Pod) []jsonpatch.JsonPatchOperation {
	if oldPod == nil || newPod == nil {
		return nil
	}

	var patches []jsonpatch.JsonPatchOperation
	patches = append(patches, buildAnnotationPatch(oldPod.Annotations, newPod.Annotations)...)
	patches = append(patches, buildPodSpecPatch(&oldPod.Spec, &newPod.Spec)...)
	return patches
}

func buildAnnotationPatch(oldAnnotations, newAnnotations map[string]string) []jsonpatch.JsonPatchOperation {
	if len(newAnnotations) == 0 {
		return nil
	}

	if len(oldAnnotations) == 0 {
		return []jsonpatch.JsonPatchOperation{addPatch("/metadata/annotations", newAnnotations)}
	}

	var patches []jsonpatch.JsonPatchOperation
	keys := make([]string, 0, len(newAnnotations))
	for key := range newAnnotations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		newValue := newAnnotations[key]
		oldValue, exists := oldAnnotations[key]
		if !exists {
			patches = append(patches, addPatch("/metadata/annotations/"+escapeJSONPointer(key), newValue))
			continue
		}
		if oldValue != newValue {
			patches = append(patches, replacePatch("/metadata/annotations/"+escapeJSONPointer(key), newValue))
		}
	}
	return patches
}

func buildPodSpecPatch(oldSpec, newSpec *corev1.PodSpec) []jsonpatch.JsonPatchOperation {
	var patches []jsonpatch.JsonPatchOperation

	if !boolPtrEqual(oldSpec.ShareProcessNamespace, newSpec.ShareProcessNamespace) && newSpec.ShareProcessNamespace != nil {
		op := addPatch
		if oldSpec.ShareProcessNamespace != nil {
			op = replacePatch
		}
		patches = append(patches, op("/spec/shareProcessNamespace", *newSpec.ShareProcessNamespace))
	}

	if oldSpec.RestartPolicy != newSpec.RestartPolicy && newSpec.RestartPolicy != "" {
		op := addPatch
		if oldSpec.RestartPolicy != "" {
			op = replacePatch
		}
		patches = append(patches, op("/spec/restartPolicy", newSpec.RestartPolicy))
	}

	patches = append(patches, buildNamedAppendPatch("/spec/volumes", oldSpec.Volumes, newSpec.Volumes, volumeName)...)
	patches = append(patches, buildNamedAppendPatch("/spec/initContainers", oldSpec.InitContainers, newSpec.InitContainers, containerName)...)
	patches = append(patches, buildNamedAppendPatch("/spec/containers", oldSpec.Containers, newSpec.Containers, containerName)...)
	patches = append(patches, buildContainerFieldPatch(oldSpec.Containers, newSpec.Containers)...)

	return patches
}

func buildContainerFieldPatch(oldContainers, newContainers []corev1.Container) []jsonpatch.JsonPatchOperation {
	newByName := map[string]corev1.Container{}
	for idx := range newContainers {
		newByName[newContainers[idx].Name] = newContainers[idx]
	}

	var patches []jsonpatch.JsonPatchOperation
	for idx := range oldContainers {
		oldContainer := oldContainers[idx]
		newContainer, exists := newByName[oldContainer.Name]
		if !exists {
			continue
		}

		basePath := fmt.Sprintf("/spec/containers/%d", idx)
		patches = append(patches, buildEnvPatch(basePath+"/env", oldContainer.Env, newContainer.Env)...)
		patches = append(patches, buildNamedAppendPatch(basePath+"/volumeMounts", oldContainer.VolumeMounts, newContainer.VolumeMounts, volumeMountName)...)
	}
	return patches
}

func buildEnvPatch(basePath string, oldEnvs, newEnvs []corev1.EnvVar) []jsonpatch.JsonPatchOperation {
	if len(newEnvs) == 0 {
		return nil
	}
	if len(oldEnvs) == 0 {
		return []jsonpatch.JsonPatchOperation{addPatch(basePath, newEnvs)}
	}
	if len(newEnvs) < len(oldEnvs) {
		return []jsonpatch.JsonPatchOperation{replacePatch(basePath, newEnvs)}
	}
	for idx := range oldEnvs {
		if !reflect.DeepEqual(oldEnvs[idx], newEnvs[idx]) {
			return []jsonpatch.JsonPatchOperation{replacePatch(basePath, newEnvs)}
		}
	}

	oldByName := map[string]int{}
	for idx := range oldEnvs {
		if _, exists := oldByName[oldEnvs[idx].Name]; !exists {
			oldByName[oldEnvs[idx].Name] = idx
		}
	}

	var patches []jsonpatch.JsonPatchOperation
	for idx := range newEnvs {
		newEnv := newEnvs[idx]
		_, exists := oldByName[newEnv.Name]
		if !exists {
			patches = append(patches, addPatch(basePath+"/-", newEnv))
		}
	}
	return patches
}

func buildNamedAppendPatch[T any](basePath string, oldItems, newItems []T, nameFn func(T) string) []jsonpatch.JsonPatchOperation {
	if len(newItems) == 0 {
		return nil
	}
	if len(oldItems) == 0 {
		return []jsonpatch.JsonPatchOperation{addPatch(basePath, newItems)}
	}

	oldNames := map[string]struct{}{}
	for _, item := range oldItems {
		oldNames[nameFn(item)] = struct{}{}
	}

	var patches []jsonpatch.JsonPatchOperation
	for _, item := range newItems {
		if _, exists := oldNames[nameFn(item)]; exists {
			continue
		}
		patches = append(patches, addPatch(basePath+"/-", item))
	}
	return patches
}

func addPatch(path string, value interface{}) jsonpatch.JsonPatchOperation {
	return jsonpatch.JsonPatchOperation{
		Operation: "add",
		Path:      path,
		Value:     value,
	}
}

func replacePatch(path string, value interface{}) jsonpatch.JsonPatchOperation {
	return jsonpatch.JsonPatchOperation{
		Operation: "replace",
		Path:      path,
		Value:     value,
	}
}

func escapeJSONPointer(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	return strings.ReplaceAll(s, "/", "~1")
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func volumeName(volume corev1.Volume) string {
	return volume.Name
}

func containerName(container corev1.Container) string {
	return container.Name
}

func volumeMountName(volumeMount corev1.VolumeMount) string {
	return volumeMount.Name
}
