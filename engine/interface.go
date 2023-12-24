package engine

import (
	"context"
	"strings"
)

type EntityRef struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type Request struct {
	// Good ole golang
	Ctx context.Context
	// Scope properties
	Store   Store
	Context *VarValue
	// request properties
	Principal EntityValue
	Resource  EntityValue
	Action    EntityValue
	Trace     bool // print debugging
}

type Decision int

const (
	Deny  Decision = iota
	Allow Decision = iota
)

func (d Decision) String() string {
	if d == Allow {
		return "Allow"
	}
	return "Deny"
}

type Result struct {
	Decision     Decision
	RulesMatched bool
	Reasons      []string
}

func (e EntityRef) ToValue() EntityValue {
	parts := strings.Split(e.Type, "::")
	parts = append(parts, e.Id)

	return EntityValue(parts)
}

func Eval(ctx context.Context, p PolicyList, request *Request) (*Result, error) {
	runtime := RuntimeRequest{
		Ctx:            ctx,
		Store:          request.Store,
		Context:        request.Context,
		principalValue: request.Principal,
		resourceValue:  request.Resource,
		actionValue:    request.Action,
		functionTable:  functionTable,
		Trace:          request.Trace,
	}

	result, err := p.evalNode(&runtime)
	if err != nil {
		return nil, err
	}

	decision := Deny
	if result.Permit {
		decision = Allow
	}

	return &Result{
		Decision:     decision,
		RulesMatched: result.Evaluated,
		Reasons:      result.RulesMatched,
	}, nil
}
