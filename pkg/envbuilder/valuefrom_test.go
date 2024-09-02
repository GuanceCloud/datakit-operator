package envbuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertFieldPath(t *testing.T) {
	cases := []struct {
		in     string
		output string
	}{
		{
			in:     "{fieldRef:metadata.name}",
			output: "metadata.name",
		},
		{
			in:     "{fieldRef:metadata.labels['app']}",
			output: "metadata.labels['app']",
		},
		{
			in:     "{fieldRef:metadata.annotations['app']}",
			output: "metadata.annotations['app']",
		},
		{
			in:     "unmatched",
			output: "",
		},
	}

	for _, tc := range cases {
		res := converFieldPath(tc.in)
		assert.Equal(t, tc.output, res)
	}
}
