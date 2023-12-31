package engine

import (
	"github.com/koblas/cedar-go/token"
)

// -----------------------------------------
// Enum types
type PolicyEffect int

const (
	EffectInvalid PolicyEffect = iota
	EffectPermit  PolicyEffect = iota
	EffectForbid  PolicyEffect = iota
)

func (v PolicyEffect) String() string {
	return [...]string{"invalid", "permit", "forbid"}[v]
}

type Condition int

const (
	ConditionInvalid Condition = iota
	ConditionWhen    Condition = iota
	ConditionUnless  Condition = iota
)

func (v Condition) String() string {
	return [...]string{"invalid", "when", "unless"}[v]
}

// RunVar is an enum for runtime variable sources
type RunVar int

const (
	RunVarInvalid       RunVar = iota
	RunVarResource      RunVar = iota
	RunVarAction        RunVar = iota
	RunVarPrincipal     RunVar = iota
	RunVarContext       RunVar = iota
	RunVarSlotPrincipal RunVar = iota
	RunVarSlotResource  RunVar = iota
)

var runVarStrings = [...]string{
	RunVarInvalid:       "invalid",
	RunVarResource:      "resource",
	RunVarAction:        "action",
	RunVarPrincipal:     "principal",
	RunVarContext:       "context",
	RunVarSlotPrincipal: "?principal",
	RunVarSlotResource:  "?resource",
}

func (v RunVar) String() string {
	return runVarStrings[v]
}

type Operand int

const (
	OpInvalid Operand = iota

	// Comparisons
	OpEql Operand = iota
	OpLss Operand = iota
	OpGtr Operand = iota
	OpNeq Operand = iota
	OpLeq Operand = iota
	OpGeq Operand = iota

	// Boolean
	OpLand Operand = iota
	OpLor  Operand = iota
	OpNot  Operand = iota

	// Math
	OpAdd Operand = iota
	OpSub Operand = iota
	OpMul Operand = iota
	OpQuo Operand = iota
	OpRem Operand = iota

	// Sets
	OpIn   Operand = iota
	OpLike Operand = iota

	// Entities
	OpIs Operand = iota

	// Variables
	OpHas    Operand = iota
	OpLookup Operand = iota
)

func (v Operand) String() string {
	return [...]string{
		OpInvalid: "[invalid]",
		// Comparisons
		OpEql: "==",
		OpLss: "<",
		OpGtr: ">",
		OpNeq: "!=",
		OpLeq: "<=",
		OpGeq: ">=",
		// Boolean
		OpLand: "&&",
		OpLor:  "||",
		OpNot:  "!",
		// Math
		OpAdd: "+",
		OpSub: "-",
		OpMul: "*",
		OpQuo: "/",
		OpRem: "%",
		// Entities
		OpIs: "is",
		// Sets
		OpIn:   "in",
		OpLike: "like",
		// Variable
		OpHas:    "has",
		OpLookup: "_lookup",
	}[v]
}

// -----------------------------------------
// Interfaces
type (
	Node interface {
		Pos() token.Position // position of first character belonging to the node - for error traces
	}

	ExprNode interface {
		Node
		exprNode()
	}
)

// -----------------------------------------
// Nodes
type (
	ValueNode struct {
		Value EvalValue
	}

	// This can either be a SET or an argument LIST
	// Parsing has already dis-ambiguated this
	ListExpr struct {
		StartPos token.Position
		AsSet    bool
		Exprs    []EvalNode
	}

	UnaryExpr struct {
		StartPos token.Position
		Op       Operand
		Left     EvalNode
	}

	BinaryExpr struct {
		StartPos token.Position
		Op       Operand
		Left     EvalNode
		Right    EvalNode
	}

	Reference struct {
		StartPos token.Position
		Source   RunVar
	}

	// Used in `has` expression
	Identifier struct {
		StartPos token.Position
		Value    string
	}

	FunctionCall struct {
		StartPos token.Position
		Name     string
		Self     EvalNode
		Args     []EvalNode
	}

	IfExpr struct {
		StartPos token.Position
		If       EvalNode
		Then     EvalNode
		Else     EvalNode
	}

	// not an expr
	VariablePair struct {
		Key   string
		Value EvalNode
	}
	VariableDef struct {
		StartPos token.Position
		Pairs    []VariablePair
	}

	PolicyCondition struct {
		StartPos  token.Position
		Condition Condition
		Expr      EvalNode
	}

	Policy struct {
		StartPos    token.Position
		Id          string // Unique identifier
		Effect      PolicyEffect
		If          EvalNode
		Conditions  []*PolicyCondition
		Annotations map[string]string
	}

	PolicyList []*Policy
)

// Things that can be evaluated
func (*BinaryExpr) exprNode()      {}
func (*FunctionCall) exprNode()    {}
func (*Identifier) exprNode()      {}
func (*IfExpr) exprNode()          {}
func (*ListExpr) exprNode()        {}
func (*Policy) exprNode()          {}
func (*PolicyCondition) exprNode() {}
func (*Reference) exprNode()       {}
func (*UnaryExpr) exprNode()       {}
func (*VariableDef) exprNode()     {}

func (n *BinaryExpr) Pos() token.Position      { return n.StartPos }
func (n *FunctionCall) Pos() token.Position    { return n.StartPos }
func (n *Identifier) Pos() token.Position      { return n.StartPos }
func (n *IfExpr) Pos() token.Position          { return n.StartPos }
func (n *ListExpr) Pos() token.Position        { return n.StartPos }
func (n *Policy) Pos() token.Position          { return n.StartPos }
func (n *PolicyCondition) Pos() token.Position { return n.StartPos }
func (n *Reference) Pos() token.Position       { return n.StartPos }
func (n *UnaryExpr) Pos() token.Position       { return n.StartPos }
func (n *VariableDef) Pos() token.Position     { return n.StartPos }
