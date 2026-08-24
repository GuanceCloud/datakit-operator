// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckMarkdown(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		checkSection bool
		wantErr      bool
	}{
		{
			name:         "valid document",
			content:      "# Document\n\n## Section {#section}\n\nContent.\n",
			checkSection: true,
		},
		{
			name:         "missing section anchor",
			content:      "# Document\n\n## Section\n\nContent.\n",
			checkSection: true,
			wantErr:      true,
		},
		{
			name:         "section check disabled",
			content:      "# Document\n\n## Section\n\nContent.\n",
			checkSection: false,
		},
		{
			name:         "invalid Chinese spacing",
			content:      "# Document\n\n中文English\n",
			checkSection: true,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "document.md")
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}

			var output bytes.Buffer
			err := checkMarkdown(dir, tt.checkSection, &output)
			if tt.wantErr {
				assert.Error(t, err)
				assert.NotEmpty(t, output.String())
			} else {
				assert.NoError(t, err)
				assert.Empty(t, output.String())
			}

			actual, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			assert.Equal(t, tt.content, string(actual), "document checks must not modify files")
		})
	}
}
