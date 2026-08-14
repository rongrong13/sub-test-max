//go:build !go1.21

package dns

import "errors"

var ErrUnsupported = errors.New("unsupported operation")
