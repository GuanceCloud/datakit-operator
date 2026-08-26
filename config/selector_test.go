// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package config

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInjectSelectorValidation(t *testing.T) {
	tests := []struct {
		name      string
		selector  Selector
		namespace string
		labels    map[string]string
		want      bool
	}{
		{
			name:      "invalid configured dimension fails closed",
			selector:  Selector{Namespaces: []string{"["}, Labels: []string{"app=web"}},
			namespace: "production",
			labels:    map[string]string{"app": "web"},
		},
		{
			name:      "valid namespace survives invalid entries",
			selector:  Selector{Namespaces: []string{"", "[", "^production$"}},
			namespace: "production",
			want:      true,
		},
		{
			name:     "valid label survives invalid entries",
			selector: Selector{Labels: []string{"", "app in (", "app=web"}},
			labels:   map[string]string{"app": "web"},
			want:     true,
		},
		{
			name:      "namespace and label are both required",
			selector:  Selector{Namespaces: []string{"^production$"}, Labels: []string{"app=web"}},
			namespace: "staging",
			labels:    map[string]string{"app": "web"},
		},
		{
			name:      "wildcard namespace and glob label remain compatible",
			selector:  Selector{Namespaces: []string{"*"}, Labels: []string{"app=web-*"}},
			namespace: "production",
			labels:    map[string]string{"app": "web-api"},
			want:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rules := LogfwdRules{&LogfwdRule{InjectRule: InjectRule{Selector: tc.selector}}}
			rules.Setup()
			matched, _ := rules.Matches(tc.namespace, tc.labels)
			assert.Equal(t, tc.want, matched)
		})
	}
}

func TestSelectorsWithinOneDimensionUseOR(t *testing.T) {
	tests := []struct {
		name      string
		selector  Selector
		namespace string
		labels    map[string]string
	}{
		{
			name:      "namespace",
			selector:  Selector{Namespaces: []string{"^staging$", "^production$"}},
			namespace: "production",
		},
		{
			name:     "label",
			selector: Selector{Labels: []string{"app=api", "app=web"}},
			labels:   map[string]string{"app": "web"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rules := LogfwdRules{&LogfwdRule{InjectRule: InjectRule{Selector: tc.selector}}}
			rules.Setup()
			matched, _ := rules.Matches(tc.namespace, tc.labels)
			assert.True(t, matched)
		})
	}
}

func TestInvalidInjectRuleDoesNotStopLaterRules(t *testing.T) {
	rules := LogfwdRules{
		&LogfwdRule{InjectRule: InjectRule{
			Name:     "invalid",
			Selector: Selector{Namespaces: []string{""}},
		}},
		&LogfwdRule{InjectRule: InjectRule{
			Name:     "valid",
			Selector: Selector{Namespaces: []string{"^production$"}},
		}},
	}
	rules.Setup()

	matched, rule := rules.Matches("production", nil)
	if assert.True(t, matched) {
		assert.Equal(t, "valid", rule.Name)
	}
}

func TestMatchAllSkipsInvalidRules(t *testing.T) {
	rules := OTelRules{
		&OTelRule{InjectRule: InjectRule{Selector: Selector{Labels: []string{""}}}},
		&OTelRule{InjectRule: InjectRule{
			Name:     "valid",
			Selector: Selector{Labels: []string{"app=web"}},
		}},
	}
	rules.Setup()

	matched, selected := rules.MatchesAll("production", map[string]string{"app": "web"})
	if assert.True(t, matched) && assert.Len(t, selected, 1) {
		assert.Equal(t, "valid", selected[0].Name)
	}
}

func TestMutateRulesRequireBothSelectorDimensions(t *testing.T) {
	rules := MutateRules{
		&MutateRule{Selector: Selector{Namespaces: []string{""}, Labels: []string{"app=web"}}, Config: "invalid"},
		&MutateRule{Selector: Selector{Namespaces: []string{"^production$"}, Labels: []string{"app=web"}}, Config: "valid"},
	}
	rules.Setup()

	matched, rule := rules.Matches("production", map[string]string{"app": "web"})
	if assert.True(t, matched) {
		assert.Equal(t, "valid", rule.Config)
	}
}

func TestInvalidSelectorsProduceActionableWarnings(t *testing.T) {
	var logs bytes.Buffer
	originalLog := log
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(&logs),
		zap.WarnLevel,
	)
	log = &logger.Logger{SugaredLogger: zap.New(core).Sugar()}
	t.Cleanup(func() { log = originalLog })

	LogfwdRules{
		&LogfwdRule{InjectRule: InjectRule{
			Name: "broken",
			Selector: Selector{
				Namespaces: []string{"", "["},
				Labels:     []string{"", "app in ("},
			},
		}},
		&LogfwdRule{InjectRule: InjectRule{Name: "missing"}},
	}.Setup()

	for _, fragment := range []string{
		`ruleset=logfwds rule_index=0 rule_name="broken" selector=namespace`,
		`selector=label`,
		`reason=no_valid_namespace_selectors`,
		`reason=no_valid_label_selectors`,
		`rule_name="missing" reason=no_selectors`,
	} {
		assert.Contains(t, logs.String(), fragment)
	}
}
