package config

import "gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/selector"

type AdmissionMutateConfig struct {
	Loggings LoggingConfigs `json:"loggings"`
}

func (c *AdmissionMutateConfig) setup() {
	for idx := range c.Loggings {
		for _, labelSelector := range c.Loggings[idx].Labels {
			p, err := selector.ParseSelector(labelSelector)
			if err != nil {
				log.Warnf("Unexpected labelSelector '%s', parse error: %s", labelSelector, err)
				continue
			}
			c.Loggings[idx].labelSelectors = append(c.Loggings[idx].labelSelectors, p)
		}
	}
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
	Namespaces     []string `json:"namespace_selectors"`
	Labels         []string `json:"label_selectors"`
	labelSelectors []selector.Selector
}

func (s Selector) matchNamespace(ns string) bool {
	for _, namespace := range s.Namespaces {
		if ns == namespace {
			return true
		}
	}
	return false
}

func (s Selector) matchLabels(labels map[string]string) bool {
	for _, se := range s.labelSelectors {
		if se.Matches(labels) {
			return true
		}
	}
	return false
}
