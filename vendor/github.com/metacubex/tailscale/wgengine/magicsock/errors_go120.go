//go:build !go1.21

package magicsock

import "errors"

var ErrUnsupported = errors.New("unsupported operation")
