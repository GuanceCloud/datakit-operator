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
		comparable
		commonRule() *InjectRule
	}
)

func (r *InjectRule) commonRule() *InjectRule {
	return r
}

func (rs DDTraceRules) Setup() {
	setupInjectRules("ddtraces", rs)
}

func (rs DDTraceRules) MatchesAll(ns string, labels map[string]string) (bool, []*DDTraceRule) {
	return matchAllInjectRules(rs, ns, labels)
}

func (rs OTelRules) Setup() {
	setupInjectRules("otels", rs)
}

func (rs OTelRules) MatchesAll(ns string, labels map[string]string) (bool, []*OTelRule) {
	return matchAllInjectRules(rs, ns, labels)
}

func (rs LogfwdRules) Setup() {
	setupInjectRules("logfwds", rs)
}

func (rs LogfwdRules) Matches(ns string, labels map[string]string) (bool, *LogfwdRule) {
	return matchInjectRule(rs, ns, labels)
}

func (rs FlameshotRules) Setup() {
	setupInjectRules("flameshots", rs)
}

func (rs FlameshotRules) Matches(ns string, labels map[string]string) (bool, *FlameshotRule) {
	return matchInjectRule(rs, ns, labels)
}

func (rs ProfilerRules) Setup() {
	setupInjectRules("profilers", rs)
}

func (rs ProfilerRules) Matches(ns string, labels map[string]string) (bool, *ProfilerRule) {
	return matchInjectRule(rs, ns, labels)
}

func setupInjectRules[T injectRule](ruleset string, rs []T) {
	for idx := range rs {
		if isNullInjectRule(rs[idx]) {
			log.Warnf("admission rule disabled: ruleset=%s rule_index=%d reason=null_rule", ruleset, idx)
			continue
		}
		rule := rs[idx].commonRule()
		if rule.Resources.Nil() {
			rule.Resources = defaultResourceRequirements()
		} else if err := rule.Resources.Verify(); err != nil {
			log.Warnf("invalid resource requirements: rule_index=%d, error=%v, using default", idx, err)
			rule.Resources = defaultResourceRequirements()
		}

		rule.setup(ruleset, idx, rule.Name)
		if len(rule.Namespaces) == 0 && len(rule.Labels) == 0 {
			log.Warnf("admission rule disabled: ruleset=%s rule_index=%d rule_name=%q reason=no_selectors",
				ruleset, idx, rule.Name)
		}
		rule.setupEnvs()
	}
}

func matchInjectRule[T injectRule](rs []T, ns string, labels map[string]string) (bool, T) {
	var zero T
	for idx := range rs {
		if isNullInjectRule(rs[idx]) {
			continue
		}
		rule := rs[idx].commonRule()
		if injectRuleMatches(rule, ns, labels) {
			return true, rs[idx]
		}
	}
	return false, zero
}

func matchAllInjectRules[T injectRule](rs []T, ns string, labels map[string]string) (bool, []T) {
	var rules []T
	for idx := range rs {
		if isNullInjectRule(rs[idx]) {
			continue
		}
		rule := rs[idx].commonRule()
		if injectRuleMatches(rule, ns, labels) {
			rules = append(rules, rs[idx])
		}
	}
	return len(rules) > 0, rules
}

func isNullInjectRule[T injectRule](rule T) bool {
	var zero T
	return rule == zero
}

func injectRuleMatches(rule *InjectRule, ns string, labels map[string]string) bool {
	if len(rule.Namespaces) == 0 && len(rule.Labels) == 0 {
		return false
	}
	if len(rule.Namespaces) > 0 && len(rule.namespaceSelectors) == 0 {
		return false
	}
	if len(rule.Labels) > 0 && len(rule.labelSelectors) == 0 {
		return false
	}

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
		if rs[idx] == nil {
			log.Warnf("admission rule disabled: ruleset=loggings rule_index=%d reason=null_rule", idx)
			continue
		}
		rs[idx].setup("loggings", idx, "")
		if len(rs[idx].Namespaces) == 0 {
			log.Warnf("admission rule disabled: ruleset=loggings rule_index=%d rule_name=%q reason=no_namespace_selectors", idx, "")
		}
		if len(rs[idx].Labels) == 0 {
			log.Warnf("admission rule disabled: ruleset=loggings rule_index=%d rule_name=%q reason=no_label_selectors", idx, "")
		}
	}
}

func (rs MutateRules) Matches(ns string, labels map[string]string) (bool, *MutateRule) {
	for idx := range rs {
		if rs[idx] == nil {
			continue
		}
		if rs[idx].matchNamespace(ns) && rs[idx].matchLabels(labels) {
			return true, rs[idx]
		}
	}
	return false, nil
}
