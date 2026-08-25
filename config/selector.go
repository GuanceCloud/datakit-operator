// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package config

import (
	"regexp"
	"strings"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/labels"
)

type Selector struct {
	Namespaces         []string `json:"namespace_selectors"`
	Labels             []string `json:"label_selectors"`
	namespaceSelectors []*regexp.Regexp
	labelSelectors     []labels.Selector
}

func (s *Selector) Setup() {
	s.setup("selector", 0, "")
}

func (s *Selector) setup(ruleset string, ruleIndex int, ruleName string) {
	s.namespaceSelectors = nil
	s.labelSelectors = nil

	for selectorIndex, namespaceSelector := range s.Namespaces {
		if strings.TrimSpace(namespaceSelector) == "" {
			log.Warnf("invalid admission selector: ruleset=%s rule_index=%d rule_name=%q selector=namespace selector_index=%d value=%q reason=blank",
				ruleset, ruleIndex, ruleName, selectorIndex, namespaceSelector)
			continue
		}
		ns := replaceAsteriskWithDotAsterisk(namespaceSelector)
		re, err := regexp.Compile(ns)
		if err != nil {
			log.Warnf("invalid admission selector: ruleset=%s rule_index=%d rule_name=%q selector=namespace selector_index=%d value=%q reason=compile_error error=%q",
				ruleset, ruleIndex, ruleName, selectorIndex, namespaceSelector, err)
			continue
		}
		s.namespaceSelectors = append(s.namespaceSelectors, re)
	}

	for selectorIndex, labelSelector := range s.Labels {
		p, err := labels.Parse(labelSelector)
		if err != nil {
			log.Warnf("invalid admission selector: ruleset=%s rule_index=%d rule_name=%q selector=label selector_index=%d value=%q reason=parse_error error=%q",
				ruleset, ruleIndex, ruleName, selectorIndex, labelSelector, err)
			continue
		}
		if p.Empty() {
			reason := "empty"
			if strings.TrimSpace(labelSelector) == "" {
				reason = "blank"
			}
			log.Warnf("invalid admission selector: ruleset=%s rule_index=%d rule_name=%q selector=label selector_index=%d value=%q reason=%s",
				ruleset, ruleIndex, ruleName, selectorIndex, labelSelector, reason)
			continue
		}
		s.labelSelectors = append(s.labelSelectors, p)
	}

	if len(s.Namespaces) > 0 && len(s.namespaceSelectors) == 0 {
		log.Warnf("admission rule disabled: ruleset=%s rule_index=%d rule_name=%q reason=no_valid_namespace_selectors",
			ruleset, ruleIndex, ruleName)
	}
	if len(s.Labels) > 0 && len(s.labelSelectors) == 0 {
		log.Warnf("admission rule disabled: ruleset=%s rule_index=%d rule_name=%q reason=no_valid_label_selectors",
			ruleset, ruleIndex, ruleName)
	}
}

func (s *Selector) matchNamespace(ns string) bool {
	for _, re := range s.namespaceSelectors {
		if re.MatchString(ns) {
			return true
		}
	}
	return false
}

func (s *Selector) matchLabels(m map[string]string) bool {
	for _, se := range s.labelSelectors {
		if se.Matches(labels.Set(m)) {
			return true
		}
	}
	return false
}

// replaceAsteriskWithDotAsterisk is syntactic sugar that simplifies the syntax.
func replaceAsteriskWithDotAsterisk(s string) string {
	if s == "*" {
		return ".*"
	}
	return s
}
