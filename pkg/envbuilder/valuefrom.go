package envbuilder

import (
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

var (
	valueFromMap = map[string]func() *corev1.EnvVarSource{
		"{fieldRef:metadata.name}": func() *corev1.EnvVarSource {
			return &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
			}
		},
		"{fieldRef:metadata.namespace}": func() *corev1.EnvVarSource {
			return &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			}
		},
		"{fieldRef:metadata.uid}": func() *corev1.EnvVarSource {
			return &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.uid"},
			}
		},
		"{fieldRef:spec.serviceAccountName}": func() *corev1.EnvVarSource {
			return &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.serviceAccountName"},
			}
		},
		"{fieldRef:spec.nodeName}": func() *corev1.EnvVarSource {
			return &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
			}
		},
		"{fieldRef:status.hostIP}": func() *corev1.EnvVarSource {
			return &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.hostIP"},
			}
		},
		"{fieldRef:status.hostIPs}": func() *corev1.EnvVarSource {
			return &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.hostIPs"},
			}
		},
		"{fieldRef:status.podIP}": func() *corev1.EnvVarSource {
			return &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"},
			}
		},
		"{fieldRef:status.podIPs}": func() *corev1.EnvVarSource {
			return &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIPs"},
			}
		},
	}

	annotationsRegex = regexp.MustCompile(`{fieldRef:metadata.annotations\['.*'\]}`)
	labelsRegex      = regexp.MustCompile(`{fieldRef:metadata.labels\['.*'\]}`)
)

func newEnvVarSource(v string) *corev1.EnvVarSource {
	newEnvVarSource, ok := valueFromMap[v]
	if ok {
		return newEnvVarSource()
	}

	if annotationsRegex.MatchString(v) {
		key := strings.TrimPrefix(v, "{fieldRef:metadata.annotations['")
		key = strings.TrimSuffix(key, "']}")

		if IsQualifiedName(key) {
			return &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: fmt.Sprintf("metadata.annotations['%s']", key)},
			}
		}
	}

	if labelsRegex.MatchString(v) {
		key := strings.TrimPrefix(v, "{fieldRef:metadata.labels['")
		key = strings.TrimSuffix(key, "']}")

		if IsQualifiedName(key) {
			return &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: fmt.Sprintf("metadata.labels['%s']", key)},
			}
		}
	}

	return nil
}
