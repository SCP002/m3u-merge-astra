package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m3u-merge-astra-copy-test.txt")

	expected, err := os.ReadFile("copy-test.txt")
	require.NoError(t, err)

	err = Copy("copy-test.txt", path)
	assert.NoError(t, err)
	assert.FileExists(t, path)

	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Exactly(t, expected, actual)

	// Test overwrite of longer file, so the rest of it is truncated.
	err = os.WriteFile(path, []byte("longer content than in the source file"), 0644)
	require.NoError(t, err)

	err = Copy("copy-test.txt", path)
	assert.NoError(t, err)

	actual, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Exactly(t, expected, actual, "should truncate existing destination file")

	// Test missing source.
	err = Copy("missing.txt", path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
