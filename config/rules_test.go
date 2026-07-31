// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdmissionInjectConfigJSONCompatibility(t *testing.T) {
	raw := `{
		"ddtraces": [{
			"name": "ddtrace",
			"namespace_selectors": ["default"],
			"label_selectors": ["app=web"],
			"image": "ddtrace:latest",
			"envs": {"DD_ENV": "test"},
			"resources": {
				"requests": {"cpu": "100m", "memory": "64Mi"},
				"limits": {"cpu": "500m", "memory": "512Mi"}
			},
			"check_annotation": true,
			"language": "php",
			"php_loader_flavor": "linux-musl"
		}],
		"logfwds": [{
			"name": "logfwd",
			"namespace_selectors": ["default"],
			"image": "logfwd:latest",
			"check_annotation": true,
			"log_configs": "[{\"path\":\"/var/log/app.log\"}]",
			"log_volume_paths": ["/var/log"]
		}],
		"flameshots": [{
			"name": "flameshot",
			"namespace_selectors": ["default"],
			"image": "flameshot:latest",
			"processes": "app",
			"enable_prometheus_annotations": true
		}],
		"profilers": [{
			"name": "profiler",
			"namespace_selectors": ["default"],
			"image": "profiler:latest",
			"check_annotation": true,
			"language": "java"
		}]
	}`

	var cfg AdmissionInjectConfig
	if !assert.NoError(t, json.Unmarshal([]byte(raw), &cfg)) {
		return
	}

	if !assert.Len(t, cfg.DDTraces, 1) {
		return
	}
	assert.Equal(t, "ddtrace", cfg.DDTraces[0].Name)
	assert.Equal(t, []string{"default"}, cfg.DDTraces[0].Namespaces)
	assert.Equal(t, []string{"app=web"}, cfg.DDTraces[0].Labels)
	assert.Equal(t, "ddtrace:latest", cfg.DDTraces[0].Image)
	assert.Equal(t, "DD_ENV", cfg.DDTraces[0].Environments[0].Key)
	assert.Equal(t, "100m", cfg.DDTraces[0].Resources.Requests.CPU)
	assert.True(t, cfg.DDTraces[0].CheckAnnotation)
	assert.Equal(t, "php", cfg.DDTraces[0].Language)
	assert.Equal(t, "linux-musl", cfg.DDTraces[0].PHPLoaderFlavor)

	if !assert.Len(t, cfg.Logfwds, 1) {
		return
	}
	assert.True(t, cfg.Logfwds[0].CheckAnnotation)
	assert.JSONEq(t, `[{"path":"/var/log/app.log"}]`, cfg.Logfwds[0].LogConfigs)
	assert.Equal(t, []string{"/var/log"}, cfg.Logfwds[0].LogVolumePaths)

	if !assert.Len(t, cfg.Flameshots, 1) {
		return
	}
	assert.Equal(t, "app", cfg.Flameshots[0].Processes)
	assert.True(t, cfg.Flameshots[0].EnablePrometheusAnnotations)

	if !assert.Len(t, cfg.Profilers, 1) {
		return
	}
	assert.True(t, cfg.Profilers[0].CheckAnnotation)
	assert.Equal(t, "java", cfg.Profilers[0].Language)

	assert.NoError(t, cfg.Setup())
	ddtraceMatched, ddtraceRules := cfg.DDTraces.MatchesAll("default", map[string]string{"app": "web"})
	assert.True(t, ddtraceMatched)
	assert.Equal(t, cfg.DDTraces, DDTraceRules(ddtraceRules))

	logfwdMatched, logfwdRule := cfg.Logfwds.Matches("default", nil)
	assert.True(t, logfwdMatched)
	assert.Equal(t, cfg.Logfwds[0], logfwdRule)

	flameshotMatched, flameshotRule := cfg.Flameshots.Matches("default", nil)
	assert.True(t, flameshotMatched)
	assert.Equal(t, cfg.Flameshots[0], flameshotRule)

	profilerMatched, profilerRule := cfg.Profilers.Matches("default", nil)
	assert.True(t, profilerMatched)
	assert.Equal(t, cfg.Profilers[0], profilerRule)
}
