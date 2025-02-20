package config

import (
	"github.com/ake-persson/mapslice-json"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/selector"
)

type AdmissionInjectConfig struct {
	DDTrace  ContainerConfig `json:"ddtrace"`
	Logfwd   ContainerConfig `json:"logfwd"`
	Profiler ContainerConfig `json:"profiler"`
}

func (c AdmissionInjectConfig) setup() {
	c.DDTrace.fillEnvs()
	c.DDTrace.fillLabelSelectors()
	c.Logfwd.fillEnvs()
	c.Profiler.fillEnvs()
}

type Envs []struct{ Key, Value string }

type NamespaceCondition struct{ Namespace, Language string }
type LabelSelectorCondition struct {
	LabelSelector string
	Language      string
	selector      selector.Selector
}

type ContainerConfig struct {
	EnabledNamespaces     []NamespaceCondition     `json:"enabled_namespaces,omitempty"`
	EnabledLabelSelectors []LabelSelectorCondition `json:"enabled_labelselectors,omitempty"`
	Images                map[string]string        `json:"images"`
	Environments          mapslice.MapSlice        `json:"envs"`
	envs                  Envs
}

func (c ContainerConfig) Image(name string) string { return c.Images[name] }
func (c ContainerConfig) Envs() Envs               { return c.envs }
func (c ContainerConfig) MatchNamespace(ns string) string {
	for _, s := range c.EnabledNamespaces {
		if s.Namespace == ns {
			return s.Language
		}
	}
	return ""
}
func (c ContainerConfig) MatchLabelSelector(labels map[string]string) string {
	for _, s := range c.EnabledLabelSelectors {
		if s.selector != nil && s.selector.Matches(labels) {
			return s.Language
		}
	}
	return ""
}

func newContainerConfig() ContainerConfig {
	return ContainerConfig{
		Images:       make(map[string]string),
		Environments: mapslice.MapSlice{},
	}
}

func (c *ContainerConfig) fillEnvs() {
	if len(c.Environments) == 0 {
		return
	}

	for _, item := range c.Environments {
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
		c.envs = append(c.envs, struct{ Key, Value string }{key, value})
	}
}

func (c *ContainerConfig) fillLabelSelectors() {
	if len(c.EnabledLabelSelectors) == 0 {
		return
	}

	for idx, item := range c.EnabledLabelSelectors {
		p, err := selector.ParseSelector(item.LabelSelector)
		if err != nil {
			log.Warnf("Unexpected labelSelector '%s', parse error: %s", item.LabelSelector, err)
			continue
		}
		c.EnabledLabelSelectors[idx].selector = p
	}
}
