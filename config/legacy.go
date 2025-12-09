// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package config

import "github.com/ake-persson/mapslice-json"

const (
	// DeprecatedDDTraceJavaImageKey 用于从旧配置的 Images map 中提取 DDTrace Java agent image
	DeprecatedDDTraceJavaImageKey = "java_agent_image"
	// DeprecatedLogfwdImageKey 用于从旧配置的 Images map 中提取 Logfwd image
	DeprecatedLogfwdImageKey = "logfwd_image"
)

type DeprecatedInjectConfig struct {
	DDTrace  DeprecatedInjectRule `json:"ddtrace"`
	Logfwd   DeprecatedInjectRule `json:"logfwd"`
	Profiler DeprecatedInjectRule `json:"profiler"`
}

type DeprecatedInjectRule struct {
	EnabledNamespaces []struct {
		Namespace string
		Language  string
	} `json:"enabled_namespaces,omitempty"`

	EnabledLabelSelectors []struct {
		LabelSelector string
		Language      string
	} `json:"enabled_labelselectors,omitempty"`

	Images       map[string]string     `json:"images"`
	Environments mapslice.MapSlice     `json:"envs"`
	Resources    *ResourceRequirements `json:"resources,omitempty"`
}

func convertDeprecatedToAdmissionInject(cfg *DeprecatedInjectConfig) AdmissionInjectConfig {
	if cfg == nil {
		return AdmissionInjectConfig{}
	}

	result := AdmissionInjectConfig{
		DDTrace:   convertDeprecatedToInjectRules(&cfg.DDTrace, "java", DeprecatedDDTraceJavaImageKey),
		Logfwd:    convertDeprecatedToInjectRules(&cfg.Logfwd, "", DeprecatedLogfwdImageKey),
		Flameshot: InjectRules{},
	}

	return result
}

func convertDeprecatedToInjectRules(deprecated *DeprecatedInjectRule, language, imageKey string) InjectRules {
	if deprecated == nil {
		return InjectRules{}
	}

	namespaces := make([]string, 0)
	labels := make([]string, 0)

	for _, ns := range deprecated.EnabledNamespaces {
		if ns.Namespace != "" {
			namespaces = append(namespaces, ns.Namespace)
		}
	}
	for _, ls := range deprecated.EnabledLabelSelectors {
		if ls.LabelSelector != "" {
			labels = append(labels, ls.LabelSelector)
		}
	}

	// 从 Images map 中提取指定 key 的 image
	var image string
	if deprecated.Images != nil {
		image = deprecated.Images[imageKey]
	}

	// 如果没有 namespace 和 label selector，但存在其他配置（image, envs, resources），
	// 仍然创建一个 InjectRule 以保持兼容性
	if len(namespaces) == 0 && len(labels) == 0 &&
		(image != "" || deprecated.Environments != nil || deprecated.Resources != nil) {
		// 创建一个空的 InjectRule，只包含 image, envs, resources
		rule := &InjectRule{
			Selector: Selector{
				Namespaces: []string{},
				Labels:     []string{},
			},
			Language:     language,
			Images:       image,
			Environments: deprecated.Environments,
			Resources:    deprecated.Resources,
		}
		return InjectRules{rule}
	}

	// 如果没有任何配置，返回空数组
	if len(namespaces) == 0 && len(labels) == 0 {
		return InjectRules{}
	}

	// 创建一个 InjectRule，包含所有的 namespace 和 label selector
	rule := &InjectRule{
		Selector: Selector{
			Namespaces: namespaces,
			Labels:     labels,
		},
		Language:     language,
		Images:       image,
		Environments: deprecated.Environments,
		Resources:    deprecated.Resources,
	}

	return InjectRules{rule}
}
