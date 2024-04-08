package selector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	testcases := []struct {
		in   string
		fail bool
	}{
		{
			in:   "environment=production,tier=frontend",
			fail: false,
		},
		{
			in:   "environment in (production),tier in (frontend)",
			fail: false,
		},
		{
			in:   "environment in (production, qa)",
			fail: false,
		},
		{
			in:   "environment,environment notin (frontend)",
			fail: false,
		},
		{
			in:   "environment(error)xxxxxx",
			fail: true,
		},
	}

	for _, tc := range testcases {
		_, err := ParseSelector(tc.in)
		if tc.fail {
			assert.Error(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}
