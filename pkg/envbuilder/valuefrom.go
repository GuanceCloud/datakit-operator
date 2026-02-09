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

type refType string

const (
	refTypeFieldRef         refType = "fieldRef"
	refTypeResourceFieldRef refType = "resourceFieldRef"
)

var valuefroms = []struct {
	key   string
	keyRe *regexp.Regexp
	fn    func(args string) (string, refType)
}{
	// fieldRef
	{
		key: "{fieldRef:metadata.name}",
		fn:  func(_ string) (string, refType) { return "metadata.name", refTypeFieldRef },
	},
	{
		key: "{fieldRef:metadata.namespace}",
		fn:  func(_ string) (string, refType) { return "metadata.namespace", refTypeFieldRef },
	},
	{
		key: "{fieldRef:metadata.uid}",
		fn:  func(_ string) (string, refType) { return "metadata.uid", refTypeFieldRef },
	},
	{
		key: "{fieldRef:spec.serviceAccountName}",
		fn:  func(_ string) (string, refType) { return "spec.serviceAccountName", refTypeFieldRef },
	},
	{
		key: "{fieldRef:spec.nodeName}",
		fn:  func(_ string) (string, refType) { return "spec.nodeName", refTypeFieldRef },
	},
	{
		key: "{fieldRef:status.hostIP}",
		fn:  func(_ string) (string, refType) { return "status.hostIP", refTypeFieldRef },
	},
	{
		key: "{fieldRef:status.hostIPs}",
		fn:  func(_ string) (string, refType) { return "status.hostIPs", refTypeFieldRef },
	},
	{
		key: "{fieldRef:status.podIP}",
		fn:  func(_ string) (string, refType) { return "status.podIP", refTypeFieldRef },
	},
	{
		// e.g. {fieldRef:metadata.labels['app']}
		keyRe: regexp.MustCompile(`{fieldRef:metadata.labels\['(.+)'\]}`),
		fn: func(s string) (string, refType) {
			return fmt.Sprintf("metadata.labels['%s']", s), refTypeFieldRef
		},
	},
	{
		// e.g. {fieldRef:metadata.annotations['app']}
		keyRe: regexp.MustCompile(`{fieldRef:metadata.annotations\['(.+)'\]}`),
		fn: func(s string) (string, refType) {
			return fmt.Sprintf("metadata.annotations['%s']", s), refTypeFieldRef
		},
	},

	// resourceFieldRef
	{
		key: "{resourceFieldRef:limits.cpu}",
		fn:  func(_ string) (string, refType) { return "limits.cpu", refTypeResourceFieldRef },
	},
	{
		key: "{resourceFieldRef:limits.memory}",
		fn:  func(_ string) (string, refType) { return "limits.memory", refTypeResourceFieldRef },
	},
	{
		key: "{resourceFieldRef:requests.cpu}",
		fn:  func(_ string) (string, refType) { return "requests.cpu", refTypeResourceFieldRef },
	},
	{
		key: "{resourceFieldRef:requests.memory}",
		fn:  func(_ string) (string, refType) { return "requests.memory", refTypeResourceFieldRef },
	},
}

func converFieldPath(s string) (string, refType) {
	for _, v := range valuefroms {
		if v.key == s {
			return v.fn("")
		}

		if v.keyRe != nil && v.keyRe.MatchString(s) {
			args := v.keyRe.FindStringSubmatch(s)
			if len(args) != 2 {
				break
			}
			if !IsQualifiedName(args[1]) {
				break
			}
			return v.fn(args[1])
		}
	}
	return "", ""
}

func newEnvVarSource(v string) *corev1.EnvVarSource {
	fieldPath, rt := converFieldPath(v)
	if fieldPath != "" {
		switch rt {
		case refTypeFieldRef:
			return &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: fieldPath},
			}
		case refTypeResourceFieldRef:
			var divisor resource.Quantity
			if strings.Contains(fieldPath, "cpu") {
				divisor = resource.MustParse("1m")
			} else if strings.Contains(fieldPath, "memory") {
				divisor = resource.MustParse("1Mi")
			}
			return &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{ContainerName: "", Resource: fieldPath, Divisor: divisor},
			}
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
