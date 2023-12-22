package ast

import (
	"context"
	"errors"
	"fmt"
)

type EvalValue interface {
	NamedType
}

type policyResult struct {
	Decision  Decision
	Evaluated bool
	Permit    bool
	Forbid    bool
}

type RuntimeRequest struct {
	// Good ole golang
	Ctx context.Context
	// Scope properties
	Store   Store
	Context *VarValue

	// computed values
	principalValue EntityValue
	resourceValue  EntityValue
	actionValue    EntityValue

	//
	functionTable map[string]Function

	// Debugging
	Trace bool
}

type EvalNode interface {
	evalNode(*RuntimeRequest) (EvalValue, error)
}

var ErrEvalError = errors.New("eval error")
var ErrTypeError = errors.New("type error")

func evalError(n ExprNode, msg string) error {
	return fmt.Errorf("%s: %s: %w", n.Pos().String(), msg, ErrEvalError)
}

func asBool(n ExprNode, v EvalValue) (bool, error) {
	value, ok := v.(BoolValue)
	if !ok {
		return false, fmt.Errorf("%s: expected bool got %s: %w", n.Pos().String(), v.TypeName(), ErrEvalError)
	}

	return bool(value), nil
}

//
//

func (n *ValueNode) evalNode(*RuntimeRequest) (EvalValue, error) {
	return n.Value, nil
}

func (n *UnaryExpr) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		fmt.Printf("UnaryExpr(%s)\n", n.Op.String())
	}
	result, err := n.Left.evalNode(request)
	if err != nil {
		return nil, err
	}
	switch n.Op {
	case OpSub:
		ltype, ok := result.(MathType)
		if !ok {
			return nil, evalError(n, "LHS does not support logic ops")
		}

		return ltype.OpUnaryMinus()
	case OpNot:
		ltype, ok := result.(LogicType)
		if !ok {
			return nil, evalError(n, "LHS does not support logic ops")
		}

		return ltype.OpNot()
	}
	return nil, fmt.Errorf("unknown unary op: %s", n.StartPos)
}

func (n *BinaryExpr) evalEntityOrStore(left NamedType, right NamedType, request *RuntimeRequest) (NamedType, error) {
	ltype, ok := left.(*VarValue)
	if ok {
		return ltype, nil
	}
	etype, ok := left.(EntityValue)
	if !ok {
		msg := fmt.Sprintf("type error: not supported %s %s %s",
			left.TypeName(), n.Op.String(), right.TypeName())
		return nil, evalError(n, msg)
	}

	val, err := request.Store.Get(etype)
	if errors.Is(err, ErrValueNotFound) {
		val = &VarValue{}
	}
	return val, nil
}

func (n *BinaryExpr) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		fmt.Printf("BinaryExpr(%s)\n", n.Op.String())
	}
	left, err := n.Left.evalNode(request)
	if err != nil {
		return nil, err
	}
	right, err := n.Right.evalNode(request)
	if err != nil {
		return nil, err
	}

	switch n.Op {
	case OpEql, OpNeq:
		r, err := left.OpEqual(right)
		if err != nil {
			return nil, err
		}
		if n.Op != OpNeq {
			return r, nil
		}
		return BoolValue(!r), nil
	case OpAdd, OpSub, OpMul, OpRem, OpQuo:
		ltype, ok := left.(MathType)
		if !ok {
			msg := fmt.Sprintf("type error: not supported %s %s %s",
				left.TypeName(), n.Op.String(), right.TypeName())
			return nil, evalError(n, msg)
		}
		switch n.Op {
		case OpAdd:
			return ltype.OpAdd(right)
		case OpSub:
			return ltype.OpSub(right)
		case OpMul:
			return ltype.OpMul(right)
		case OpQuo:
			return ltype.OpQuo(right)
		case OpRem:
			return ltype.OpRem(right)
		}
	// Logic operations
	case OpLss:
		ltype, ok := left.(ComparisonType)
		if !ok {
			msg := fmt.Sprintf("type error: not supported %s %s %s",
				left.TypeName(), n.Op.String(), right.TypeName())
			return nil, evalError(n, msg)
		}
		return ltype.OpLss(right)
	case OpLeq:
		ltype, ok := left.(ComparisonType)
		if !ok {
			msg := fmt.Sprintf("type error: not supported %s %s %s",
				left.TypeName(), n.Op.String(), right.TypeName())
			return nil, evalError(n, msg)
		}
		return ltype.OpLeq(right)
	case OpGeq:
		// Swap left and right
		rtype, ok := right.(ComparisonType)
		if !ok {
			msg := fmt.Sprintf("type error: not supported %s %s %s",
				left.TypeName(), n.Op.String(), right.TypeName())
			return nil, evalError(n, msg)
		}
		return rtype.OpLeq(left)
	case OpGtr:
		// Swap left and right
		rtype, ok := right.(ComparisonType)
		if !ok {
			msg := fmt.Sprintf("type error: not supported %s %s %s",
				left.TypeName(), n.Op.String(), right.TypeName())
			return nil, evalError(n, msg)
		}
		return rtype.OpLss(left)
	// Boolean operators
	case OpLand, OpLor:
		ltype, ok := left.(LogicType)
		if !ok {
			msg := fmt.Sprintf("type error: not supported %s %s %s",
				left.TypeName(), n.Op.String(), right.TypeName())
			return nil, evalError(n, msg)
		}

		if n.Op == OpLand {
			return ltype.OpLand(right)
		}
		return ltype.OpLor(right)

	case OpIs:
		ltype, ok := left.(IsType)
		if !ok {
			msg := fmt.Sprintf("type error: not supported %s %s %s",
				left.TypeName(), n.Op.String(), right.TypeName())
			return nil, evalError(n, msg)
		}

		return ltype.OpIs(right)

	case OpIn:
		ltype, ok := left.(InType)
		if !ok {
			msg := fmt.Sprintf("type error: not supported %s %s %s",
				left.TypeName(), n.Op.String(), right.TypeName())
			return nil, evalError(n, msg)
		}

		return ltype.OpIn(right, request.Store)

	case OpLike:
		ltype, ok := left.(LikeType)
		if !ok {
			msg := fmt.Sprintf("type error: not supported %s %s %s",
				left.TypeName(), n.Op.String(), right.TypeName())
			return nil, evalError(n, msg)
		}

		return ltype.OpLike(right)

	case OpHas:
		ltype, err := n.evalEntityOrStore(left, right, request)
		if err != nil {
			return nil, err
		}
		vtype, ok := ltype.(VariableType)
		if !ok {
			msg := fmt.Sprintf("type error: not supported %s %s %s",
				left.TypeName(), n.Op.String(), right.TypeName())
			return nil, evalError(n, msg)
		}

		return vtype.OpHas(right)

	case OpLookup:
		ltype, err := n.evalEntityOrStore(left, right, request)
		if err != nil {
			return nil, err
		}
		vtype, ok := ltype.(VariableType)
		if !ok {
			msg := fmt.Sprintf("type error: not supported %s %s %s",
				left.TypeName(), n.Op.String(), right.TypeName())
			return nil, evalError(n, msg)
		}
		return vtype.OpLookup(right)
	}

	return nil, evalError(n, fmt.Sprintf("Unexpected binary op %s", n.Op.String()))
}

func (n *IfExpr) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		fmt.Printf("IfExpr()\n")
	}
	cond, err := n.If.evalNode(request)
	if err != nil {
		return nil, err
	}
	value, err := asBool(n, cond)
	if err != nil {
		return nil, err
	}
	expr := n.Then
	branch := "Then"
	if !value {
		branch = "Else"
		expr = n.Else
	}
	if request.Trace {
		fmt.Printf("If%s()\n", branch)
	}

	return expr.evalNode(request)
}

func (n *FunctionCall) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		fmt.Printf("Function(%s[...])\n", n.Name)
	}

	var left EvalValue
	if n.Self != nil {
		lval, err := n.Self.evalNode(request)
		if err != nil {
			return nil, err
		}
		left = lval
	}

	var args []EvalValue
	for _, arg := range n.Args {
		val, err := arg.evalNode(request)
		if err != nil {
			return nil, err
		}
		args = append(args, val)
	}

	handler, found := request.functionTable[n.Name]
	if !found {
		return nil, fmt.Errorf("function named %s not found: %w", n.Name, ErrEvalError)
	}
	result, err := handler(left, args)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (n *ListExpr) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		fmt.Printf("ListExpr(__)\n")
	}
	values := []NamedType{}
	for _, item := range n.Exprs {
		val, err := item.evalNode(request)
		if err != nil {
			return nil, err
		}
		values = append(values, val)
	}

	return SetValue(values), nil
}

func (n *Reference) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		fmt.Printf("ReferenceExpr(%s)\n", n.Source.String())
	}
	if n.Source == PrincipalContext {
		// TODO - shouldn't happen
		return request.Context, nil
	}

	switch n.Source {
	case PrincipalPrincipal:
		return request.principalValue, nil
	case PrincipalAction:
		return request.actionValue, nil
	case PrincipalResource:
		return request.resourceValue, nil
	}

	return nil, fmt.Errorf("not implemented")
}

func (n *Identifier) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		fmt.Printf("Identifier(%s)\n", n.Value)
	}
	// This feels like a hack
	return IdentifierValue(n.Value), nil
}

func (n *PolicyCondition) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		fmt.Printf("  PolicyCondition(%s)\n", n.Condition.String())
	}
	result, err := n.Expr.evalNode(request)
	if err != nil {
		return nil, err
	}
	boolValue, err := asBool(n, result)
	if err != nil {
		return nil, err
	}

	if n.Condition == ConditionWhen {
		return BoolValue(boolValue), nil
	}
	// Condition unless
	return BoolValue(!boolValue), nil
}

/*
func (n *PolicyVariable) evalNode(request *RuntimeRequest) (BoolValue, error) {
	if request.Trace {
		fmt.Printf("PolicyVariable(%s)\n", n.Var.String())
	}
	// Invalid == "All"
	if n.Op == OpInvalid {
		return BoolValue(true), nil
	}

	var lhs EntityValue
	switch n.Var {
	case PrincipalPrincipal:
		lhs = request.principalValue
	case PrincipalResource:
		lhs = request.resourceValue
	case PrincipalAction:
		lhs = request.actionValue
	}

	if n.Op == OpEql {
		return lhs.OpEqual(n.Entity)
	}

	if n.Entities != nil {
		return lhs.OpIn(n.Entities)
	}
	return lhs.OpIn(n.Entity)
}
*/

func (n *Policy) evalNode(request *RuntimeRequest) (*policyResult, error) {
	if request.Trace {
		fmt.Printf("Policy(id=%s, type=%s)\n", n.Id, n.Effect.String())
	}

	if r, err := n.If.evalNode(request); err != nil {
		return nil, err
	} else if v, err := asBool(n, r); err != nil {
		return nil, err
	} else if !v {
		return &policyResult{}, nil
	}

	evalResult := true
	for _, item := range n.Conditions {
		result, err := item.evalNode(request)
		if err != nil {
			return nil, err
		}
		boolValue, err := asBool(n, result)
		if err != nil {
			return nil, err
		}
		evalResult = evalResult && boolValue
		if request.Trace {
			fmt.Printf("  PolicyCondition(%s) = %v\n", item.Condition.String(), boolValue)
		}
		// Early exit
		if !evalResult {
			break
		}
	}

	forbid := false
	permit := false
	if evalResult {
		if n.Effect == EffectForbid {
			forbid = true
		} else if n.Effect == EffectPermit {
			permit = true
		}
	}

	return &policyResult{
		Evaluated: true,
		Forbid:    forbid,
		Permit:    permit,
	}, nil
}

func (p PolicyList) evalNode(request *RuntimeRequest) (*policyResult, error) {
	if request.Trace {
		fmt.Printf("PolicyList()\n")
	}
	allowed := false
	forbid := false

	var elist []error
	for _, item := range p {
		res, err := item.evalNode(request)
		if err != nil {
			elist = append(elist, err)
			continue
		}
		if !res.Evaluated {
			continue
		}
		// fmt.Println(res)

		forbid = forbid || res.Forbid
		allowed = allowed || res.Permit
	}

	var err error
	if elist != nil {
		err = errors.Join(elist...)
	}

	// Must be explcitly allowed
	if allowed && !forbid {
		return &policyResult{
			Decision: Allow,
			Permit:   true,
		}, err
	}

	return &policyResult{
		Decision: Deny,
		Forbid:   true,
	}, err
}
