// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package envbuilder

import (
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var fieldRefValuefroms = []struct {
	key         string
	keyRe       *regexp.Regexp
	toFieldPath func(args string) string
}{
	{
		key:         "{fieldRef:metadata.name}",
		toFieldPath: func(_ string) string { return "metadata.name" },
	},
	{
		key:         "{fieldRef:metadata.namespace}",
		toFieldPath: func(_ string) string { return "metadata.namespace" },
	},
	{
		key:         "{fieldRef:metadata.uid}",
		toFieldPath: func(_ string) string { return "metadata.uid" },
	},
	{
		key:         "{fieldRef:spec.serviceAccountName}",
		toFieldPath: func(_ string) string { return "spec.serviceAccountName" },
	},
	{
		key:         "{fieldRef:spec.nodeName}",
		toFieldPath: func(_ string) string { return "spec.nodeName" },
	},
	{
		key:         "{fieldRef:status.hostIP}",
		toFieldPath: func(_ string) string { return "status.hostIP" },
	},
	{
		key:         "{fieldRef:status.hostIPs}",
		toFieldPath: func(_ string) string { return "status.hostIPs" },
	},
	{
		key:         "{fieldRef:status.podIP}",
		toFieldPath: func(_ string) string { return "status.podIP" },
	},
	{
		// e.g. {fieldRef:metadata.labels['app']}
		keyRe: regexp.MustCompile(`{fieldRef:metadata.labels\['(.+)'\]}`),
		toFieldPath: func(s string) string {
			return fmt.Sprintf("metadata.labels['%s']", s)
		},
	},
	{
		// e.g. {fieldRef:metadata.annotations['app']}
		keyRe: regexp.MustCompile(`{fieldRef:metadata.annotations\['(.+)'\]}`),
		toFieldPath: func(s string) string {
			return fmt.Sprintf("metadata.annotations['%s']", s)
		},
	},
}

var resourceFieldRefValuefroms = []struct {
	key         string
	toFieldPath func(args string) string
}{
	{
		key:         "{resourceFieldRef:limits.cpu}",
		toFieldPath: func(_ string) string { return "limits.cpu" },
	},
	{
		key:         "{resourceFieldRef:limits.memory}",
		toFieldPath: func(_ string) string { return "limits.memory" },
	},
	{
		key:         "{resourceFieldRef:requests.cpu}",
		toFieldPath: func(_ string) string { return "requests.cpu" },
	},
	{
		key:         "{resourceFieldRef:requests.memory}",
		toFieldPath: func(_ string) string { return "requests.memory" },
	},
}

func converFieldRefPath(s string) string {
	for _, v := range fieldRefValuefroms {
		if v.key == s {
			return v.toFieldPath("")
		}

		if v.keyRe != nil && v.keyRe.MatchString(s) {
			args := v.keyRe.FindStringSubmatch(s)
			if len(args) != 2 {
				break
			}
			if !IsQualifiedName(args[1]) {
				break
			}
			return v.toFieldPath(args[1])
		}
	}
	return ""
}

func converResourceFieldRefPath(s string) string {
	for _, v := range resourceFieldRefValuefroms {
		if v.key == s {
			return v.toFieldPath("")
		}
	}
	return ""
}

func newEnvVarSource(v string) *corev1.EnvVarSource {
	// 先尝试 fieldRef
	fieldPath := converFieldRefPath(v)
	if fieldPath != "" {
		return &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{FieldPath: fieldPath},
		}
	}

	// 再尝试 resourceFieldRef
	resourcePath := converResourceFieldRefPath(v)
	if resourcePath != "" {
		var divisor resource.Quantity
		if strings.Contains(resourcePath, "cpu") {
			divisor = resource.MustParse("1m")
		} else if strings.Contains(resourcePath, "memory") {
			divisor = resource.MustParse("1Mi")
		}
		return &corev1.EnvVarSource{
			ResourceFieldRef: &corev1.ResourceFieldSelector{ContainerName: "", Resource: resourcePath, Divisor: divisor},
		}
	}

	return nil
}

// FilterAndSetResourceFieldRefEnvVars 过滤并设置 resourceFieldRef 环境变量
// 1. 如果环境变量有 resourceFieldRef，设置第一个容器的名称到 ContainerName
// 2. 检查第一个容器是否有对应的资源限制，如果没有则从列表中移除该环境变量
func FilterAndSetResourceFieldRefEnvVars(envs []corev1.EnvVar, pod *corev1.Pod) []corev1.EnvVar {
	if pod == nil || len(pod.Spec.Containers) == 0 {
		return envs
	}

	firstContainer := &pod.Spec.Containers[0]
	firstContainerName := firstContainer.Name

	var filtered []corev1.EnvVar
	for _, env := range envs {
		if env.ValueFrom == nil || env.ValueFrom.ResourceFieldRef == nil {
			filtered = append(filtered, env)
			continue
		}

		resourceFieldRef := env.ValueFrom.ResourceFieldRef
		resourcePath := resourceFieldRef.Resource

		resourceFieldRef.ContainerName = firstContainerName
		if !hasResourceInContainer(resourcePath, firstContainer) {
			continue
		}

		filtered = append(filtered, env)
	}

	return filtered
}

// hasResourceInContainer 检查容器是否有指定的资源限制
func hasResourceInContainer(resourcePath string, container *corev1.Container) bool {
	if container.Resources.Limits == nil && container.Resources.Requests == nil {
		return false
	}

	switch resourcePath {
	case "limits.cpu":
		if container.Resources.Limits != nil {
			_, ok := container.Resources.Limits[corev1.ResourceCPU]
			return ok
		}
	case "limits.memory":
		if container.Resources.Limits != nil {
			_, ok := container.Resources.Limits[corev1.ResourceMemory]
			return ok
		}
	case "requests.cpu":
		if container.Resources.Requests != nil {
			_, ok := container.Resources.Requests[corev1.ResourceCPU]
			return ok
		}
	case "requests.memory":
		if container.Resources.Requests != nil {
			_, ok := container.Resources.Requests[corev1.ResourceMemory]
			return ok
		}
	}

	return false
}
