package schema

import (
	"errors"

	"github.com/koblas/cedar-go/core/ast"
)

type EntityStoreItem struct {
	entity  ast.EntityValue
	parents []ast.EntityValue
	values  *ast.VarValue
}

type EntityStore map[string]EntityStoreItem

type EmptyStore struct{}

var ErrNotFoundInStore = errors.New("not found in store")

var _ ast.Store = (*EmptyStore)(nil)

func NewEmptyStore() *EmptyStore {
	return &EmptyStore{}
}

func (store *EmptyStore) Get(key ast.EntityValue) (ast.EvalValue, error) {
	return nil, nil
}

func (store *EmptyStore) GetParents(key ast.EntityValue) ([]ast.EntityValue, error) {
	return nil, nil
}

var _ ast.Store = (EntityStore)(nil)

func (store EntityStore) Get(key ast.EntityValue) (ast.EvalValue, error) {
	value, found := store[key.String()]

	if !found {
		return nil, ast.ErrValueNotFound
	}

	return value.values, nil
}

func (store EntityStore) GetParents(key ast.EntityValue) ([]ast.EntityValue, error) {
	seen := map[string]ast.EntityValue{}
	todo := []ast.EntityValue{key}

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

	output := []ast.EntityValue{}
	for _, value := range seen {
		output = append(output, value)
	}

	return output, nil
}
