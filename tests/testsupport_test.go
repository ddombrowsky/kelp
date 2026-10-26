package tests

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomString(t *testing.T) {
	alphanumeric := regexp.MustCompile("^[a-zA-Z0-9]*$")

	assert.Len(t, RandomString(), 10)
	for _, n := range []int{0, 1, 32} {
		s := RandomStringWithLen(n)
		assert.Len(t, s, n)
		assert.Regexp(t, alphanumeric, s)
	}
}

func TestRandomInt(t *testing.T) {
	for i := 0; i < 100; i++ {
		v := RandomInt()
		assert.True(t, v >= 0 && v < 10, "RandomInt() = %d", v)

		v = RandomIntWithMax(3)
		assert.True(t, v >= 0 && v < 3, "RandomIntWithMax(3) = %d", v)
	}
}
