package cedar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/koblas/cedar-go/engine"
	"github.com/koblas/cedar-go/parser"
	"github.com/koblas/cedar-go/schema"
)

// Request is used to setup per-request variables to the authorization engine
type Request struct {
	Principal engine.EntityValue
	Action    engine.EntityValue
	Resource  engine.EntityValue
	Context   *engine.VarValue
}

// AuthDetail provides additional information about the authorized evaluation.
type AuthDetail struct {
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
	trace    bool
}

type EmptyStore struct{}

// StoreFromJson create a store object based on the "standard" entity store
// as defined in the Cedar specification
func StoreFromJson(reader io.Reader, sdef *schema.Schema) (engine.Store, error) {
	if sdef == nil {
		sdef = schema.NewEmptySchema()
	}

	entities := schema.JsonEntities{}
	err := json.NewDecoder(reader).Decode(&entities)
	if err != nil {
		panic(fmt.Errorf("unable to decode entities: %w", err))
	}

	return sdef.NormalizeEntites(entities)
}

// Option handles conditional options to the auth engine
type Option func(*SchemaAuthorizer)

// WithSchema add a schema definition to the engine this
// is used to parse the input Context information
func WithSchema(s *schema.Schema) Option {
	return func(sa *SchemaAuthorizer) {
		sa.Schema = s
	}
}

// WithStore add an interface to external data storage
// either with JSON entities or a custom storage
func WithStore(s engine.Store) Option {
	return func(sa *SchemaAuthorizer) {
		sa.Store = s
	}
}

// WithTracing enables basic evaluation tracing, useful for
// development purposes
func WithTracing() Option {
	return func(sa *SchemaAuthorizer) {
		sa.trace = true
	}
}

// NewAuthorizer constructs a authorization engine with pre-parsed
// rules and options
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

// IsAuthorizedDetail provides additional detail from the evaluation engine about why the
// result was formed. The `IsAuthorized“ is the perfered method that validation engines
// should use
func (auth *SchemaAuthorizer) IsAuthorizedDetail(ctx context.Context, request *Request) (*AuthDetail, error) {
	req := engine.Request{
		Principal: request.Principal,
		Action:    request.Action,
		Resource:  request.Resource,
		Context:   request.Context,
		Store:     auth.Store,
		Trace:     auth.trace,
	}

	result, err := engine.Eval(ctx, auth.Policies, &req)

	if err != nil {
		return nil, err
	}
	return &AuthDetail{
		IsAllowed: result.Decision == engine.Allow,
		Matches:   result.Reasons,
	}, nil
}

// IsAuthorized is the primary entry point that services should use to evaluate based on the
// pre-loaded rules and store information.
func (auth *SchemaAuthorizer) IsAuthorized(ctx context.Context, request *Request) (bool, error) {
	detail, err := auth.IsAuthorizedDetail(ctx, request)
	if err != nil {
		return false, err
	}

	return detail.IsAllowed, nil
}

// NewEntity constructs an engine.EntityValue object based on the kind (e.g. User or Action)
// and the id (e.g. "alice" or "view")
func NewEntity(kind, id string) engine.EntityValue {
	return engine.NewEntityValue(kind, id)
}

// ParsePolicies will parse the policy definition and return a runtime
// evaluation engine for the data.
func ParsePolicies(policies string) (engine.PolicyList, error) {
	return parser.ParseRules(policies)
}
