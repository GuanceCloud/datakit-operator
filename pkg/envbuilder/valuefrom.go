package envbuilder

import (
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
)

var valuefroms = []struct {
	key   string
	keyRe *regexp.Regexp
	fn    func(args string) string
}{
	{
		key: "{fieldRef:metadata.name}",
		fn:  func(_ string) string { return "metadata.name" },
	},
	{
		key: "{fieldRef:metadata.namespace}",
		fn:  func(_ string) string { return "metadata.namespace" },
	},
	{
		key: "{fieldRef:metadata.uid}",
		fn:  func(_ string) string { return "metadata.uid" },
	},
	{
		key: "{fieldRef:spec.serviceAccountName}",
		fn:  func(_ string) string { return "spec.serviceAccountName" },
	},
	{
		key: "{fieldRef:spec.nodeName}",
		fn:  func(_ string) string { return "spec.nodeName" },
	},
	{
		key: "{fieldRef:status.hostIP}",
		fn:  func(_ string) string { return "status.hostIP" },
	},
	{
		key: "{fieldRef:status.hostIPs}",
		fn:  func(_ string) string { return "status.hostIPs" },
	},
	{
		key: "{fieldRef:status.podIP}",
		fn:  func(_ string) string { return "status.podIP" },
	},
	{
		// e.g. {fieldRef:metadata.labels['app']}
		keyRe: regexp.MustCompile(`{fieldRef:metadata.labels\['(.+)'\]}`),
		fn: func(s string) string {
			return fmt.Sprintf("metadata.labels['%s']", s)
		},
	},
	{
		// e.g. {fieldRef:metadata.annotations['app']}
		keyRe: regexp.MustCompile(`{fieldRef:metadata.annotations\['(.+)'\]}`),
		fn: func(s string) string {
			return fmt.Sprintf("metadata.annotations['%s']", s)
		},
	},
}

func converFieldPath(s string) string {
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
	return ""
}

func newEnvVarSource(v string) *corev1.EnvVarSource {
	fieldPath := converFieldPath(v)
	if fieldPath != "" {
		return &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{FieldPath: fieldPath},
		}
	}
	return nil
}
