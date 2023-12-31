package engine

import (
	"context"
	"errors"
	"fmt"
)

type EvalValue interface {
	NamedType
}

type policyResult struct {
	Decision     Decision
	Evaluated    bool
	Permit       bool
	Forbid       bool
	RulesMatched []string
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
	//
	principalSlot NamedType
	resourceSlot  NamedType

	//
	functionTable map[string]Function

	// Debugging
	Trace  bool
	indent int
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

func printTrace(indent int, format string, args ...any) {
	const dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
	const n = len(dots)
	i := 2 * indent
	for i > n {
		fmt.Print(dots)
		i -= n
	}
	// i <= n
	fmt.Print(dots[0:i])
	fmt.Printf(format, args...)
	fmt.Print("\n")
}

func trace(r *RuntimeRequest, format string, args ...any) *RuntimeRequest {
	printTrace(r.indent, format+"%s", append(args, "(")...)
	r.indent += 1
	return r
}

func un(r *RuntimeRequest) {
	r.indent -= 1
	printTrace(r.indent, ")")
}

//
//

func (n *ValueNode) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		printTrace(request.indent, "%s[%s]", n.Value.TypeName(), n.Value)
	}
	return n.Value, nil
}

func (n *UnaryExpr) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		defer un(trace(request, "UnaryExpr[%s]", n.Op.String()))
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

func (n *BinaryExpr) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		defer un(trace(request, "BinaryExpr[%s]", n.Op.String()))
	}
	left, err := n.Left.evalNode(request)
	if err != nil {
		return nil, err
	}
	var right EvalValue
	if n.Op != OpLand && n.Op != OpLor {
		// Delay evaluation for logical operations
		r, err := n.Right.evalNode(request)
		if err != nil {
			return nil, err
		}
		right = r
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
			msg := fmt.Sprintf("type error: not supported %s %s",
				left.TypeName(), n.Op.String())
			return nil, evalError(n, msg)
		}

		// Short circuit these
		lval, err := asBool(n, left)
		if err != nil {
			return nil, err
		}
		if n.Op == OpLor && lval {
			return left, nil
		}
		if n.Op == OpAdd && !lval {
			return left, nil
		}
		// Now process the right hand side
		right, err := n.Right.evalNode(request)
		if err != nil {
			return nil, err
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
		ltype, ok := left.(VariableType)
		if !ok {
			msg := fmt.Sprintf("type error: lookup not supported %s", left.TypeName())
			return nil, evalError(n, msg)
		}

		return ltype.OpHas(right, request.Store)

	case OpLookup:
		ltype, ok := left.(VariableType)
		if !ok {
			msg := fmt.Sprintf("type error: lookup not supported %s", left.TypeName())
			return nil, evalError(n, msg)
		}

		return ltype.OpLookup(right, request.Store)
	}

	return nil, evalError(n, fmt.Sprintf("Unexpected binary op %s", n.Op.String()))
}

func (n *IfExpr) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		defer un(trace(request, "IfExpr"))
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
		defer un(trace(request, "If%s", branch))
	}

	return expr.evalNode(request)
}

func (n *FunctionCall) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		defer un(trace(request, "Function[%s]", n.Name))
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
		defer un(trace(request, "ListExpr"))
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
		printTrace(request.indent, "ReferenceExpr[%s]", n.Source.String())
	}
	if n.Source == RunVarContext {
		// TODO - shouldn't happen
		return request.Context, nil
	}

	switch n.Source {
	case RunVarPrincipal:
		return request.principalValue, nil
	case RunVarAction:
		return request.actionValue, nil
	case RunVarResource:
		return request.resourceValue, nil
	case RunVarSlotPrincipal:
		return request.principalSlot, nil
	case RunVarSlotResource:
		return request.resourceSlot, nil
	}

	return nil, fmt.Errorf("not implemented")
}

func (n *VariableDef) evalNode(request *RuntimeRequest) (EvalValue, error) {
	data := map[string]NamedType{}

	for _, item := range n.Pairs {
		value, err := item.Value.evalNode(request)
		if err != nil {
			return nil, err
		}

		data[item.Key] = value
	}

	return NewVarValue(data), nil
}

func (n *Identifier) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		printTrace(request.indent, "Identifier[%s]", n.Value)
	}
	// This feels like a hack
	return IdentifierValue(n.Value), nil
}

func (n *PolicyCondition) evalNode(request *RuntimeRequest) (EvalValue, error) {
	if request.Trace {
		defer un(trace(request, "PolicyCondition[%s]", n.Condition.String()))
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

func (n *Policy) evalNode(request *RuntimeRequest) (*policyResult, error) {
	if request.Trace {
		defer un(trace(request, "Policy[id=%s, type=%s]", n.Id, n.Effect.String()))
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
		// if request.Trace {
		// 	fmt.Printf("PolicyCondition(%s) = %v", item.Condition.String(), boolValue)
		// }
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
		defer un(trace(request, "PolicyList"))
	}
	allowed := false
	forbid := false

	var matches []string
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

		if res.Forbid || res.Permit {
			matches = append(matches, item.Id)
		}
	}

	var err error
	if elist != nil {
		err = errors.Join(elist...)
	}

	// Must be explcitly allowed
	if allowed && !forbid {
		return &policyResult{
			Decision:     Allow,
			Permit:       true,
			RulesMatched: matches,
		}, err
	}

	return &policyResult{
		Decision:     Deny,
		Forbid:       true,
		RulesMatched: matches,
	}, err
}
