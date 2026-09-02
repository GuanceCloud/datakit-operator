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
)

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

func repositoryCommitTag(t *testing.T) string {
	t.Helper()

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = ".."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("resolve repository commit: %v", err)
	}
	commit := strings.TrimSpace(string(output))
	if len(commit) < 12 {
		t.Fatalf("repository commit %q is shorter than 12 characters", commit)
	}
	return "sha-" + commit[:12]
}

func TestRCImageReleasePublishesDateAndCommitTags(t *testing.T) {
	const releaseDate = "20000101"
	rcVersion := repositoryVersion(t) + "-rc-" + releaseDate
	shaVersion := repositoryCommitTag(t)
	cmd := exec.Command("make", "--dry-run", "pub_rc_image", "RC_DATE="+releaseDate)
	cmd.Dir = ".."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dry-run RC image release: %v\n%s", err, output)
	}

	releasePlan := string(output)
	expectedImages := []string{
		"registry.jiagouyun.com/datakit-operator/datakit-operator:" + rcVersion,
		"registry.jiagouyun.com/datakit-operator/datakit-operator:" + shaVersion,
		"pubrepo.guance.com/datakit-operator/datakit-operator:" + rcVersion,
		"pubrepo.guance.com/datakit-operator/datakit-operator:" + shaVersion,
		"pubrepo.guance.com/truewatch/datakit-operator:" + rcVersion,
		"pubrepo.guance.com/truewatch/datakit-operator:" + shaVersion,
	}
	for _, image := range expectedImages {
		if !strings.Contains(releasePlan, "-t "+image) {
			t.Errorf("RC release does not publish %q:\n%s", image, releasePlan)
		}
	}
	if count := strings.Count(releasePlan, "-t "); count != len(expectedImages) {
		t.Errorf("RC release must publish exactly %d image tags, got %d:\n%s", len(expectedImages), count, releasePlan)
	}

	if count := strings.Count(releasePlan, "docker buildx build"); count != 1 {
		t.Errorf("RC images must come from one buildx invocation, got %d:\n%s", count, releasePlan)
	}
	if !strings.Contains(releasePlan, "--platform linux/arm64,linux/amd64") {
		t.Errorf("RC release must build both supported architectures:\n%s", releasePlan)
	}
	if !strings.Contains(releasePlan, `make local VERSION="`+rcVersion+`"`) {
		t.Errorf("RC release must build binaries with version %q:\n%s", rcVersion, releasePlan)
	}

	for _, forbidden := range []string{":latest", "helm ", "brand.sh", "upload.sh", "Dockerfile.uos"} {
		if strings.Contains(releasePlan, forbidden) {
			t.Errorf("RC image-only release unexpectedly contains %q:\n%s", forbidden, releasePlan)
		}
	}
}

func TestGitLabCIExposesManualRCImageReleaseForBranchPipelines(t *testing.T) {
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
		`if: '$CI_COMMIT_BRANCH'`,
		"when: manual",
		"allow_failure: true",
		"script:\n    - make pub_rc_image",
	} {
		if !strings.Contains(job, required) {
			t.Errorf("RC release job does not contain %q:\n%s", required, job)
		}
	}
	if count := strings.Count(job, "\n    - make "); count != 1 {
		t.Errorf("RC release job must delegate to exactly one Make target, got %d calls:\n%s", count, job)
	}
	for _, forbidden := range []string{
		"CI_COMMIT_TAG",
		"CI_COMMIT_SHA",
		"RC_DATE",
		"RC_VERSION",
		"SHA_VERSION",
		"pub_image",
		"pub_testing_image",
		"pub_uos_image",
	} {
		if strings.Contains(job, forbidden) {
			t.Errorf("RC release job unexpectedly contains %q:\n%s", forbidden, job)
		}
	}
}
