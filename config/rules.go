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

	InjectRules []*InjectRule
	InjectRule  struct {
		Selector
		Language     string               `json:"language"`
		Images       string               `json:"images"`
		Environments mapslice.MapSlice    `json:"envs"`
		Resources    ResourceRequirements `json:"resources"`
		Envs         Envs                 `json:"-"`
	}
)

func (rs InjectRules) Setup() error {
	for idx := range rs {
		if rs[idx].Resources.Nil() {
			rs[idx].Resources = defaultResourceRequirements()
		} else if err := rs[idx].Resources.Verify(); err != nil {
			log.Warnf("invalid resource requirements: %v", err)
			rs[idx].Resources = defaultResourceRequirements()
		}

		rs[idx].Selector.Setup()
		rs[idx].setupEnvs()
	}
	return nil
}

func (rs InjectRules) Matches(ns string, labels map[string]string) (bool, *InjectRule) {
	for idx := range rs {
		if matched := rs[idx].Selector.matchNamespace(ns); matched {
			return true, rs[idx]
		}
		if matched := rs[idx].Selector.matchLabels(labels); matched {
			return true, rs[idx]
		}
	}
	return false, nil
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

func (rs MutateRules) Setup() error {
	for idx := range rs {
		rs[idx].Selector.Setup()
	}
	return nil
}

func (rs MutateRules) Matches(ns string, labels map[string]string) (bool, *MutateRule) {
	for idx := range rs {
		if rs[idx].Selector.matchNamespace(ns) && rs[idx].Selector.matchLabels(labels) {
			return true, rs[idx]
		}
	}
	return false, nil
}
