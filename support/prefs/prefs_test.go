package prefs

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirstTime(t *testing.T) {
	p := Make(filepath.Join(t.TempDir(), "kelp.prefs"))
	assert.True(t, p.FirstTime())

	assert.NoError(t, p.SetNotFirstTime())
	assert.False(t, p.FirstTime())

	// a second Preferences on the same file sees the saved state
	assert.False(t, Make(p.filepath).FirstTime())
}

func TestSetNotFirstTime_UnwritablePath(t *testing.T) {
	p := Make(filepath.Join(t.TempDir(), "missing-dir", "kelp.prefs"))
	assert.Error(t, p.SetNotFirstTime())
	assert.True(t, p.FirstTime())
}
