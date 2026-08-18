// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package scripts

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

var chinaStandardTime = time.FixedZone("China Standard Time", 8*60*60)

func rcVersionForDate(base string, date time.Time) string {
	return base + "-rc-" + date.In(chinaStandardTime).Format("20060102")
}

func repositoryVersion(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile("../Makefile")
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VERSION=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "VERSION="))
		}
	}
	t.Fatal("Makefile does not declare VERSION")
	return ""
}

func checkRCVersion(version string) ([]byte, error) {
	cmd := exec.Command("make", "check_rc_version", "RC_VERSION="+version)
	cmd.Dir = ".."
	return cmd.CombinedOutput()
}

func TestRCImageReleasePublishesOnlyImmutableImageTags(t *testing.T) {
	rcVersion := rcVersionForDate(repositoryVersion(t), time.Now())
	cmd := exec.Command("make", "--dry-run", "pub_rc_image", "RC_VERSION="+rcVersion)
	cmd.Dir = ".."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dry-run RC image release: %v\n%s", err, output)
	}

	releasePlan := string(output)
	expectedImages := []string{
		"registry.jiagouyun.com/datakit-operator/datakit-operator:" + rcVersion,
		"pubrepo.guance.com/datakit-operator/datakit-operator:" + rcVersion,
		"pubrepo.truewatch.com/truewatch/datakit-operator:" + rcVersion,
	}
	for _, image := range expectedImages {
		if !strings.Contains(releasePlan, "-t "+image) {
			t.Errorf("RC release does not publish %q:\n%s", image, releasePlan)
		}
	}

	if count := strings.Count(releasePlan, "docker buildx build"); count != 1 {
		t.Errorf("RC images must come from one buildx invocation, got %d:\n%s", count, releasePlan)
	}
	if !strings.Contains(releasePlan, "--platform linux/arm64,linux/amd64") {
		t.Errorf("RC release must build both supported architectures:\n%s", releasePlan)
	}

	for _, forbidden := range []string{":latest", "helm ", "brand.sh", "upload.sh", "Dockerfile.uos"} {
		if strings.Contains(releasePlan, forbidden) {
			t.Errorf("RC image-only release unexpectedly contains %q:\n%s", forbidden, releasePlan)
		}
	}
}

func TestRCImageReleaseValidatesVersion(t *testing.T) {
	baseVersion := repositoryVersion(t)
	t.Run("valid current release", func(t *testing.T) {
		version := rcVersionForDate(baseVersion, time.Now())
		output, err := checkRCVersion(version)
		if err != nil {
			t.Fatalf("valid RC version was rejected: %v\n%s", err, output)
		}
	})

	tests := []struct {
		name      string
		version   string
		wantError string
	}{
		{
			name:      "invalid format",
			version:   baseVersion + "-rc.1",
			wantError: "RC_VERSION must match vX.Y.Z-rc-YYYYMMDD",
		},
		{
			name:      "missing version prefix",
			version:   "1.8.10-rc-20260818",
			wantError: "RC_VERSION must match vX.Y.Z-rc-YYYYMMDD",
		},
		{
			name:      "incomplete semantic version",
			version:   "v1.8-rc-20260818",
			wantError: "RC_VERSION must match vX.Y.Z-rc-YYYYMMDD",
		},
		{
			name:      "invalid date width",
			version:   baseVersion + "-rc-2026081",
			wantError: "RC_VERSION must match vX.Y.Z-rc-YYYYMMDD",
		},
		{
			name:      "different base version",
			version:   "v999.999.999-rc-20000101",
			wantError: "RC base version must match VERSION " + baseVersion,
		},
		{
			name:      "non-current date",
			version:   baseVersion + "-rc-20000101",
			wantError: "RC date must match current date ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output, err := checkRCVersion(tc.version)
			if err == nil {
				t.Fatalf("invalid RC version %q was accepted", tc.version)
			}
			if !strings.Contains(string(output), tc.wantError) {
				t.Fatalf("unexpected validation error for %q:\n%s", tc.version, output)
			}
		})
	}
}

func TestGitLabCIPublishesRCImagesFromRCTags(t *testing.T) {
	data, err := os.ReadFile("../.gitlab-ci.yml")
	if err != nil {
		t.Fatalf("read GitLab CI config: %v", err)
	}

	contents := string(data)
	start := strings.Index(contents, "release-rc-image:")
	if start < 0 {
		t.Fatal("GitLab CI config does not define release-rc-image")
	}
	jobLines := strings.Split(contents[start:], "\n")
	end := len(jobLines)
	for idx := 1; idx < len(jobLines); idx++ {
		line := jobLines[idx]
		if line != "" && line[0] != ' ' && line[0] != '\t' && !strings.HasPrefix(line, "#") {
			end = idx
			break
		}
	}
	job := strings.Join(jobLines[:end], "\n")

	for _, required := range []string{
		"needs: [test]",
		`$CI_COMMIT_TAG =~ /^v[0-9]+\.[0-9]+\.[0-9]+-rc-[0-9]{8}$/`,
		`make VERSION="$CI_COMMIT_TAG"`,
		`make pub_rc_image RC_VERSION="$CI_COMMIT_TAG"`,
	} {
		if !strings.Contains(job, required) {
			t.Errorf("RC release job does not contain %q:\n%s", required, job)
		}
	}
	for _, forbidden := range []string{"pub_image", "pub_testing_image", "pub_uos_image"} {
		if strings.Contains(job, forbidden) {
			t.Errorf("RC release job unexpectedly contains %q:\n%s", forbidden, job)
		}
	}
}
