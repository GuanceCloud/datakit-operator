// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package config

import (
	"testing"

	"github.com/ake-persson/mapslice-json"
	"github.com/stretchr/testify/assert"
)

func TestConvertDeprecatedToAdmissionInject(t *testing.T) {
	input := &DeprecatedInjectConfig{
		DDTrace: DeprecatedInjectRule{
			EnabledNamespaces: []struct {
				Namespace string
				Language  string
			}{
				{Namespace: "default", Language: "java"},
				{Namespace: "production", Language: "java"},
			},
			EnabledLabelSelectors: []struct {
				LabelSelector string
				Language      string
			}{
				{LabelSelector: "app=myapp", Language: "java"},
			},
			Images: map[string]string{
				"java_agent_image":   "pubrepo.guance.com/datakit-operator/dd-lib-java-init:v1.8.4",
				"python_agent_image": "pubrepo.guance.com/datakit-operator/dd-lib-python-init:v1.6.2",
			},
			Environments: mapslice.MapSlice{
				{Key: "DD_AGENT_HOST", Value: "datakit-service.datakit.svc"},
				{Key: "DD_TRACE_AGENT_PORT", Value: "9529"},
			},
			Resources: ResourceRequirements{
				Requests: ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
				Limits:   ResourceQuotaConfig{CPU: "500m", Memory: "512Mi"},
			},
		},
		Logfwd: DeprecatedInjectRule{
			Images: map[string]string{
				"logfwd_image": "pubrepo.guance.com/datakit/logfwd:1.5.8",
			},
			Environments: mapslice.MapSlice{
				{Key: "LOGFWD_ENV", Value: "test"},
			},
			Resources: ResourceRequirements{
				Requests: ResourceQuotaConfig{CPU: "50m", Memory: "32Mi"},
				Limits:   ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
			},
		},
	}

	expected := AdmissionInjectConfig{
		DDTraces: InjectRules{
			&InjectRule{
				Legacy: false,
				Selector: Selector{
					Namespaces: []string{"default", "production"},
					Labels:     []string{"app=myapp"},
				},
				Language: "java",
				Image:    "pubrepo.guance.com/datakit-operator/dd-lib-java-init:v1.8.4",
				Environments: mapslice.MapSlice{
					{Key: "DD_AGENT_HOST", Value: "datakit-service.datakit.svc"},
					{Key: "DD_TRACE_AGENT_PORT", Value: "9529"},
				},
				Resources: ResourceRequirements{
					Requests: ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   ResourceQuotaConfig{CPU: "500m", Memory: "512Mi"},
				},
			},
		},
		Logfwds: InjectRules{
			&InjectRule{
				Legacy: true,
				Selector: Selector{
					Namespaces: []string{".*"},
					Labels:     []string{},
				},
				Language: "",
				Image:    "pubrepo.guance.com/datakit/logfwd:1.5.8",
				Environments: mapslice.MapSlice{
					{Key: "LOGFWD_ENV", Value: "test"},
				},
				Resources: ResourceRequirements{
					Requests: ResourceQuotaConfig{CPU: "50m", Memory: "32Mi"},
					Limits:   ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			},
		},
		Flameshots: InjectRules{},
	}

	result := convertDeprecatedToAdmissionInject(input)
	assert.Equal(t, expected, result)
}
