package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSubmitMode(t *testing.T) {
	testCases := []struct {
		input string
		want  SubmitMode
	}{
		{input: "maker_only", want: SubmitModeMakerOnly},
		{input: "both", want: SubmitModeBoth},
		{input: "", want: SubmitModeBoth},
	}

	for _, k := range testCases {
		got, e := ParseSubmitMode(k.input)
		if assert.NoError(t, e, k.input) {
			assert.Equal(t, k.want, got, k.input)
		}
	}

	_, e := ParseSubmitMode("taker_only")
	assert.Error(t, e)
}

func TestSubmitModeString(t *testing.T) {
	for _, s := range []string{"maker_only", "both"} {
		mode, e := ParseSubmitMode(s)
		if assert.NoError(t, e) {
			assert.Equal(t, s, mode.String())
		}
	}
}
