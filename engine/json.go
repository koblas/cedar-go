package engine

import (
	"encoding/json"
	"errors"
	"fmt"
)

var ErrInvalidJsonNode = errors.New("node doesn't support json")

type jsonBuilder interface {
	ToJson() any
}

// General types
type JsonEntityType struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}
type JsonEntitiesType []JsonEntityType

// The policy header
type JsonVariable struct {
	Op       string           `json:"op"`
	Entity   *JsonEntityType  `json:"entity,omitempty"`
	Entities JsonEntitiesType `json:"entities,omitempty"`
	Slot     string           `json:"slot,omitempty"`
}

// Now onto the conditions
type JsonBinary struct {
	Left  JsonExpr `json:"left"`
	Right JsonExpr `json:"right"`
}

type JsonIfThenElse struct {
	If   JsonExpr `json:"if"`
	Then JsonExpr `json:"then"`
	Else JsonExpr `json:"else"`
}

// Simpler
type JsonExpr map[string]any

type JsonCondition struct {
	Kind string   `json:"kind"`
	Body JsonExpr `json:"body"`
}

type JsonPolicy struct {
	Effect      string            `json:"effect"`
	Principal   *JsonVariable     `json:"principal"`
	Action      *JsonVariable     `json:"action"`
	Resource    *JsonVariable     `json:"resource"`
	Conditions  []*JsonCondition  `json:"conditions,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// / -------
func (n *ValueNode) ToJson() any {
	if value, ok := n.Value.(BoolValue); ok {
		return value
	}
	if value, ok := n.Value.(StrValue); ok {
		return value
	}
	if value, ok := n.Value.(IntValue); ok {
		return value
	}
	if value, ok := n.Value.(EntityValue); ok {
		return value.String()
	}
	if _, ok := n.Value.(SetValue); ok {
		// values := []any{}
		panic("not implemented")
		// for _, item := range value {
		// 	values = append(values, item.ToJson())
		// }
		// return values
	}

	panic(fmt.Sprintf("invalid result type %v", n))
}

func (n *EntityRef) ToJson() *JsonEntityType {
	return &JsonEntityType{
		Type: n.Type,
		Id:   n.Id,
	}
}

func (n *UnaryExpr) ToJson() any {
	arg, _ := n.Left.(jsonBuilder)
	if arg == nil {
		panic(fmt.Sprintf("left expr no ToJson %T", n.Left))
	}
	return &JsonExpr{
		n.Op.String(): map[string]any{
			"arg": arg.ToJson(),
		},
	}
}

func (n *BinaryExpr) ToJson() any {
	left, _ := n.Left.(jsonBuilder)
	right, _ := n.Right.(jsonBuilder)
	if left == nil {
		panic(fmt.Sprintf("left expr no ToJson %T", n.Left))
	}
	if right == nil {
		panic(fmt.Sprintf("right expr no ToJson %T", n.Right))
	}
	return &JsonExpr{
		n.Op.String(): map[string]any{
			"left":  left.ToJson(),
			"right": right.ToJson(),
		},
	}
}

func (n *PolicyCondition) ToJson() *JsonCondition {
	var expr JsonExpr
	if b, ok := n.Expr.(jsonBuilder); ok {
		value := b.ToJson()
		if value, ok := value.(map[string]any); ok {
			expr = value
		} else {
			panic(fmt.Sprintf("invalid ToJson return %v", n.Expr))
		}
	} else {
		panic(fmt.Sprintf("Missing ToJson %v", n.Expr))
	}

	return &JsonCondition{
		Kind: n.Condition.String(),
		Body: expr,
	}
}

// func (n *PolicyStmt) ToJson() *JsonPolicy {
func (n *Policy) ToJson() any {
	var conditions []*JsonCondition
	for _, item := range n.Conditions {
		conditions = append(conditions, item.ToJson())
	}

	return &JsonPolicy{
		Effect:      n.Effect.String(),
		Conditions:  conditions,
		Annotations: n.Annotations,
	}
}

func (n PolicyList) ToJson() any {
	result := []any{}

	for _, item := range n {
		result = append(result, item.ToJson())
	}

	return result
}

/// ------

func ToJson(policies PolicyList) ([]byte, error) {
	data := policies.ToJson()

	return json.Marshal(data)
}
