// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type translationMetadata struct {
	Version        int               `json:"version"`
	TargetLanguage string            `json:"target_language"`
	SourceHashes   map[string]string `json:"source_hashes"`
}

var protectedSyntax = []struct {
	name       string
	expression *regexp.Regexp
}{
	{name: "template placeholders", expression: regexp.MustCompile(`(?s)\{\{.*?\}\}`)},
	{name: "brand variables", expression: regexp.MustCompile(`(?s)<<<.*?>>>`)},
	{name: "markdown attributes", expression: regexp.MustCompile(`\{[ \t]*(?:[#.:][^{}\n]*)\}`)},
}

var fenceLine = regexp.MustCompile("^[ \\t]*(`{3,}|~{3,})[^\\n]*$")

func listMarkdownDocuments(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read document directory %s: %w", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("no Markdown documents found in %s", dir)
	}
	return names, nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func loadMetadata(root, language string) (*translationMetadata, error) {
	path := filepath.Join(root, language, ".translation", "metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read translation metadata %s: %w", path, err)
	}
	var metadata translationMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("decode translation metadata %s: %w", path, err)
	}
	if metadata.Version != 1 {
		return nil, fmt.Errorf("unsupported translation metadata version %d in %s", metadata.Version, path)
	}
	if metadata.TargetLanguage != language {
		return nil, fmt.Errorf("translation metadata %s targets %q, want %q", path, metadata.TargetLanguage, language)
	}
	return &metadata, nil
}

func hashDocument(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func checkProtectedSyntax(sourceLanguage, targetLanguage, name string, source, target []byte) error {
	for _, syntax := range protectedSyntax {
		sourceValues := syntax.expression.FindAllString(string(source), -1)
		targetValues := syntax.expression.FindAllString(string(target), -1)
		if !sameStrings(sourceValues, targetValues) {
			return fmt.Errorf("%s differ in %s/%s and %s/%s", syntax.name, sourceLanguage, name, targetLanguage, name)
		}
	}

	sourceBracketSignatures, err := fencedCodeBracketSignatures(source)
	if err != nil {
		return fmt.Errorf("read fenced code bracket signatures in %s/%s: %w", sourceLanguage, name, err)
	}
	targetBracketSignatures, err := fencedCodeBracketSignatures(target)
	if err != nil {
		return fmt.Errorf("read fenced code bracket signatures in %s/%s: %w", targetLanguage, name, err)
	}
	if !sameStrings(sourceBracketSignatures, targetBracketSignatures) {
		return fmt.Errorf("fenced code bracket signatures differ in %s/%s and %s/%s", sourceLanguage, name, targetLanguage, name)
	}
	return nil
}

func fencedCodeBracketSignatures(content []byte) ([]string, error) {
	var signatures []string
	var brackets strings.Builder
	var fenceMarker byte
	var fenceLength int

	for _, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSuffix(rawLine, "\r")
		match := fenceLine.FindStringSubmatch(line)
		if fenceMarker == 0 {
			if match != nil {
				fenceMarker = match[1][0]
				fenceLength = len(match[1])
				brackets.Reset()
			}
			continue
		}
		if match != nil && match[1][0] == fenceMarker && len(match[1]) >= fenceLength {
			signatures = append(signatures, brackets.String())
			fenceMarker = 0
			fenceLength = 0
			continue
		}
		for _, character := range line {
			if strings.ContainsRune("{}[]", character) {
				brackets.WriteRune(character)
			}
		}
	}
	if fenceMarker != 0 {
		return nil, errors.New("unclosed Markdown code fence")
	}
	return signatures, nil
}

func checkTranslations(root, sourceLanguage string, targetLanguages []string) error {
	sourceDir := filepath.Join(root, sourceLanguage)
	sourceDocuments, err := listMarkdownDocuments(sourceDir)
	if err != nil {
		return err
	}

	for _, language := range targetLanguages {
		if language == "" {
			return errors.New("target language must not be empty")
		}
		targetDir := filepath.Join(root, language)
		targetDocuments, err := listMarkdownDocuments(targetDir)
		if err != nil {
			return err
		}
		if !sameStrings(sourceDocuments, targetDocuments) {
			return fmt.Errorf("markdown document filenames differ in %s and %s", sourceDir, targetDir)
		}

		metadata, err := loadMetadata(root, language)
		if err != nil {
			return err
		}
		for _, name := range sourceDocuments {
			sourcePath := filepath.Join(sourceDir, name)
			targetPath := filepath.Join(targetDir, name)
			sourceHash, err := hashDocument(sourcePath)
			if err != nil {
				return fmt.Errorf("hash source document %s: %w", sourcePath, err)
			}
			translatedSourceHash, ok := metadata.SourceHashes[name]
			if !ok || translatedSourceHash != sourceHash {
				return fmt.Errorf("stale translation metadata for %s/%s", language, name)
			}
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				return fmt.Errorf("read source document %s: %w", sourcePath, err)
			}
			target, err := os.ReadFile(targetPath)
			if err != nil {
				return fmt.Errorf("read translated document %s: %w", targetPath, err)
			}
			if len(strings.TrimSpace(string(target))) == 0 {
				return fmt.Errorf("translated document is empty: %s", targetPath)
			}
			if err := checkProtectedSyntax(sourceLanguage, language, name, source, target); err != nil {
				return err
			}
		}
	}
	return nil
}

func main() {
	if err := checkTranslations("export", "zh", []string{"en", "ja", "ko"}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
