package logger

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

// captureLog redirects the standard logger to a buffer without timestamps for the duration of fn
func captureLog(fn func()) string {
	var buf bytes.Buffer
	flags, output := log.Flags(), log.Writer()
	log.SetFlags(0)
	log.SetOutput(&buf)
	defer func() {
		log.SetFlags(flags)
		log.SetOutput(output)
	}()

	fn()
	return buf.String()
}

func TestBasicLogger(t *testing.T) {
	l := MakeBasicLogger()

	assert.Equal(t, "info message\n", captureLog(func() { l.Info("info message") }))
	assert.Equal(t, "count=3\n", captureLog(func() { l.Infof("count=%d\n", 3) }))
	assert.Equal(t, "error message\n", captureLog(func() { l.Error("error message") }))
	assert.Equal(t, "failed: boom\n", captureLog(func() { l.Errorf("failed: %s\n", "boom") }))
}
