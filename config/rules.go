// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package config

import (
	"github.com/ake-persson/mapslice-json"
)

type (
	Envs []struct{ Key, Value string }

	InjectRule struct {
		Name string `json:"name"`
		Selector

		Image        string               `json:"image"`
		Environments mapslice.MapSlice    `json:"envs"`
		Resources    ResourceRequirements `json:"resources"`
		Envs         Envs                 `json:"-"`
	}

	DDTraceRules []*DDTraceRule
	DDTraceRule  struct {
		InjectRule
		CheckAnnotation bool   `json:"check_annotation"`
		Language        string `json:"language"`
		PHPLoaderFlavor string `json:"php_loader_flavor,omitempty"`
	}

	OTelRules []*OTelRule
	OTelRule  struct {
		InjectRule
		CheckAnnotation bool   `json:"check_annotation"`
		Language        string `json:"language"`
	}

	LogfwdRules []*LogfwdRule
	LogfwdRule  struct {
		InjectRule
		CheckAnnotation bool     `json:"check_annotation"`
		LogConfigs      string   `json:"log_configs"`
		LogVolumePaths  []string `json:"log_volume_paths"`
	}

	FlameshotRules []*FlameshotRule
	FlameshotRule  struct {
		InjectRule
		Processes                   string `json:"processes"`
		EnablePrometheusAnnotations bool   `json:"enable_prometheus_annotations,omitempty"`
	}

	ProfilerRules []*ProfilerRule
	ProfilerRule  struct {
		InjectRule
		CheckAnnotation bool              `json:"check_annotation"`
		Language        string            `json:"language"`
		Images          map[string]string `json:"-"`
	}

	injectRule interface {
		commonRule() *InjectRule
	}
)

func (r *InjectRule) commonRule() *InjectRule {
	return r
}

func (rs DDTraceRules) Setup() {
	setupInjectRules(rs)
}

func (rs DDTraceRules) MatchesAll(ns string, labels map[string]string) (bool, []*DDTraceRule) {
	return matchAllInjectRules(rs, ns, labels)
}

func (rs OTelRules) Setup() {
	setupInjectRules(rs)
}

func (rs OTelRules) MatchesAll(ns string, labels map[string]string) (bool, []*OTelRule) {
	return matchAllInjectRules(rs, ns, labels)
}

func (rs LogfwdRules) Setup() {
	setupInjectRules(rs)
}

func (rs LogfwdRules) Matches(ns string, labels map[string]string) (bool, *LogfwdRule) {
	return matchInjectRule(rs, ns, labels)
}

func (rs FlameshotRules) Setup() {
	setupInjectRules(rs)
}

func (rs FlameshotRules) Matches(ns string, labels map[string]string) (bool, *FlameshotRule) {
	return matchInjectRule(rs, ns, labels)
}

func (rs ProfilerRules) Setup() {
	setupInjectRules(rs)
}

func (rs ProfilerRules) Matches(ns string, labels map[string]string) (bool, *ProfilerRule) {
	return matchInjectRule(rs, ns, labels)
}

func setupInjectRules[T injectRule](rs []T) {
	for idx := range rs {
		rule := rs[idx].commonRule()
		if rule.Resources.Nil() {
			rule.Resources = defaultResourceRequirements()
		} else if err := rule.Resources.Verify(); err != nil {
			log.Warnf("invalid resource requirements: rule_index=%d, error=%v, using default", idx, err)
			rule.Resources = defaultResourceRequirements()
		}

		rule.Setup()
		rule.setupEnvs()
	}
}

func matchInjectRule[T injectRule](rs []T, ns string, labels map[string]string) (bool, T) {
	var zero T
	for idx := range rs {
		rule := rs[idx].commonRule()
		if len(rule.namespaceSelectors) == 0 && len(rule.labelSelectors) == 0 {
			return false, zero
		}

		if injectRuleMatches(rule, ns, labels) {
			return true, rs[idx]
		}
	}
	return false, zero
}

func matchAllInjectRules[T injectRule](rs []T, ns string, labels map[string]string) (bool, []T) {
	var rules []T
	for idx := range rs {
		rule := rs[idx].commonRule()
		if len(rule.namespaceSelectors) == 0 && len(rule.labelSelectors) == 0 {
			continue
		}

		if injectRuleMatches(rule, ns, labels) {
			rules = append(rules, rs[idx])
		}
	}
	return len(rules) > 0, rules
}

func injectRuleMatches(rule *InjectRule, ns string, labels map[string]string) bool {
	namespaceMatched := true
	labelMatched := true

	if len(rule.namespaceSelectors) != 0 {
		namespaceMatched = rule.matchNamespace(ns)
	}
	if len(rule.labelSelectors) != 0 {
		labelMatched = rule.matchLabels(labels)
	}

	return namespaceMatched && labelMatched
}

func (r *InjectRule) setupEnvs() {
	for _, item := range r.Environments {
		key, ok := item.Key.(string)
		if !ok {
			log.Warnf("Unexpected environment key: %#v", item.Key)
			continue
		}
		value, ok := item.Value.(string)
		if !ok {
			log.Warnf("Unexpected environment value: %#v", item.Value)
			continue
		}
		r.Envs = append(r.Envs, struct{ Key, Value string }{key, value})
	}
}

type (
	MutateRules []*MutateRule
	MutateRule  struct {
		Selector
		Config string `json:"config"`
	}
)

func (rs MutateRules) Setup() {
	for idx := range rs {
		rs[idx].Setup()
	}
}

func (rs MutateRules) Matches(ns string, labels map[string]string) (bool, *MutateRule) {
	for idx := range rs {
		if rs[idx].matchNamespace(ns) && rs[idx].matchLabels(labels) {
			return true, rs[idx]
		}
	}
	return false, nil
}
