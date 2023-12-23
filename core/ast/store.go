package ast

import (
	"errors"
)

var ErrStoreNotFound = errors.New("store not found")

// Store is the interface that provides a standard mechansim for
// retreiving values from external sources during the evaluation
// phase.
type Store interface {
	// Get is used in the context of `principal.active` where
	// we are given the entity (e.g. `User:"alice"`) and should
	// return approprate attributes for that entity. This should be
	// a VarValue object as the current time
	Get(EntityValue) (EvalValue, error)
	// GetParents does a transitive lookup for a given entity as
	// a child of some other entity. i.e. `principal in Group::"admin"`
	// This returns a list of all transitive entitys.
	GetParents(EntityValue) ([]EntityValue, error)
}
