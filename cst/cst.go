// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cst

import (
	"strings"

	"github.com/koblas/cedar-go/core/token"
)

// ----------------------------------------------------------------------------
// Interfaces
//
// There are 3 main classes of nodes: Expressions and type nodes,
// statement nodes, and declaration nodes. The node names usually
// match the corresponding Go spec production names to which they
// correspond. The node fields correspond to the individual parts
// of the respective productions.
//
// All nodes contain position information marking the beginning of
// the corresponding source text segment; it is accessible via the
// Pos accessor method. Nodes may contain additional position info
// for language constructs where comments may be found between parts
// of the construct (typically any larger, parenthesized subpart).
// That position information is needed to properly position comments
// when printing the construct.

// All node types implement the Node interface.
type Node interface {
	Pos() token.Pos // position of first character belonging to the node
	End() token.Pos // position of first character immediately after the node
}

// All expression nodes implement the Expr interface.
type Expr interface {
	Node
	exprNode()
}

// All statement nodes implement the Stmt interface.
type Stmt interface {
	Node
	stmtNode()
}

type Decl interface {
	Node
	declNode()
}

// ----------------------------------------------------------------------------
// Comments

// A Comment node represents a single //-style or /*-style comment.
//
// The Text field contains the comment text without carriage returns (\r) that
// may have been present in the source. Because a comment's end position is
// computed using len(Text), the position reported by End() does not match the
// true source end position for comments containing carriage returns.
type Comment struct {
	Slash token.Pos // position of "/" starting the comment
	Text  string    // comment text (excluding '\n' for //-style comments)
}

func (c *Comment) Pos() token.Pos { return c.Slash }
func (c *Comment) End() token.Pos { return token.Pos(int(c.Slash) + len(c.Text)) }

// A CommentGroup represents a sequence of comments
// with no other tokens and no empty lines between.
type CommentGroup struct {
	List []*Comment // len(List) > 0
}

func (g *CommentGroup) Pos() token.Pos { return g.List[0].Pos() }
func (g *CommentGroup) End() token.Pos { return g.List[len(g.List)-1].End() }

func isWhitespace(ch byte) bool { return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' }

func stripTrailingWhitespace(s string) string {
	i := len(s)
	for i > 0 && isWhitespace(s[i-1]) {
		i--
	}
	return s[0:i]
}

// Text returns the text of the comment.
// Comment markers (//, /*, and */), the first space of a line comment, and
// leading and trailing empty lines are removed.
// Comment directives like "//line" and "//go:noinline" are also removed.
// Multiple empty lines are reduced to one, and trailing space on lines is trimmed.
// Unless the result is empty, it is newline-terminated.
func (g *CommentGroup) Text() string {
	if g == nil {
		return ""
	}
	comments := make([]string, len(g.List))
	for i, c := range g.List {
		comments[i] = c.Text
	}

	lines := make([]string, 0, 10) // most comments are less than 10 lines
	for _, c := range comments {
		// Remove comment markers.
		// The parser has given us exactly the comment text.
		switch c[1] {
		case '/':
			//-style comment (no newline at the end)
			c = c[2:]
			if len(c) != 0 && c[0] == ' ' {
				// strip first space - required for Example tests
				c = c[1:]
			}
		case '*':
			/*-style comment */
			c = c[2 : len(c)-2]
		}

		// Split on newlines.
		cl := strings.Split(c, "\n")

		// Walk lines, stripping trailing white space and adding to list.
		for _, l := range cl {
			lines = append(lines, stripTrailingWhitespace(l))
		}
	}

	// Remove leading blank lines; convert runs of
	// interior blank lines to a single blank line.
	n := 0
	for _, line := range lines {
		if line != "" || n > 0 && lines[n-1] != "" {
			lines[n] = line
			n++
		}
	}
	lines = lines[0:n]

	// Add final "" entry to get trailing newline from Join.
	if n > 0 && lines[n-1] != "" {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// ----------------------------------------------------------------------------
// Expressions and types
type (
	// A BadExpr node is a placeholder for an expression containing
	// syntax errors for which a correct expression node cannot be
	// created.
	//
	BadStmt struct {
		From, To token.Pos // position range of bad expression
	}

	AnnotationSpec struct {
		TokPos token.Pos // position of Tok
		Ident  *BasicLit
		Lparen token.Pos // position of '(', if any
		Value  BasicLit
		Rparen token.Pos // position of ')', if any
	}

	PolicyStmt struct {
		From, To    token.Pos
		Annotations []*AnnotationSpec
		Effect      token.Token
		Scope       *ScopeNew
		Conditions  []*Condition
	}
)

func (d *BadStmt) Pos() token.Pos { return d.From }
func (d *BadStmt) End() token.Pos { return d.To }

func (d *PolicyStmt) Pos() token.Pos { return d.From }
func (d *PolicyStmt) End() token.Pos { return d.To }

// declNode() ensures that only declaration nodes can be
// assigned to a Decl.
func (*PolicyStmt) declNode() {}
func (*BadStmt) declNode()    {}

// ----------------------------------
type (
	// A BadExpr node is a placeholder for an expression containing
	// syntax errors for which a correct expression node cannot be
	// created.
	//
	BadExpr struct {
		From, To token.Pos // position range of bad expression
	}

	ScopeNew struct {
		Lparen    token.Pos
		Principal Variable
		Comma1    token.Pos
		Action    Variable
		Comma2    token.Pos
		Resource  Variable
		Rparen    token.Pos
	}

	Variable struct {
		NameLit  BasicLit
		IsPos    token.Pos
		IsCheck  *EntityName
		RelPos   token.Pos
		RelOp    token.Token
		Slot     token.Token
		Entities []*EntityName
		SetExpr  *SetExpr // action in [ EntList ]
		PosEnd   token.Pos
	}

	BasicLit struct {
		ValuePos token.Pos   // literal position
		Kind     token.Token // token.INT, token.STRINGLIT, token.IDENTIFIER, token.TRUE, token.FALSE
		Value    string      //
	}

	Condition struct {
		ConditionPos token.Pos
		Condition    token.Token
		Lbrace       token.Pos
		Expr         Expr
		Rbrace       token.Pos
	}

	// A UnaryExpr node represents a unary expression.
	// Unary "*" expressions are represented via StarExpr nodes.
	//
	UnaryExpr struct {
		OpPos token.Pos   // position of Op
		Op    token.Token // operator
		X     Expr        // operand
	}

	// A BinaryExpr node represents a binary expression.
	BinaryExpr struct {
		X     Expr        // left operand
		OpPos token.Pos   // position of Op
		Op    token.Token // operator
		Y     Expr        // right operand
	}

	IfExpr struct {
		IfPos     token.Pos // position of Op
		Condition Expr      // conditonal expression
		Then      Expr      // then expression
		Else      Expr      // else expression
	}

	// A ParenExpr node represents a parenthesized expression.
	ParenExpr struct {
		Lparen token.Pos // position of "("
		X      Expr      // parenthesized expression
		Rparen token.Pos // position of ")"
	}

	// A SetExpr node represents a square bracked list of expressions
	SetExpr struct {
		Lbrack token.Pos // position of "["
		Exprs  []Expr    // parenthesized expression
		Rbrack token.Pos // position of "]"
	}

	// Ident struct {
	// 	NamePos token.Pos // identifier position
	// 	Name    string    // identifier name
	// 	// Obj     *Object   // denoted object; or nil
	// }

	MemberAccess struct {
		Ident     *BasicLit
		IsFunc    bool
		IsRef     bool
		LparenPos token.Pos
		Args      []Expr
		Index     *BasicLit
		RparenPos token.Pos
		//
	}

	MemberExpr struct {
		Primary Expr
		Access  []*MemberAccess
	}

	ReceiverInit struct {
		Literal BasicLit
		Expr    Expr
	}

	ReceiverInits struct {
		Lbrace token.Pos // position of "{"
		Exprs  []ReceiverInit
		Rbrace token.Pos // position of "}"
	}

	EntityName struct {
		Path []BasicLit
	}

	Path struct {
		Path []BasicLit
	}

	FunctionCall struct {
		Name string
		// Raw
		Ref    *Path
		Lparen token.Pos // position of "("
		Args   []Expr
		Rparen token.Pos // position of ")"
	}
)

func (x *BadExpr) Pos() token.Pos { return x.From }
func (x *BadExpr) End() token.Pos { return x.From }

func (x *BasicLit) Pos() token.Pos { return x.ValuePos }
func (x *BasicLit) End() token.Pos { return token.Pos(int(x.ValuePos) + len(x.Value)) }

func (x *UnaryExpr) Pos() token.Pos { return x.OpPos }
func (x *UnaryExpr) End() token.Pos { return x.X.End() }

func (x *BinaryExpr) Pos() token.Pos { return x.X.Pos() }
func (x *BinaryExpr) End() token.Pos { return x.Y.End() }

func (x *Variable) Pos() token.Pos { return x.NameLit.Pos() }
func (x *Variable) End() token.Pos {
	if x.Slot != 0 {
		return x.PosEnd
	}
	if len(x.Entities) == 0 {
		return x.RelPos
	}
	return x.Entities[len(x.Entities)-1].End()
}

func (x *MemberExpr) Pos() token.Pos { return x.Primary.Pos() }
func (x *MemberExpr) End() token.Pos {
	if len(x.Access) == 0 {
		return x.Primary.End()
	}
	return x.Access[len(x.Access)-1].RparenPos
}

func (x *IfExpr) Pos() token.Pos { return x.IfPos }
func (x *IfExpr) End() token.Pos { return x.Else.End() }

func (x *ParenExpr) Pos() token.Pos { return x.Lparen }
func (x *ParenExpr) End() token.Pos { return x.Rparen + 1 }

func (x *SetExpr) Pos() token.Pos { return x.Lbrack }
func (x *SetExpr) End() token.Pos { return x.Rbrack + 1 }

func (x *ReceiverInits) Pos() token.Pos { return x.Lbrace }
func (x *ReceiverInits) End() token.Pos { return x.Rbrace + 1 }

func (x *EntityName) Pos() token.Pos { return x.Path[0].Pos() }
func (x *EntityName) End() token.Pos { return x.Path[len(x.Path)-1].Pos() }

func (x *Path) Pos() token.Pos { return x.Path[0].Pos() }
func (x *Path) End() token.Pos { return x.Path[len(x.Path)-1].Pos() }

func (x *FunctionCall) Pos() token.Pos { return x.Ref.Pos() }
func (x *FunctionCall) End() token.Pos {
	if len(x.Args) != 0 {
		return x.Args[len(x.Args)-1].End()
	}
	return x.Ref.End()
}

func (x *Condition) Pos() token.Pos { return x.ConditionPos }
func (x *Condition) End() token.Pos { return x.Rbrace }

// exprNode() ensures that only expression/type nodes can be
// assigned to an Expr.
func (*Variable) exprNode()      {}
func (*BasicLit) exprNode()      {}
func (*BadExpr) exprNode()       {}
func (*UnaryExpr) exprNode()     {}
func (*BinaryExpr) exprNode()    {}
func (*MemberExpr) exprNode()    {}
func (*IfExpr) exprNode()        {}
func (*ParenExpr) exprNode()     {}
func (*SetExpr) exprNode()       {}
func (*ReceiverInits) exprNode() {}
func (*EntityName) exprNode()    {}
func (*Path) exprNode()          {}
func (*FunctionCall) exprNode()  {}
func (*Condition) exprNode()     {}

// ----------------------------------------------------------------------------
// Convenience functions for Idents

// NewIdent creates a new Ident without position.
// Useful for ASTs generated by code other than the Go parser.

// IsExported reports whether name starts with an upper-case letter.
func IsExported(name string) bool { return token.IsExported(name) }

// ----------------------------------------------------------------------------
// Files and packages
//
// The Comments list contains all comments in the source file in order of
// appearance, including the comments that are pointed to from other nodes
// via Doc and Comment fields.
//
// For correct printing of source code containing comments (using packages
// go/format and go/printer), special care must be taken to update comments
// when a File's syntax tree is modified: For printing, comments are interspersed
// between tokens based on their position. If syntax tree nodes are
// removed or moved, relevant comments in their vicinity must also be removed
// (from the File.Comments list) or moved accordingly (by updating their
// positions). A CommentMap may be used to facilitate some of these operations.
//
// Whether and how a comment is associated with a node depends on the
// interpretation of the syntax tree by the manipulating program: Except for Doc
// and Comment comments directly associated with nodes, the remaining comments
// are "free-floating" (see also issues #18593, #20744).

type File struct {
	Statements []Decl          // top-level statements; or nil
	Comments   []*CommentGroup // list of all comments in the source file
}

func (f *File) Pos() token.Pos {
	if f.Statements != nil && len(f.Statements) != 0 {
		return f.Statements[0].Pos()
	}
	return token.Pos(0)
}
func (f *File) End() token.Pos {
	if n := len(f.Statements); n > 0 {
		return f.Statements[n-1].End()
	}
	return token.Pos(0)
}
