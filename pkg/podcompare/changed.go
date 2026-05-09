// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package podcompare

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
)

func Changed(before, after *corev1.Pod) bool {
	return !reflect.DeepEqual(before, after)
}
