package admission

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetGenerateName(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{
			in:  "",
			out: "<no-one>",
		},
		{
			in:  "deployment",
			out: "deployment",
		},
		{
			in:  "deployment-",
			out: "deployment",
		},
		{
			in:  "deployment-replicaset",
			out: "deployment-replicaset",
		},
	}

	for _, tc := range cases {
		res := getGenerateName(tc.in)
		assert.Equal(t, tc.out, res)
	}

}
