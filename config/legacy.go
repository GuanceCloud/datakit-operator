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

	DeprecatedProfilerJavaImageKey   = "java_profiler_image"
	DeprecatedProfilerPythonImageKey = "python_profiler_image"
	DeprecatedProfilerGolangImageKey = "golang_profiler_image"
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

	Images       map[string]string    `json:"images"`
	Environments mapslice.MapSlice    `json:"envs"`
	Resources    ResourceRequirements `json:"resources,omitempty"`
}

func convertDeprecatedToAdmissionInject(cfg *DeprecatedInjectConfig) AdmissionInjectConfig {
	if cfg == nil {
		return AdmissionInjectConfig{}
	}

	result := AdmissionInjectConfig{
		DDTraces:   convertDeprecatedToDDTraceRules(&cfg.DDTrace),
		Logfwds:    convertDeprecatedToLogfwdRules(&cfg.Logfwd),
		Profilers:  convertDeprecatedToProfilerRules(&cfg.Profiler),
		Flameshots: FlameshotRules{},
	}

	return result
}

func convertDeprecatedToDDTraceRules(deprecated *DeprecatedInjectRule) DDTraceRules {
	rule, ok := convertDeprecatedInjectRule(deprecated, DeprecatedDDTraceJavaImageKey)
	if !ok {
		return DDTraceRules{}
	}

	return DDTraceRules{&DDTraceRule{
		InjectRule:      rule,
		CheckAnnotation: true,
		Language:        "java",
	}}
}

func convertDeprecatedToLogfwdRules(deprecated *DeprecatedInjectRule) LogfwdRules {
	rule, ok := convertDeprecatedInjectRule(deprecated, DeprecatedLogfwdImageKey)
	if !ok {
		return LogfwdRules{}
	}

	return LogfwdRules{&LogfwdRule{
		InjectRule:      rule,
		CheckAnnotation: true,
	}}
}

func convertDeprecatedToProfilerRules(deprecated *DeprecatedInjectRule) ProfilerRules {
	rule, ok := convertDeprecatedInjectRule(deprecated, "")
	if !ok {
		return ProfilerRules{}
	}

	return ProfilerRules{&ProfilerRule{
		InjectRule:      rule,
		CheckAnnotation: true,
		Images:          deprecated.Images,
	}}
}

func convertDeprecatedInjectRule(deprecated *DeprecatedInjectRule, imageKey string) (InjectRule, bool) {
	if deprecated == nil {
		return InjectRule{}, false
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

	rule := InjectRule{
		Name: "Used InjectV1 Config",
		Selector: Selector{
			Namespaces: namespaces,
			Labels:     labels,
		},
		Image:        image,
		Environments: deprecated.Environments,
		Resources:    deprecated.Resources,
	}

	// 如果没有 namespace 和 label selector，但存在其他配置（image, envs, resources）
	// InjectRule 的 Namespaces 为 [".*"] 以匹配所有 namespace
	if len(namespaces) == 0 && len(labels) == 0 && (image != "" || deprecated.Environments != nil) {
		rule.Selector = Selector{
			Namespaces: []string{".*"},
			Labels:     []string{},
		}
		return rule, true
	}

	// 如果没有任何配置，返回空数组
	if len(namespaces) == 0 && len(labels) == 0 {
		return InjectRule{}, false
	}

	return rule, true
}
