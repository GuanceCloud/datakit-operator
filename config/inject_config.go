// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package config

import (
	"regexp"

	"github.com/ake-persson/mapslice-json"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/labels"
)

type AdmissionInjectConfig struct {
	DDTrace  ContainerConfig `json:"ddtrace"`
	Logfwd   ContainerConfig `json:"logfwd"`
	Profiler ContainerConfig `json:"profiler"`
}

func (c *AdmissionInjectConfig) Setup() error {
	if err := c.DDTrace.Setup(); err != nil {
		return err
	}
	if err := c.Logfwd.Setup(); err != nil {
		return err
	}
	if err := c.Profiler.Setup(); err != nil {
		return err
	}
	return nil
}

type Envs []struct{ Key, Value string }

type NamespaceCondition struct {
	Namespace string
	Language  string
	re        *regexp.Regexp
}

type LabelSelectorCondition struct {
	LabelSelector string
	Language      string
	selector      labels.Selector
}

type ContainerConfig struct {
	EnabledNamespaces     []*NamespaceCondition     `json:"enabled_namespaces,omitempty"`
	EnabledLabelSelectors []*LabelSelectorCondition `json:"enabled_labelselectors,omitempty"`
	Images                map[string]string         `json:"images"`
	Environments          mapslice.MapSlice         `json:"envs"`
	Resources             *ResourceRequirements     `json:"resources"`
	envs                  Envs
}

func (c *ContainerConfig) Setup() error {
	if c.Resources == nil {
		c.Resources = defaultResourceRequirements()
	}
	c.setupEnvs()
	c.setupNamespacesAndLabelsSelectors()
	return c.Resources.Verify()
}

func (c ContainerConfig) Image(name string) string { return c.Images[name] }
func (c ContainerConfig) Envs() Envs               { return c.envs }

func (c ContainerConfig) GetLanguageFromNamespace(ns string) string {
	for _, s := range c.EnabledNamespaces {
		if s.re != nil && s.re.MatchString(ns) {
			return s.Language
		}
	}
	return ""
}

func (c ContainerConfig) GetLanguageFromLabels(m map[string]string) string {
	for _, s := range c.EnabledLabelSelectors {
		if s.selector != nil && s.selector.Matches(labels.Set(m)) {
			return s.Language
		}
	}
	return ""
}

func (c ContainerConfig) ResourceRequests() (cpu string, memory string) {
	if c.Resources != nil {
		return c.Resources.Requests.CPU, c.Resources.Requests.Memory
	}
	return "", ""
}

func (c ContainerConfig) ResourceLimits() (cpu string, memory string) {
	if c.Resources != nil {
		return c.Resources.Limits.CPU, c.Resources.Limits.Memory
	}
	return "", ""
}

func newContainerConfig() ContainerConfig {
	return ContainerConfig{
		Images:       make(map[string]string),
		Environments: mapslice.MapSlice{},
	}
}

func (c *ContainerConfig) setupEnvs() {
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

func (c *ContainerConfig) setupNamespacesAndLabelsSelectors() {
	if len(c.EnabledNamespaces) == 0 && len(c.EnabledLabelSelectors) == 0 {
		return
	}

	for idx := range c.EnabledNamespaces {
		ns := replaceAsteriskWithDotAsterisk(c.EnabledNamespaces[idx].Namespace)
		re, err := regexp.Compile(ns)
		if err != nil {
			log.Warnf("Unexpected namespaceSelector '%s', compile error: %s", ns, err)
			continue
		}
		c.EnabledNamespaces[idx].re = re
	}

	for idx := range c.EnabledLabelSelectors {
		se := c.EnabledLabelSelectors[idx].LabelSelector
		p, err := labels.Parse(se)
		if err != nil {
			log.Warnf("Unexpected labelSelector '%s', parse error: %s", se, err)
			continue
		}
		c.EnabledLabelSelectors[idx].selector = p
	}
}

// replaceAsteriskWithDotAsterisk is syntactic sugar that simplifies the syntax.
func replaceAsteriskWithDotAsterisk(s string) string {
	if s == "*" {
		return ".*"
	}
	return s
}
