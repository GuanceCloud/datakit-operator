// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckTranslations(t *testing.T) {
	root := t.TempDir()
	source := "## 安装 {#install}\n\n{{.Image}}\n\n<<<custom_key.brand_name>>>\n"
	writeDocument(t, root, "zh", "guide.md", source)
	for _, lang := range []string{"en", "ja", "ko"} {
		writeDocument(t, root, lang, "guide.md", source)
		writeMetadata(t, root, lang, map[string]string{"guide.md": sourceHash(source)})
	}

	if err := checkTranslations(root, "zh", []string{"en", "ja", "ko"}); err != nil {
		t.Fatalf("checkTranslations() error = %v", err)
	}
}

func TestCheckTranslationsRejectsStaleSourceHash(t *testing.T) {
	root := t.TempDir()
	source := "# 文档\n"
	writeDocument(t, root, "zh", "guide.md", source)
	writeDocument(t, root, "en", "guide.md", "# Guide\n")
	writeMetadata(t, root, "en", map[string]string{"guide.md": sourceHash("old")})

	err := checkTranslations(root, "zh", []string{"en"})
	if err == nil || !strings.Contains(err.Error(), "stale translation metadata") {
		t.Fatalf("checkTranslations() error = %v, want stale metadata error", err)
	}
}

func TestCheckTranslationsRejectsChangedProtectedSyntax(t *testing.T) {
	root := t.TempDir()
	source := "## 安装 {#install}\n\n{{.Image}}\n"
	writeDocument(t, root, "zh", "guide.md", source)
	writeDocument(t, root, "ja", "guide.md", "## インストール {#installation}\n\n{{.Image}}\n")
	writeMetadata(t, root, "ja", map[string]string{"guide.md": sourceHash(source)})

	err := checkTranslations(root, "zh", []string{"ja"})
	if err == nil || !strings.Contains(err.Error(), "markdown attributes differ") {
		t.Fatalf("checkTranslations() error = %v, want protected syntax error", err)
	}
}

func TestCheckTranslationsRejectsChangedFencedCodeBracketSignatures(t *testing.T) {
	root := t.TempDir()
	source := "```json\n{\"rules\":[{\"name\":\"demo\"}]}\n```\n"
	writeDocument(t, root, "zh", "guide.md", source)
	writeDocument(t, root, "ko", "guide.md", "```json\n{\"rules\":[[\"name\":\"demo\"]]}\n```\n")
	writeMetadata(t, root, "ko", map[string]string{"guide.md": sourceHash(source)})

	err := checkTranslations(root, "zh", []string{"ko"})
	if err == nil || !strings.Contains(err.Error(), "fenced code bracket signatures differ") {
		t.Fatalf("checkTranslations() error = %v, want fenced code bracket signature error", err)
	}
}

func writeDocument(t *testing.T, root, lang, name, content string) {
	t.Helper()
	dir := filepath.Join(root, lang)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeMetadata(t *testing.T, root, lang string, hashes map[string]string) {
	t.Helper()
	value := map[string]interface{}{
		"version":         1,
		"target_language": lang,
		"source_hashes":   hashes,
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, lang, ".translation")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func sourceHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
