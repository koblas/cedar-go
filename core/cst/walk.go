package cst

import "fmt"

type Visitor interface {
	Visit(node Node) (w Visitor)
}

func Walk(visitor Visitor, node Node) {
	if visitor = visitor.Visit(node); visitor == nil {
		return
	}

	switch n := node.(type) {
	case (*BadExpr):
	case *Comment:
		// Exprs
	case (*BasicLit): // not a expr
	case (*EntityName): // only contain Literals
	case (*Path): // only contain Literals

	case (*Variable):
		if n.SetExpr != nil {
			Walk(visitor, n.SetExpr)
		}
	case (*UnaryExpr):
		Walk(visitor, n.X)
	case (*BinaryExpr):
		Walk(visitor, n.X)
		Walk(visitor, n.Y)
	case (*MemberExpr):
		Walk(visitor, n.Primary)
	case (*ParenExpr):
		Walk(visitor, n.X)

	case (*IfExpr):
		Walk(visitor, n.Condition)
		Walk(visitor, n.Then)
		Walk(visitor, n.Else)
	case (*SetExpr):
		for _, item := range n.Exprs {
			Walk(visitor, item)
		}
	case (*ReceiverInits):
		for _, item := range n.Exprs {
			Walk(visitor, item.Expr)
		}
	case (*FunctionCall):
		for _, item := range n.Args {
			Walk(visitor, item)
		}
	case (*PolicyStmt):
		//
		for _, item := range n.Conditions {
			Walk(visitor, item)
		}
	case (*File):
		for _, item := range n.Comments {
			Walk(visitor, item)
		}
		for _, item := range n.Statements {
			Walk(visitor, item)
		}
		//
		// Ignore these nodes, shouldn't happen
	default:
		panic(fmt.Sprintf("ast.Walk: unexpected node type %T", n))
	}
}
