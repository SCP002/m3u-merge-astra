package file

import (
	"io"
	"os"

	"github.com/cockroachdb/errors"
)

// Copy copies `src` file path to `dst` file path, overwriting `dst` if it exists.
func Copy(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	// Write errors may only be reported on close.
	defer func() { err = errors.Join(err, out.Close()) }()

	_, err = io.Copy(out, in)
	return err
}
