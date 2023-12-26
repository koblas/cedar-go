package schema

import (
	"errors"

	"github.com/koblas/cedar-go/engine"
)

type EntityStoreItem struct {
	entity  engine.EntityValue
	parents []engine.EntityValue
	values  *engine.VarValue
}

type EntityStore map[string]EntityStoreItem

type EmptyStore struct{}

var ErrNotFoundInStore = errors.New("not found in store")

var _ engine.Store = (*EmptyStore)(nil)

func NewEmptyStore() *EmptyStore {
	return &EmptyStore{}
}

func (store *EmptyStore) Get(key engine.EntityValue, str string) (engine.EvalValue, error) {
	return nil, nil
}

func (store *EmptyStore) GetParents(key engine.EntityValue) ([]engine.EntityValue, error) {
	return nil, nil
}

var _ engine.Store = (EntityStore)(nil)

func (store EntityStore) Get(key engine.EntityValue, str string) (engine.EvalValue, error) {
	value, found := store[key.String()]

	if !found || value.values == nil {
		return nil, engine.ErrValueNotFound
	}

	// In a "database" oritend Store we could just fetch the right
	// column.  In this case we're re-using the the knowledge that
	// this is an entity store with VarValue as the map type
	val, err := value.values.OpLookup(engine.StrValue(str), store)
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (store EntityStore) GetParents(key engine.EntityValue) ([]engine.EntityValue, error) {
	seen := map[string]engine.EntityValue{}
	todo := []engine.EntityValue{key}

	for len(todo) != 0 {
		first := todo[0]
		todo = todo[1:]

		lookup := first.String()
		if _, found := seen[lookup]; found {
			continue
		}
		seen[lookup] = first

		value, found := store[lookup]
		if !found {
			continue
		}

		todo = append(todo, value.parents...)
	}

	output := []engine.EntityValue{}
	for _, value := range seen {
		output = append(output, value)
	}

	return output, nil
}
