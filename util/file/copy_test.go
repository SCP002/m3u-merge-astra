package file

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m3u-merge-astra-copy-test.txt")

	err := Copy("copy-test.txt", path)
	assert.NoError(t, err)

	// Test overwrite.
	err = Copy("copy-test.txt", path)
	assert.NoError(t, err)

	assert.FileExists(t, path)
}
