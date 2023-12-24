package cedar

import (
	"context"

	"github.com/koblas/cedar-go/engine"
	"github.com/koblas/cedar-go/schema"
)

type Request struct {
	Principal engine.EntityValue
	Action    engine.EntityValue
	Resource  engine.EntityValue
	Context   *engine.VarValue
}

type Detail struct {
	IsAllowed bool
	Matches   []string
}

type Authorizer interface {
	IsAuthorized(ctx context.Context, request *Request) (bool, error)
}

//
//

type SchemaAuthorizer struct {
	Policies engine.PolicyList
	Schema   *schema.Schema
	Store    engine.Store
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

func WithStore(s engine.Store) Option {
	return func(sa *SchemaAuthorizer) {
		sa.Store = s
	}
}

func NewAuthorizer(p engine.PolicyList, options ...Option) *SchemaAuthorizer {
	conf := SchemaAuthorizer{
		Policies: p,
		Store:    schema.NewEmptyStore(),
	}

	for _, opt := range options {
		opt(&conf)
	}

	return &conf
}

func (auth *SchemaAuthorizer) IsAuthorizedDetail(ctx context.Context, request *Request) (*Detail, error) {
	req := engine.Request{
		Principal: request.Principal,
		Action:    request.Action,
		Resource:  request.Resource,
		Context:   request.Context,
		Store:     auth.Store,
		Trace:     false,
	}

	result, err := engine.Eval(ctx, auth.Policies, &req)

	if err != nil {
		return nil, err
	}
	return &Detail{
		IsAllowed: result.Decision == engine.Allow,
		Matches:   result.Reasons,
	}, nil
}

func (auth *SchemaAuthorizer) IsAuthorized(ctx context.Context, request *Request) (bool, error) {
	detail, err := auth.IsAuthorizedDetail(ctx, request)
	if err != nil {
		return false, err
	}

	return detail.IsAllowed, nil
}
