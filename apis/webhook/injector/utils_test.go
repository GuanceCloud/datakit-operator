// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendKVPairs(t *testing.T) {
	cases := []struct {
		inOrigin string
		inNew    string
		output   string
	}{
		{
			inOrigin: "host:node01,name:nginx",
			inNew:    "app:system",
			output:   "host:node01,name:nginx,app:system",
		},
		{
			inOrigin: "host:node01,name:nginx",
			inNew:    "host:node02",
			output:   "host:node01,name:nginx",
		},
		{
			inOrigin: "",
			inNew:    "app:system",
			output:   "app:system",
		},
		{
			inOrigin: "host:node01,name:nginx",
			inNew:    "app::system",
			output:   "host:node01,name:nginx",
		},
	}

	for _, tc := range cases {
		res := appendKVPairs(tc.inOrigin, tc.inNew)
		assert.Equal(t, tc.output, res)
	}
}

func TestGetMountPath(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "path with wildcard",
			input:    "/data/logs/**/123",
			expected: "/data/logs",
		},
		{
			name:     "path without wildcard",
			input:    "/data/logs",
			expected: "/data/logs",
		},
		{
			name:     "root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "single level path",
			input:    "/data",
			expected: "/data",
		},
		{
			name:     "wildcard at beginning",
			input:    "*/logs",
			expected: ".",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := getMountPath(tc.input)
			assert.Equal(t, tc.expected, res)
		})
	}
}
