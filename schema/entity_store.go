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

func (store *EmptyStore) Get(key engine.EntityValue) (engine.EvalValue, error) {
	return nil, nil
}

func (store *EmptyStore) GetParents(key engine.EntityValue) ([]engine.EntityValue, error) {
	return nil, nil
}

var _ engine.Store = (EntityStore)(nil)

func (store EntityStore) Get(key engine.EntityValue) (engine.EvalValue, error) {
	value, found := store[key.String()]

	if !found {
		return nil, engine.ErrValueNotFound
	}

	return value.values, nil
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
