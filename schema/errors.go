package schema

import "errors"

var ErrInvalidEntityFormat = errors.New("invalid entity format")
var ErrUnsupportedType = errors.New("unsupported type in store generation")
var ErrInvalidMapKey = errors.New("invalid map key")
var ErrValueNotFound = errors.New("value not found in store")
