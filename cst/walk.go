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
	case (*Comment):
		// nothing
	case (*BadStmt):
		// nothing
	// case (*AnnotationSpec): // NOT a Node
	case (*PolicyStmt):
		for _, item := range n.Conditions {
			Walk(visitor, item)
		}

	case (*BadExpr):
	case (*ScopeNew):
	case (*Variable):
		if n.SetExpr != nil {
			Walk(visitor, n.SetExpr)
		}
	case (*BasicLit): // not a expr
	case (*UnaryExpr):
		Walk(visitor, n.X)
	case (*BinaryExpr):
		Walk(visitor, n.X)
		Walk(visitor, n.Y)
	case (*IfExpr):
		Walk(visitor, n.Condition)
		Walk(visitor, n.Then)
		Walk(visitor, n.Else)
	case (*ParenExpr):
		Walk(visitor, n.X)
	case (*SetExpr):
		for _, item := range n.Exprs {
			Walk(visitor, item)
		}
	case (*MemberAccess):
		if n.Ident != nil {
			Walk(visitor, n.Ident)
		}
		if n.Ident != nil {
			Walk(visitor, n.Index)
		}
		if len(n.Args) != 0 {
			for _, arg := range n.Args {
				Walk(visitor, arg)
			}
		}
	case (*MemberExpr):
		Walk(visitor, n.Primary)
		for _, item := range n.Access {
			Walk(visitor, item)
		}
	case (*ReceiverInit):
		Walk(visitor, n.Literal)
		Walk(visitor, n.Expr)
	case (*ReceiverInits):
		for _, item := range n.Exprs {
			Walk(visitor, item.Expr)
		}

	case (*EntityName): // only contain Literals
	case (*Path): // only contain Literals

	case (*FunctionCall):
		for _, item := range n.Args {
			Walk(visitor, item)
		}
	case (*File):
		for _, item := range n.Comments {
			Walk(visitor, item)
		}
		for _, item := range n.Statements {
			Walk(visitor, item)
		}
	default:
		panic(fmt.Sprintf("engine.Walk: unexpected node type %T", n))
	}
}
