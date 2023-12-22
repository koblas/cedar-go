package ast

import (
	"errors"
)

var ErrStoreNotFound = errors.New("store not found")

type Store interface {
	Get(EntityValue) (EvalValue, error)
	GetParents(EntityValue) ([]EntityValue, error)
}
