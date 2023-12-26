package cst

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/koblas/cedar-go/engine"
	"github.com/koblas/cedar-go/token"
)

type astBuilder interface {
	ToAst(file *token.File) (engine.EvalNode, error)
}

var ErrInternal = errors.New("internal consistency error")
var ErrExpectFile = errors.New("expected engine.File input")

// Helper to unpack a node
func toEvalNode(file *token.File, node any, msg string) (engine.EvalNode, error) {
	builder, ok := node.(astBuilder)
	if !ok || builder == nil {
		return nil, fmt.Errorf("invalid %s type %T: %w", msg, node, ErrInternal)
	}
	return builder.ToAst(file)
}

func isHexDigit(ch rune) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func hexValue(b rune) int {
	c := rune(b)
	switch {
	case '0' <= c && c <= '9':
		return int(c - '0')
	case 'a' <= c && c <= 'f':
		return int(c - 'a' + 10)
	case 'A' <= c && c <= 'F':
		return int(c - 'A' + 10)
	}
	return 0
}

// Unquote a quoted string
func unquote(str string) string {
	// short circuts
	if str[0] != '"' || str[len(str)-1] != '"' {
		return str
	}
	inQuote := false
	for _, ch := range str {
		if ch == '\\' {
			inQuote = true
			break
		}
	}
	if !inQuote {
		return str[1 : len(str)-1]
	}

	// Hard work
	news := []rune{}
	inQuote = false

	index := 0
	next := func() {
		index += 1
	}
	runeStr := []rune(str[1 : len(str)-1])

	for ; index < len(runeStr); next() {
		ch := runeStr[index]
		if !inQuote {
			if ch == '\\' {
				inQuote = true
			} else {
				news = append(news, ch)
			}
		} else {
			inQuote = false
			switch ch {
			case '\\':
				news = append(news, '\\')
			case '0':
				news = append(news, '\000')
			case '\'':
				news = append(news, '\'')
			case 't':
				news = append(news, '\t')
			case '"':
				news = append(news, '"')
			case 'r':
				news = append(news, '\r')
			case 'n':
				news = append(news, '\n')
			case 'u':
				next() // skip 'u'
				if index == len(runeStr) {
					news = append(news, 'u')
					continue
				}
				if runeStr[index] != '{' {
					news = append(news, 'u')
					news = append(news, runeStr[index])
					continue
				}
				data := []rune{}
				value := 0
				for next(); index < len(runeStr) && isHexDigit(runeStr[index]); next() {
					data = append(data, runeStr[index])
					value = value<<4 + hexValue(runeStr[index])
				}
				if index >= len(runeStr) || runeStr[index] != '}' {
					// invalid end, append everything
					news = append(news, '\\', 'u', '{')
					news = append(news, data...)
					if index < len(runeStr) {
						news = append(news, runeStr[index])
					}
				} else {
					news = append(news, rune(value))
				}
			}
		}
	}

	return string(news)
}

var (
	trueValue = &engine.ValueNode{
		Value: engine.BoolValue(true),
	}
	falseValue = &engine.ValueNode{
		Value: engine.BoolValue(false),
	}
)

func (n *BasicLit) ToAst(file *token.File) (engine.EvalNode, error) {
	switch n.Kind {
	case token.STRINGLIT:
		return &engine.ValueNode{
			Value: engine.StrValue(unquote(n.Value)),
		}, nil
	case token.INT:
		value, err := strconv.Atoi(n.Value)
		if err != nil {
			return nil, err
		}
		return &engine.ValueNode{
			Value: engine.IntValue(value),
		}, nil
	case token.TRUE:
		return trueValue, nil
	case token.FALSE:
		return falseValue, nil

	// This needs to do a variable lookup from the runtime context
	case token.PRINCIPAL:
		return &engine.Reference{
			StartPos: file.Position(n.Pos()),
			Source:   engine.RunVarPrincipal,
		}, nil
	case token.ACTION:
		return &engine.Reference{
			StartPos: file.Position(n.Pos()),
			Source:   engine.RunVarAction,
		}, nil
	case token.RESOURCE:
		return &engine.Reference{
			StartPos: file.Position(n.Pos()),
			Source:   engine.RunVarResource,
		}, nil
	case token.CONTEXT:
		return &engine.Reference{
			StartPos: file.Position(n.Pos()),
			Source:   engine.RunVarContext,
		}, nil

	case token.IDENTIFER:
		return &engine.Identifier{
			StartPos: file.Position(n.Pos()),
			Value:    n.Value,
		}, nil
	}

	panic(fmt.Sprintf("%s: invalid literal type %v", file.Position(n.Pos()), n.Kind.String()))
}

func (n *EntityName) ToAst(file *token.File) (engine.EvalNode, error) {
	l := len(n.Path) - 1
	var parts []string
	for _, item := range n.Path[0:l] {
		parts = append(parts, item.Value)
	}

	parts = append(parts, unquote(n.Path[l].Value))

	value := engine.EntityValue(parts)

	return &engine.ValueNode{
		Value: value,
	}, nil
}

func (n *UnaryExpr) ToAst(file *token.File) (engine.EvalNode, error) {
	left, err := toEvalNode(file, n.X, "left")
	if err != nil {
		return nil, err
	}

	var opcode engine.Operand
	switch n.Op {
	case token.SUB:
		opcode = engine.OpSub
	case token.NOT:
		opcode = engine.OpNot
	default:
		return nil, fmt.Errorf("unimplemented unary opcode %s: %w", n.Op.String(), ErrInternal)
	}

	return &engine.UnaryExpr{
		StartPos: file.Position(n.Pos()),
		Op:       opcode,
		Left:     left,
	}, nil
}

func (n *BinaryExpr) ToAst(file *token.File) (engine.EvalNode, error) {
	left, err := toEvalNode(file, n.X, "left")
	if err != nil {
		return nil, err
	}
	right, err := toEvalNode(file, n.Y, "right")
	if err != nil {
		return nil, err
	}

	var opcode engine.Operand
	switch n.Op {
	case token.EQL:
		opcode = engine.OpEql
	case token.LSS:
		opcode = engine.OpLss
	case token.GTR:
		opcode = engine.OpGtr
	case token.NEQ:
		opcode = engine.OpNeq
	case token.LEQ:
		opcode = engine.OpLeq
	case token.GEQ:
		opcode = engine.OpGeq

	case token.LAND:
		opcode = engine.OpLand
	case token.LOR:
		opcode = engine.OpLor
	case token.NOT:
		opcode = engine.OpNot

	case token.ADD:
		opcode = engine.OpAdd
	case token.SUB:
		opcode = engine.OpSub
	case token.MUL:
		opcode = engine.OpMul
	case token.QUO:
		opcode = engine.OpQuo
	case token.REM:
		opcode = engine.OpRem

	case token.IN:
		opcode = engine.OpIn
	case token.LIKE:
		opcode = engine.OpLike
	case token.HAS:
		opcode = engine.OpHas

	default:
		return nil, fmt.Errorf("unimplemented binary opcode %s: %w", n.Op.String(), ErrInternal)
	}

	return &engine.BinaryExpr{
		StartPos: file.Position(n.Pos()),
		Op:       opcode,
		Left:     left,
		Right:    right,
	}, nil
}

func (n *ParenExpr) ToAst(file *token.File) (engine.EvalNode, error) {
	return toEvalNode(file, n.X, "left")
}

func (n *IfExpr) ToAst(file *token.File) (engine.EvalNode, error) {
	ifExpr, err := toEvalNode(file, n.Condition, "if")
	if err != nil {
		return nil, err
	}
	thenExpr, err := toEvalNode(file, n.Then, "then")
	if err != nil {
		return nil, err
	}
	elseExpr, err := toEvalNode(file, n.Else, "else")
	if err != nil {
		return nil, err
	}
	return &engine.IfExpr{
		StartPos: file.Position(n.Pos()),
		If:       ifExpr,
		Then:     thenExpr,
		Else:     elseExpr,
	}, nil
}

func (n *ReceiverInits) ToAst(file *token.File) (engine.EvalNode, error) {
	var variables []engine.VariablePair

	for _, item := range n.Exprs {
		var key string
		if item.Literal.Kind == token.STRINGLIT {
			key = unquote(item.Literal.Value)
		} else if item.Literal.Kind == token.IDENTIFER {
			key = item.Literal.Value
		}

		value, err := toEvalNode(file, item.Expr, "value")
		if err != nil {
			return nil, err
		}
		pair := engine.VariablePair{
			Key:   key,
			Value: value,
		}
		variables = append(variables, pair)
	}

	return &engine.VariableDef{
		StartPos: file.Position(n.Pos()),
		Pairs:    variables,
	}, nil
}

func (n *SetExpr) ToAst(file *token.File) (engine.EvalNode, error) {
	exprs := []engine.EvalNode{}
	for _, item := range n.Exprs {
		expr, err := toEvalNode(file, item, "set")
		if err != nil {
			return nil, err
		}

		exprs = append(exprs, expr)
	}
	return &engine.ListExpr{
		StartPos: file.Position(n.Pos()),
		AsSet:    true,
		Exprs:    exprs,
	}, nil
}

func (n *MemberExpr) ToAst(file *token.File) (engine.EvalNode, error) {
	left, err := toEvalNode(file, n.Primary, "member")
	if err != nil {
		return nil, err
	}

	for _, item := range n.Access {
		lit, err := item.Ident.ToAst(file)
		if err != nil {
			return nil, fmt.Errorf("MemberExpr: invalid condition type %T: %w", n.Primary, ErrInternal)
		}

		if item.IsFunc {
			var args []engine.EvalNode

			for _, arg := range item.Args {
				expr, err := toEvalNode(file, arg, "any")
				if err != nil {
					return nil, err
				}
				args = append(args, expr)
			}

			left = &engine.FunctionCall{
				StartPos: file.Position(n.Pos()),
				Name:     item.Ident.Value,
				Self:     left,
				Args:     args,
			}
		} else {
			left = &engine.BinaryExpr{
				StartPos: file.Position(n.Pos()),
				Op:       engine.OpLookup,
				Left:     left,
				Right:    lit,
			}
		}
	}

	return left, nil
}

func (n *FunctionCall) ToAst(file *token.File) (engine.EvalNode, error) {
	var args []engine.EvalNode

	for _, arg := range n.Args {
		expr, err := toEvalNode(file, arg, "any")
		if err != nil {
			return nil, err
		}
		args = append(args, expr)
	}

	// Handle function call
	return &engine.FunctionCall{
		StartPos: file.Position(n.Pos()),
		Name:     n.Name,
		Self:     nil,
		Args:     args,
	}, nil
}

func (n *Condition) ToAst(file *token.File) (*engine.PolicyCondition, error) {
	var condition engine.Condition
	switch n.Condition {
	case token.WHEN:
		condition = engine.ConditionWhen
	case token.UNLESS:
		condition = engine.ConditionUnless
	default:
		return nil, fmt.Errorf("Condition: invalid condition type %s: %w", n.Condition.String(), ErrInternal)
	}

	aexpr, err := toEvalNode(file, n.Expr, "condition")
	if err != nil {
		return nil, err
	}

	return &engine.PolicyCondition{
		Condition: condition,
		Expr:      aexpr,
	}, nil
}

func (n *Variable) ToAst(file *token.File) (engine.EvalNode, error) {
	source, err := n.NameLit.ToAst(file)
	if err != nil {
		return nil, err
	}

	opcode := engine.OpInvalid
	switch n.RelOp {
	case token.EQL:
		opcode = engine.OpEql
	case token.IN:
		opcode = engine.OpIn
	case token.ILLEGAL:
		// nothing
	default:
		return nil, fmt.Errorf("unimplemented unary opcode in variables %s: %w", n.RelOp.String(), ErrInternal)
	}

	var expr engine.EvalNode
	if opcode != engine.OpInvalid {
		var right engine.EvalNode
		if n.SetExpr != nil {
			right, err = n.SetExpr.ToAst(file)
			if err != nil {
				return nil, err
			}
		} else {
			right, err = n.Entities[0].ToAst(file)
			if err != nil {
				return nil, err
			}
		}
		if err != nil {
			return nil, err
		}
		expr = &engine.BinaryExpr{
			StartPos: file.Position(n.Pos()),
			Op:       opcode,
			Left:     source,
			Right:    right,
		}
	} else {
		expr = trueValue
	}

	// Reformulate the "is" check into an if statement
	if n.IsCheck != nil {
		rval, err := n.IsCheck.ToAst(file)
		if err != nil {
			return nil, err
		}
		isExpr := &engine.BinaryExpr{
			Op:    engine.OpIs,
			Left:  source,
			Right: rval,
		}

		expr = &engine.IfExpr{
			If:   isExpr,
			Then: expr,
			Else: falseValue,
		}
	}

	return expr, nil
}

func (n *PolicyStmt) ToAst(file *token.File) (*engine.Policy, error) {
	var annotations map[string]string

	if len(n.Annotations) != 0 {
		annotations = make(map[string]string)

		for _, item := range n.Annotations {
			annotations[item.Ident.Value] = item.Value.Value
		}
	}

	var conditions []*engine.PolicyCondition
	for _, item := range n.Conditions {
		value, err := item.ToAst(file)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, value)
	}

	var effect engine.PolicyEffect
	switch n.Effect {
	case token.PERMIT:
		effect = engine.EffectPermit
	case token.FORBID:
		effect = engine.EffectForbid
	}

	p, err := n.Scope.Principal.ToAst(file)
	if err != nil {
		return nil, err
	}
	a, err := n.Scope.Action.ToAst(file)
	if err != nil {
		return nil, err
	}
	r, err := n.Scope.Resource.ToAst(file)
	if err != nil {
		return nil, err
	}

	var ifExpr engine.EvalNode
	for _, c := range []engine.EvalNode{p, r, a} {
		if c == trueValue {
			continue
		}
		if ifExpr == nil {
			ifExpr = c
			continue
		}
		ifExpr = &engine.BinaryExpr{
			Op:    engine.OpLand,
			Left:  c,
			Right: ifExpr,
		}
	}
	if ifExpr == nil {
		ifExpr = trueValue
	}

	return &engine.Policy{
		StartPos:    file.Position(n.Pos()),
		Effect:      effect,
		If:          ifExpr,
		Conditions:  conditions,
		Annotations: annotations,
	}, nil
}

func (n *File) ToAst(file *token.File) (engine.PolicyList, error) {
	result := engine.PolicyList{}

	for idx, item := range n.Statements {
		if b, ok := item.(*PolicyStmt); ok {
			value, err := b.ToAst(file)
			if err != nil {
				return nil, err
			}
			if id, found := value.Annotations["id"]; found {
				value.Id = id
			} else {
				value.Id = fmt.Sprintf("policy%d", idx)
			}
			result = append(result, value)
		}
	}

	return result, nil
}

func ToAst(file *token.File, node Node) (engine.PolicyList, error) {
	b, ok := node.(*File)
	if !ok {
		return nil, ErrExpectFile
	}

	return b.ToAst(file)
}
