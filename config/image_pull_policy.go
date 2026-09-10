// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package config

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
)

// ImagePullPolicy accepts invalid configuration values so they can fall back to
// Always without preventing the rest of the configuration from loading.
type ImagePullPolicy string

func (p *ImagePullPolicy) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		value = string(data)
	}
	*p = ImagePullPolicy(value)
	if value != "" && corev1.PullPolicy(value) != p.Value() {
		log.Warnf("invalid image_pull_policy: value=%q fallback=%s", value, corev1.PullAlways)
	}
	return nil
}

// Value returns the configured policy, defaulting to Always.
func (p ImagePullPolicy) Value() corev1.PullPolicy {
	switch corev1.PullPolicy(p) {
	case corev1.PullAlways, corev1.PullIfNotPresent, corev1.PullNever:
		return corev1.PullPolicy(p)
	default:
		return corev1.PullAlways
	}
}
