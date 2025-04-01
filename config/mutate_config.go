package config

import (
	"regexp"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/labels"
)

type AdmissionMutateConfig struct {
	Loggings LoggingConfigs `json:"loggings"`
}

func (c *AdmissionMutateConfig) Setup() error {
	for idx := range c.Loggings {
		c.Loggings[idx].Selector.Setup()
	}
	return nil
}

type LoggingConfigs []LoggingConfig

type LoggingConfig struct {
	Selector
	Config string `json:"config"`
}

func (cfgs LoggingConfigs) MatchNamespace(ns string) string {
	for _, cfg := range cfgs {
		if matched := cfg.Selector.matchNamespace(ns); matched {
			return cfg.Config
		}
	}
	return ""
}

func (cfgs LoggingConfigs) MatchLabels(labels map[string]string) string {
	for _, cfg := range cfgs {
		if matched := cfg.Selector.matchLabels(labels); matched {
			return cfg.Config
		}
	}
	return ""
}

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
