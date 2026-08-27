// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package scripts

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
)

func TestDeploymentTemplatesUseTracingDefaults(t *testing.T) {
	paths := []string{
		"../templates/charts-values.template.yaml",
		"../templates/datakit-operator.template.yaml",
	}

	configs := make([]string, 0, len(paths))
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			cfg, rawConfig := readTemplateConfiguration(t, path)
			configs = append(configs, rawConfig)

			if len(cfg.AdmissionInject.DDTraces) != 1 {
				t.Fatalf("default ddtraces must contain only Java, got %d rules", len(cfg.AdmissionInject.DDTraces))
			}
			java := cfg.AdmissionInject.DDTraces[0]
			if java.Language != "java" {
				t.Fatalf("default DDTrace language = %q, want java", java.Language)
			}
			if !reflect.DeepEqual(java.Namespaces, []string{"default"}) || len(java.Labels) != 0 {
				t.Fatalf("default Java DDTrace selectors = namespaces %v, labels %v", java.Namespaces, java.Labels)
			}
			if len(cfg.AdmissionInject.OTels) != 0 {
				t.Fatalf("default otels must be empty, got %d rules", len(cfg.AdmissionInject.OTels))
			}
		})
	}

	if len(configs) == len(paths) && configs[0] != configs[1] {
		t.Fatal("Helm values and single-manifest templates must contain identical default JSON configuration")
	}
}

func TestFlameshotDocumentationImageMatchesDeploymentDefaults(t *testing.T) {
	wantTag := readExportImageTag(t, "../export/config.env", "FlameshotImage")
	for _, path := range []string{
		"../templates/charts-values.template.yaml",
		"../templates/datakit-operator.template.yaml",
	} {
		t.Run(path, func(t *testing.T) {
			cfg, _ := readTemplateConfiguration(t, path)
			if len(cfg.AdmissionInject.Flameshots) != 1 {
				t.Fatalf("default flameshots must contain one rule, got %d", len(cfg.AdmissionInject.Flameshots))
			}
			if got := imageTag(t, cfg.AdmissionInject.Flameshots[0].Image); got != wantTag {
				t.Fatalf("default Flameshot image tag = %q, documentation uses %q", got, wantTag)
			}
		})
	}
}

func readExportImageTag(t *testing.T, path, key string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	prefix := key + "="
	for _, line := range strings.Split(string(data), "\n") {
		if image, found := strings.CutPrefix(line, prefix); found {
			return imageTag(t, image)
		}
	}
	t.Fatalf("%s does not define %s", path, key)
	return ""
}

func imageTag(t *testing.T, image string) string {
	t.Helper()
	idx := strings.LastIndex(image, ":")
	if idx == -1 || idx < strings.LastIndex(image, "/") {
		t.Fatalf("image %q does not contain a tag", image)
	}
	return image[idx+1:]
}

func readTemplateConfiguration(t *testing.T, path string) (config.Configuration, string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	const marker = "  jsonconfig: |-\n"
	_, content, found := strings.Cut(string(data), marker)
	if !found {
		t.Fatalf("%s does not contain jsonconfig", path)
	}

	var lines []string
	for _, line := range strings.Split(content, "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "    ") {
			break
		}
		lines = append(lines, strings.TrimPrefix(line, "    "))
	}

	rawConfig := strings.Join(lines, "\n")
	var cfg config.Configuration
	if err := json.Unmarshal([]byte(rawConfig), &cfg); err != nil {
		t.Fatalf("parse %s jsonconfig: %v", path, err)
	}
	return cfg, rawConfig
}
