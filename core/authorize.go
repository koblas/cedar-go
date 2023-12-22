package core

import (
	"context"

	"github.com/koblas/cedar-go/core/ast"
	"github.com/koblas/cedar-go/core/schema"
)

type Request struct {
	Principal ast.EntityValue
	Action    ast.EntityValue
	Resource  ast.EntityValue
	Context   *ast.VarValue
}

type Authorizer interface {
	IsAuthorized(ctx context.Context, request *Request) (bool, error)
}

//
//

type SchemaAuthorizer struct {
	Policies ast.PolicyList
	Schema   *schema.Schema
	Store    ast.Store
}

type EmptyStore struct{}

//
//

type Option func(*SchemaAuthorizer)

func WithSchema(s *schema.Schema) Option {
	return func(sa *SchemaAuthorizer) {
		sa.Schema = s
	}
}

func WithStore(s ast.Store) Option {
	return func(sa *SchemaAuthorizer) {
		sa.Store = s
	}
}

func NewAuthorizer(p ast.PolicyList, options ...Option) *SchemaAuthorizer {
	conf := SchemaAuthorizer{
		Policies: p,
		Store:    schema.NewEmptyStore(),
	}

	for _, opt := range options {
		opt(&conf)
	}

	return &conf
}

func (auth *SchemaAuthorizer) IsAuthorized(ctx context.Context, request *Request) (bool, error) {
	// TODO
	// if Schema is not null
	// - lookup action
	// - validate context against action info

	req := ast.Request{
		Principal: request.Principal,
		Action:    request.Action,
		Resource:  request.Resource,
		Context:   request.Context,
		Store:     auth.Store,
		Trace:     false,
	}

	result, err := ast.Eval(ctx, auth.Policies, &req)
	if err != nil {
		return false, err
	}
	return result.Decision == ast.Allow, nil
}
