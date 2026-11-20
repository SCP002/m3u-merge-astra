package copier

import (
	"github.com/cockroachdb/errors"
	"github.com/jinzhu/copier"
)

// MustDeep returns deep copy of `inp`, panicking if copier fails.
//
// Used in tests to keep original values for comparison.
func MustDeep[T any](inp T) T {
	out, err := deep(inp)
	if err != nil {
		panic(err)
	}
	return out
}

// deep returns deep copy of `inp`.
func deep[T any](inp T) (out T, err error) {
	err = copier.CopyWithOption(&out, &inp, copier.Option{DeepCopy: true, IgnoreEmpty: true})
	err = errors.Wrap(err, "Copier")
	return
}
