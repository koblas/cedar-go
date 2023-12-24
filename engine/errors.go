package engine

import (
	"errors"
)

var ErrUnsupportedType = errors.New("unsupported type in store generation")
var ErrInvalidMapKey = errors.New("invalid map key")
var ErrInvalidEntityFormat = errors.New("invalid entity format")
var ErrValueNotFound = errors.New("value not found in store")
var ErrNotImplemented = errors.New("not implemented")
