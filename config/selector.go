// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package config

import (
	"regexp"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/labels"
)

type Selector struct {
	Namespaces         []string `json:"namespace_selectors"`
	Labels             []string `json:"label_selectors"`
	namespaceSelectors []*regexp.Regexp
	labelSelectors     []labels.Selector
}

func (s *Selector) Setup() {
	for _, namespaceSelector := range s.Namespaces {
		ns := replaceAsteriskWithDotAsterisk(namespaceSelector)
		re, err := regexp.Compile(ns)
		if err != nil {
			log.Warnf("Unexpected namespaceSelector '%s', compile error: %s", ns, err)
			continue
		}
		s.namespaceSelectors = append(s.namespaceSelectors, re)
	}

	for _, labelSelector := range s.Labels {
		p, err := labels.Parse(labelSelector)
		if err != nil {
			log.Warnf("Unexpected labelSelector '%s', parse error: %s", labelSelector, err)
			continue
		}
		s.labelSelectors = append(s.labelSelectors, p)
	}
}

func (s *Selector) matchNamespace(ns string) bool {
	if len(s.namespaceSelectors == 0) {
		return true
	}
	for _, re := range s.namespaceSelectors {
		if re.MatchString(ns) {
			return true
		}
	}
	return false
}

func (s *Selector) matchLabels(m map[string]string) bool {
	if len(s.labelSelectors == 0) {
		return true
	}
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
