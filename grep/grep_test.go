package grep

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrep(t *testing.T) {
	t.Parallel()

	tCases := []struct {
		Name           string
		Input          string
		Args           []string
		ExpectedStatus int
		Err            string
	}{
		{
			Name:           "grep match char",
			Input:          "aboba",
			Args:           []string{"grep", "-E", "a"},
			ExpectedStatus: statusCodeOK,
		},
		{
			Name:           "grep match char not found",
			Input:          "aboba",
			Args:           []string{"grep", "-E", "x"},
			ExpectedStatus: statusCodeNotFound,
		},
		{
			Name:           "grep invalid args",
			Args:           []string{"grep"},
			ExpectedStatus: statusCodeErr,
			Err:            "usage: mygrep -E <pattern>",
		},
		{
			Name:           "grep invalid pattern",
			Args:           []string{"grep", "-E", "[]()"},
			ExpectedStatus: statusCodeErr,
			Err:            "unsupported pattern: \"[]()\"",
		},
		{
			Name:           "grep invalid option",
			Args:           []string{"grep", "-A", "a"},
			ExpectedStatus: statusCodeErr,
			Err:            "usage: mygrep -E <pattern>",
		},
		{
			Name:           "grep match digit",
			Input:          "apple123",
			Args:           []string{"grep", "-E", `\d`},
			ExpectedStatus: statusCodeOK,
		},
		{
			Name:           "grep match digit not found",
			Input:          "apple",
			Args:           []string{"grep", "-E", `\d`},
			ExpectedStatus: statusCodeNotFound,
		},
		{
			Name:           "grep match alphanumeric",
			Input:          "foo101",
			Args:           []string{"grep", "-E", `\w`},
			ExpectedStatus: statusCodeOK,
		},
		{
			Name:           "grep match alphanumeric not found",
			Input:          "$?!",
			Args:           []string{"grep", "-E", `\w`},
			ExpectedStatus: statusCodeNotFound,
		},
		{
			Name:           "grep match alphanumeric not found more complex",
			Input:          "-+×_-€÷",
			Args:           []string{"grep", "-E", `\w`},
			ExpectedStatus: statusCodeOK,
		},
		{
			Name:           "grep match char group",
			Input:          "abc",
			Args:           []string{"grep", "-E", `[abc]`},
			ExpectedStatus: statusCodeOK,
		},
		{
			Name:           "grep match char group not found",
			Input:          "abc",
			Args:           []string{"grep", "-E", `[def]`},
			ExpectedStatus: statusCodeNotFound,
		},
		{
			Name:           "grep match char group input [] not found",
			Input:          "[]",
			Args:           []string{"grep", "-E", `[abc]`},
			ExpectedStatus: statusCodeNotFound,
		},
		{
			Name:           "grep match char empty group input [] not found",
			Input:          "abc",
			Args:           []string{"grep", "-E", `[]`},
			ExpectedStatus: statusCodeNotFound,
		},
		{
			Name:           "grep match char group negative",
			Input:          "abc",
			Args:           []string{"grep", "-E", `[^xyz]`},
			ExpectedStatus: statusCodeOK,
		},
		{
			Name:           "grep match char group negative not found",
			Input:          "abc",
			Args:           []string{"grep", "-E", `[^abc]`},
			ExpectedStatus: statusCodeNotFound,
		},
		{
			Name:           "grep match char group input [] negative",
			Input:          "[]",
			Args:           []string{"grep", "-E", `[^abc]`},
			ExpectedStatus: statusCodeOK,
		},
		{
			Name:           "grep match char empty group",
			Input:          "abc",
			Args:           []string{"grep", "-E", `[^]`},
			ExpectedStatus: statusCodeOK,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Name, func(t *testing.T) {
			t.Parallel()

			g := NewGrep(tCase.Args, strings.NewReader(tCase.Input))
			status, err := g.run()
			if tCase.Err != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tCase.Err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tCase.ExpectedStatus, status)
		})
	}
}
