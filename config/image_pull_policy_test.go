// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestImagePullPolicyConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		want        corev1.PullPolicy
		warning     bool
	}{
		{"missing", "", corev1.PullAlways, false},
		{"empty", `,"image_pull_policy":""`, corev1.PullAlways, false},
		{"null", `,"image_pull_policy":null`, corev1.PullAlways, false},
		{"always", `,"image_pull_policy":"Always"`, corev1.PullAlways, false},
		{"cached", `,"image_pull_policy":"IfNotPresent"`, corev1.PullIfNotPresent, false},
		{"never", `,"image_pull_policy":"Never"`, corev1.PullNever, false},
		{"invalid", `,"image_pull_policy":"ifnotpresent"`, corev1.PullAlways, true},
		{"wrong type", `,"image_pull_policy":true`, corev1.PullAlways, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logs := captureWarningLogs(t)
			var cfg Configuration
			raw := fmt.Sprintf(`{"admission_inject_v2":{"ddtraces":[{
				"namespace_selectors":["default"]%s
			}]}}`, tc.field)
			if !assert.NoError(t, parseConfig(raw, &cfg)) || !assert.Len(t, cfg.AdmissionInject.DDTraces, 1) {
				return
			}
			assert.Equal(t, tc.want, cfg.AdmissionInject.DDTraces[0].ImagePullPolicy.Value())
			if tc.warning {
				assert.Contains(t, logs.String(), "invalid image_pull_policy")
				assert.Contains(t, logs.String(), "fallback=Always")
			} else {
				assert.Empty(t, logs.String())
			}
		})
	}
}
