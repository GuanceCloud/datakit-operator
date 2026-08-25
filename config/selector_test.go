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

func TestInjectRulesSkipRuleWithBlankNamespaceSelector(t *testing.T) {
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

func TestInjectRulesDoNotDropInvalidConfiguredDimension(t *testing.T) {
	rules := LogfwdRules{
		&LogfwdRule{InjectRule: InjectRule{
			Name: "invalid",
			Selector: Selector{
				Namespaces: []string{"["},
				Labels:     []string{"app=web"},
			},
		}},
		&LogfwdRule{InjectRule: InjectRule{
			Name:     "valid",
			Selector: Selector{Namespaces: []string{"^production$"}},
		}},
	}
	rules.Setup()

	matched, rule := rules.Matches("production", map[string]string{"app": "web"})

	if assert.True(t, matched) {
		assert.Equal(t, "valid", rule.Name)
	}
}

func TestInjectRulesSkipRuleWithBlankLabelSelector(t *testing.T) {
	rules := LogfwdRules{
		&LogfwdRule{InjectRule: InjectRule{
			Name: "invalid",
			Selector: Selector{
				Namespaces: []string{"^production$"},
				Labels:     []string{""},
			},
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

func TestRulesWarnAboutInvalidOrMissingSelectors(t *testing.T) {
	var logs bytes.Buffer
	originalLog := log
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(&logs),
		zap.WarnLevel,
	)
	log = &logger.Logger{SugaredLogger: zap.New(core).Sugar()}
	defer func() {
		log = originalLog
	}()

	rules := LogfwdRules{
		&LogfwdRule{InjectRule: InjectRule{
			Name: "invalid",
			Selector: Selector{
				Namespaces: []string{"", "[", "   "},
				Labels:     []string{"", "app in (", "   "},
			},
		}},
		&LogfwdRule{InjectRule: InjectRule{Name: "missing"}},
	}
	rules.Setup()
	MutateRules{
		&MutateRule{Selector: Selector{Labels: []string{"app=web"}}},
	}.Setup()

	assert.Contains(t, logs.String(),
		`invalid admission selector: ruleset=logfwds rule_index=0 rule_name="invalid" selector=namespace selector_index=0 value="" reason=blank`,
	)
	assert.Contains(t, logs.String(), `selector=namespace selector_index=1 value="[" reason=compile_error`)
	assert.Contains(t, logs.String(), `selector=namespace selector_index=2 value="   " reason=blank`)
	assert.Contains(t, logs.String(), `selector=label selector_index=0 value="" reason=blank`)
	assert.Contains(t, logs.String(), `selector=label selector_index=1 value="app in (" reason=parse_error`)
	assert.Contains(t, logs.String(), `selector=label selector_index=2 value="   " reason=blank`)
	assert.Contains(t, logs.String(),
		`admission rule disabled: ruleset=logfwds rule_index=0 rule_name="invalid" reason=no_valid_namespace_selectors`,
	)
	assert.Contains(t, logs.String(),
		`admission rule disabled: ruleset=logfwds rule_index=0 rule_name="invalid" reason=no_valid_label_selectors`,
	)
	assert.Contains(t, logs.String(),
		`admission rule disabled: ruleset=logfwds rule_index=1 rule_name="missing" reason=no_selectors`,
	)
	assert.Contains(t, logs.String(),
		`admission rule disabled: ruleset=loggings rule_index=0 rule_name="" reason=no_namespace_selectors`,
	)
}

func TestAllInjectRuleTypesSkipInvalidRule(t *testing.T) {
	newRule := func(name, namespace string) InjectRule {
		return InjectRule{Name: name, Selector: Selector{Namespaces: []string{namespace}}}
	}

	t.Run("ddtrace", func(t *testing.T) {
		rules := DDTraceRules{
			&DDTraceRule{InjectRule: newRule("invalid", "")},
			&DDTraceRule{InjectRule: newRule("valid", "^production$")},
		}
		rules.Setup()
		matched, matchedRules := rules.MatchesAll("production", nil)
		if assert.True(t, matched) && assert.Len(t, matchedRules, 1) {
			assert.Equal(t, "valid", matchedRules[0].Name)
		}
	})

	t.Run("otel", func(t *testing.T) {
		rules := OTelRules{
			&OTelRule{InjectRule: newRule("invalid", "")},
			&OTelRule{InjectRule: newRule("valid", "^production$")},
		}
		rules.Setup()
		matched, matchedRules := rules.MatchesAll("production", nil)
		if assert.True(t, matched) && assert.Len(t, matchedRules, 1) {
			assert.Equal(t, "valid", matchedRules[0].Name)
		}
	})

	t.Run("logfwd", func(t *testing.T) {
		rules := LogfwdRules{
			&LogfwdRule{InjectRule: newRule("invalid", "")},
			&LogfwdRule{InjectRule: newRule("valid", "^production$")},
		}
		rules.Setup()
		matched, rule := rules.Matches("production", nil)
		if assert.True(t, matched) {
			assert.Equal(t, "valid", rule.Name)
		}
	})

	t.Run("flameshot", func(t *testing.T) {
		rules := FlameshotRules{
			&FlameshotRule{InjectRule: newRule("invalid", "")},
			&FlameshotRule{InjectRule: newRule("valid", "^production$")},
		}
		rules.Setup()
		matched, rule := rules.Matches("production", nil)
		if assert.True(t, matched) {
			assert.Equal(t, "valid", rule.Name)
		}
	})

	t.Run("profiler", func(t *testing.T) {
		rules := ProfilerRules{
			&ProfilerRule{InjectRule: newRule("invalid", "")},
			&ProfilerRule{InjectRule: newRule("valid", "^production$")},
		}
		rules.Setup()
		matched, rule := rules.Matches("production", nil)
		if assert.True(t, matched) {
			assert.Equal(t, "valid", rule.Name)
		}
	})
}

func TestValidInjectSelectorsKeepMatchingBehavior(t *testing.T) {
	tests := []struct {
		name      string
		selector  Selector
		namespace string
		labels    map[string]string
		matched   bool
	}{
		{
			name:      "namespace only",
			selector:  Selector{Namespaces: []string{"^production$"}},
			namespace: "production",
			matched:   true,
		},
		{
			name:      "label only",
			selector:  Selector{Labels: []string{"app=web"}},
			namespace: "any",
			labels:    map[string]string{"app": "web"},
			matched:   true,
		},
		{
			name:      "namespace and label",
			selector:  Selector{Namespaces: []string{"^production$"}, Labels: []string{"app=web"}},
			namespace: "production",
			labels:    map[string]string{"app": "web"},
			matched:   true,
		},
		{
			name:      "namespace and label require both",
			selector:  Selector{Namespaces: []string{"^production$"}, Labels: []string{"app=web"}},
			namespace: "staging",
			labels:    map[string]string{"app": "web"},
			matched:   false,
		},
		{
			name:      "namespace selectors are or",
			selector:  Selector{Namespaces: []string{"^staging$", "^production$"}},
			namespace: "production",
			matched:   true,
		},
		{
			name:      "asterisk matches all namespaces",
			selector:  Selector{Namespaces: []string{"*"}},
			namespace: "production",
			matched:   true,
		},
		{
			name:      "dot asterisk matches all namespaces",
			selector:  Selector{Namespaces: []string{".*"}},
			namespace: "production",
			matched:   true,
		},
		{
			name:      "label glob remains supported",
			selector:  Selector{Labels: []string{"app=web-*"}},
			namespace: "production",
			labels:    map[string]string{"app": "web-api"},
			matched:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rules := LogfwdRules{
				&LogfwdRule{InjectRule: InjectRule{Name: tc.name, Selector: tc.selector}},
			}
			rules.Setup()

			matched, _ := rules.Matches(tc.namespace, tc.labels)

			assert.Equal(t, tc.matched, matched)
		})
	}
}

func TestMixedSelectorsKeepValidEntries(t *testing.T) {
	rules := LogfwdRules{
		&LogfwdRule{InjectRule: InjectRule{
			Selector: Selector{Namespaces: []string{"", "[", "^production$"}},
		}},
	}
	rules.Setup()

	productionMatched, _ := rules.Matches("production", nil)
	stagingMatched, _ := rules.Matches("staging", nil)

	assert.True(t, productionMatched)
	assert.False(t, stagingMatched)
}

func TestLabelSelectorValidationKeepsRuleBoundaries(t *testing.T) {
	t.Run("invalid configured dimension disables rule", func(t *testing.T) {
		rules := LogfwdRules{
			&LogfwdRule{InjectRule: InjectRule{
				Name: "invalid",
				Selector: Selector{
					Namespaces: []string{"^production$"},
					Labels:     []string{"app in ("},
				},
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
	})

	t.Run("mixed dimension keeps valid entries", func(t *testing.T) {
		rules := LogfwdRules{
			&LogfwdRule{InjectRule: InjectRule{
				Selector: Selector{Labels: []string{"", "app=web"}},
			}},
		}
		rules.Setup()

		webMatched, _ := rules.Matches("production", map[string]string{"app": "web"})
		apiMatched, _ := rules.Matches("production", map[string]string{"app": "api"})

		assert.True(t, webMatched)
		assert.False(t, apiMatched)
	})
}

func TestMutateRulesSkipInvalidRule(t *testing.T) {
	rules := MutateRules{
		&MutateRule{
			Selector: Selector{Namespaces: []string{""}, Labels: []string{"app=web"}},
			Config:   "invalid",
		},
		&MutateRule{
			Selector: Selector{Namespaces: []string{"^production$"}, Labels: []string{"app=web"}},
			Config:   "valid",
		},
	}
	rules.Setup()

	matched, rule := rules.Matches("production", map[string]string{"app": "web"})

	if assert.True(t, matched) {
		assert.Equal(t, "valid", rule.Config)
	}
}
