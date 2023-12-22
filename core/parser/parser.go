package parser

import (
	"fmt"

	"github.com/koblas/cedar-go/core/cst"
	"github.com/koblas/cedar-go/core/scanner"
	"github.com/koblas/cedar-go/core/token"
)

// The parser structure holds the parser's internal state.
type parser struct {
	file    *token.File
	errors  scanner.ErrorList
	scanner scanner.Scanner

	// Tracing/debugging
	mode   Mode // parsing mode
	trace  bool // == (mode & Trace != 0)
	indent int  // indentation used for tracing output

	// Comments
	comments    []*cst.CommentGroup
	leadComment *cst.CommentGroup // last lead comment
	lineComment *cst.CommentGroup // last line comment

	// Next token
	pos token.Pos   // token position
	tok token.Token // one token look-ahead
	lit string      // token literal

	// Error recovery
	// (used to limit the number of calls to parser.advance
	// w/o making scanning progress - avoids potential endless
	// loops across multiple parser functions during error recovery)
	syncPos token.Pos // last synchronization position
	syncCnt int       // number of parser.advance calls without progress

	// Non-syntactic parser control
	exprLev int  // < 0: in control clause, >= 0: in expression
	inRhs   bool // if set, the parser is parsing a rhs expression

	// Ordinary identifier scopes
	pkgScope *cst.Scope // pkgScope.Outer == nil
	topScope *cst.Scope // top-most scope; may be pkgScope
}

func (p *parser) init(fset *token.FileSet, filename string, src []byte, mode Mode) {
	p.file = fset.AddFile(filename, -1, len(src))
	var m scanner.Mode
	if mode&ParseComments != 0 {
		m = scanner.ScanComments
	}
	eh := func(pos token.Position, msg string) { p.errors.Add(pos, msg) }
	p.scanner.Init(p.file, src, eh, m)

	p.mode = mode
	p.trace = mode&Trace != 0 // for convenience (p.trace is used frequently)

	p.next()
}

// ----------------------------------------------------------------------------
// Parsing support

func (p *parser) printTrace(a ...interface{}) {
	const dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
	const n = len(dots)
	pos := p.file.Position(p.pos)
	fmt.Printf("%5d:%3d: ", pos.Line, pos.Column)
	i := 2 * p.indent
	for i > n {
		fmt.Print(dots)
		i -= n
	}
	// i <= n
	fmt.Print(dots[0:i])
	fmt.Println(a...)
}

func trace(p *parser, msg string) *parser {
	p.printTrace(msg, "(")
	p.indent++
	return p
}

// Usage pattern: defer un(trace(p, "..."))
func un(p *parser) {
	p.indent--
	p.printTrace(")")
}

// Advance to the next token.
func (p *parser) next0() {
	// Because of one-token look-ahead, print the previous token
	// when tracing as it provides a more readable output. The
	// very first token (!p.pos.IsValid()) is not initialized
	// (it is token.ILLEGAL), so don't print it .
	if p.trace && p.pos.IsValid() {
		s := p.tok.String()
		switch {
		case p.tok.IsLiteral():
			p.printTrace(s, p.lit)
		case p.tok.IsOperator(), p.tok.IsKeyword():
			p.printTrace("\"" + s + "\"")
		default:
			p.printTrace(s)
		}
	}

	p.pos, p.tok, p.lit = p.scanner.Scan()
}

// Consume a comment and return it and the line on which it ends.
func (p *parser) consumeComment() (comment *cst.Comment, endline int) {
	// /*-style comments may end on a different line than where they start.
	// Scan the comment for '\n' chars and adjust endline accordingly.
	endline = p.file.Line(p.pos)
	if p.lit[1] == '*' {
		// don't use range here - no need to decode Unicode code points
		for i := 0; i < len(p.lit); i++ {
			if p.lit[i] == '\n' {
				endline++
			}
		}
	}

	comment = &cst.Comment{Slash: p.pos, Text: p.lit}
	p.next0()

	return
}

// Consume a group of adjacent comments, add it to the parser's
// comments list, and return it together with the line at which
// the last comment in the group ends. A non-comment token or n
// empty lines terminate a comment group.
func (p *parser) consumeCommentGroup(n int) (comments *cst.CommentGroup, endline int) {
	var list []*cst.Comment
	endline = p.file.Line(p.pos)
	for p.tok == token.COMMENT && p.file.Line(p.pos) <= endline+n {
		var comment *cst.Comment
		comment, endline = p.consumeComment()
		list = append(list, comment)
	}

	// add comment group to the comments list
	comments = &cst.CommentGroup{List: list}
	p.comments = append(p.comments, comments)

	return
}

// Advance to the next non-comment token. In the process, collect
// any comment groups encountered, and remember the last lead and
// line comments.
//
// A lead comment is a comment group that starts and ends in a
// line without any other tokens and that is followed by a non-comment
// token on the line immediately after the comment group.
//
// A line comment is a comment group that follows a non-comment
// token on the same line, and that has no tokens after it on the line
// where it ends.
//
// Lead and line comments may be considered documentation that is
// stored in the cST.
func (p *parser) next() {
	p.leadComment = nil
	p.lineComment = nil
	prev := p.pos
	p.next0()

	if p.tok == token.COMMENT {
		var comment *cst.CommentGroup
		var endline int

		if p.file.Line(p.pos) == p.file.Line(prev) {
			// The comment is on same line as the previous token; it
			// cannot be a lead comment but may be a line comment.
			comment, endline = p.consumeCommentGroup(0)
			if p.file.Line(p.pos) != endline || p.tok == token.EOF {
				// The next token is on a different line, thus
				// the last comment group is a line comment.
				p.lineComment = comment
			}
		}

		// consume successor comments, if any
		endline = -1
		for p.tok == token.COMMENT {
			comment, endline = p.consumeCommentGroup(1)
		}

		if endline+1 == p.file.Line(p.pos) {
			// The next token is following on the line immediately after the
			// comment group, thus the last comment group is a lead comment.
			p.leadComment = comment
		}
	}
}

// A bailout panic is raised to indicate early termination.
type bailout struct{}

func (p *parser) error(pos token.Pos, msg string) {
	epos := p.file.Position(pos)

	// If AllErrors is not set, discard errors reported on the same line
	// as the last recorded error and stop parsing if there are more than
	// 10 errors.
	if p.mode&AllErrors == 0 {
		n := len(p.errors)
		if n > 0 && p.errors[n-1].Pos.Line == epos.Line {
			return // discard - likely a spurious error
		}
		if n > 10 {
			panic(bailout{})
		}
	}

	p.errors.Add(epos, msg)
}

func (p *parser) errorExpected(pos token.Pos, msg string) {
	msg = "expected " + msg
	if pos == p.pos {
		// the error happened at the current position;
		// make the error message more specific
		switch {
		case p.tok == token.SEMICOLON && p.lit == "\n":
			msg += ", found newline"
		case p.tok.IsLiteral():
			// print 123 rather than 'INT', etc.
			msg += ", found " + p.lit
		default:
			msg += ", found '" + p.tok.String() + "'"
		}
	}
	p.error(pos, msg)
}

func (p *parser) expect(tok token.Token) token.Pos {
	pos := p.pos
	if p.tok != tok {
		p.errorExpected(pos, "'"+tok.String()+"'")
	}
	p.next() // make progress
	return pos
}

func (p *parser) atComma(context string, follow token.Token) bool {
	if p.tok == token.COMMA {
		return true
	}
	if p.tok != follow {
		msg := "missing ','"
		if p.tok == token.SEMICOLON && p.lit == "\n" {
			msg += " before newline"
		}
		p.error(p.pos, msg+" in "+context)
		return true // "insert" comma and continue
	}
	return false
}

func assert(cond bool, msg string) {
	if !cond {
		panic("go/parser internal error: " + msg)
	}
}

// advance consumes tokens until the current token p.tok
// is in the 'to' set, or token.EOF. For error recovery.
func (p *parser) advance(to map[token.Token]bool) {
	for ; p.tok != token.EOF; p.next() {
		if to[p.tok] {
			// Return only if parser made some progress since last
			// sync or if it has not reached 10 advance calls without
			// progress. Otherwise consume at least one token to
			// avoid an endless parser loop (it is possible that
			// both parseOperand and parseStmt call advance and
			// correctly do not advance, thus the need for the
			// invocation limit p.syncCnt).
			if p.pos == p.syncPos && p.syncCnt < 10 {
				p.syncCnt++
				return
			}
			if p.pos > p.syncPos {
				p.syncPos = p.pos
				p.syncCnt = 0
				return
			}
			// Reaching here indicates a parser bug, likely an
			// incorrect token list in this function, but it only
			// leads to skipping of possibly correct code if a
			// previous error is present, and thus is preferred
			// over a non-terminating parse.
		}
	}
}

// var stmtStart = map[tok}

// ----------------------------------------------------------------------------
// RecInits ::= (IDENT | STR) ':' Expr {',' (IDENT | STR) ':' Expr}
func (p *parser) parseReceiverInits() []cst.ReceiverInit {
	if p.trace {
		defer un(trace(p, "RecInits"))
	}

	var result []cst.ReceiverInit

	for p.tok == token.STRINGLIT || p.tok == token.IDENTIFER {
		lit := cst.BasicLit{ValuePos: p.pos, Kind: p.tok, Value: p.lit}
		p.next()
		p.expect(token.COLON)
		expr := p.parseExpr()

		node := cst.ReceiverInit{
			Literal: lit,
			Expr:    expr,
		}

		result = append(result, node)

		if p.tok == token.COMMA {
			// Consume the comma
			p.next()
		}
	}

	return result
}

// ----------------------------------------------------------------------------
// ExprList ::= Expr {',' Expr}
func (p *parser) parseExprList(endTok token.Token) ([]cst.Expr, token.Pos) {
	if p.trace {
		defer un(trace(p, "ExprList"))
	}

	if p.tok == endTok {
		return nil, p.expect(endTok)
	}

	exprs := []cst.Expr{p.parseExpr()}
	for p.tok == token.COMMA {
		p.next()

		exprs = append(exprs, p.parseExpr())
	}

	return exprs, p.expect(endTok)
}

// ----------------------------------------------------------------------------
// EntList ::= Entity {',' Entity}
func (p *parser) parseEntList() []*cst.EntityName {
	var entities []*cst.EntityName
	entity := p.parseEntity()
	if entity == nil {
		return entities
	}
	entities = append(entities, entity)
	for p.tok == token.COMMA {
		p.next()

		entity := p.parseEntity()
		if entity != nil {
			entities = append(entities, entity)
		}
	}

	return entities

}

// ----------------------------------------------------------------------------
// Path ::= IDENT {'::' IDENT}
// Entity ::= Path '::' STR
// ExtFun ::= [Path '::'] IDENT
func (p *parser) parseEntityOrPath(onlyEntity bool, onlyPath bool) cst.Expr {
	if onlyEntity && onlyPath {
		panic("Invalid call both onlyEntity and onlyPath specified")
	}
	if p.trace {
		defer un(trace(p, "ParseEntityOrFunc"))
	}
	var path []cst.BasicLit
	if p.tok != token.IDENTIFER {
		if p.tok != token.IDENTIFER {
			p.errorExpected(p.pos, "'"+token.IDENTIFER.String()+"'")
		}
		return nil
	}
	lit := cst.BasicLit{ValuePos: p.pos, Kind: p.tok, Value: p.lit}
	path = append(path, lit)
	p.next()

	// follow the path
	for p.tok == token.PATH {
		p.next()
		if p.tok == token.IDENTIFER {
			lit := cst.BasicLit{ValuePos: p.pos, Kind: p.tok, Value: p.lit}
			path = append(path, lit)
			p.next()
			continue
		} else {
			break
		}
	}

	if p.tok == token.STRINGLIT {
		if onlyPath {
			p.error(p.pos, "unexpected string literal in PATH")
			return nil
		}

		lit := cst.BasicLit{ValuePos: p.pos, Kind: p.tok, Value: p.lit}
		p.next()
		path = append(path, lit)

		return &cst.EntityName{
			Path: path,
		}
	} else if onlyEntity {
		p.error(p.pos, "Expected string literal")
		return nil
	}

	// must be a ExtFunc
	return &cst.Path{
		Path: path,
	}
}

func (p *parser) parseEntity() *cst.EntityName {
	result := p.parseEntityOrPath(true, false)

	if result == nil {
		return nil
	}
	if val, ok := result.(*cst.EntityName); ok {
		return val
	}
	return nil
}

// ----------------------------------------------------------------------------
//
// Primary ::= LITERAL
//
//	| VAR
//	| Entity
//	| ExtFun '(' [ExprList] ')'
//	| '(' Expr ')'
//	| '[' [ExprList] ']'
//	| '{' [RecInits] '}'
func (p *parser) parsePrimary() cst.Expr {
	if p.trace {
		defer un(trace(p, "Primary"))
	}
	tok := p.tok

	switch tok {
	// Handle LITERAL
	case token.TRUE, token.FALSE, token.STRINGLIT, token.INT:
		lit := cst.BasicLit{ValuePos: p.pos, Kind: tok, Value: p.lit}
		p.next()
		return &lit
	// Handle VAR
	case token.PRINCIPAL, token.ACTION, token.RESOURCE, token.CONTEXT:
		lit := cst.BasicLit{ValuePos: p.pos, Kind: tok, Value: p.lit}
		p.next()
		return &lit
	case token.LPAREN:
		//	'(' [Expr] ')'
		pos := p.expect(token.LPAREN)
		x := p.parseExpr()
		end := p.expect(token.RPAREN)
		return &cst.ParenExpr{
			Lparen: pos,
			X:      x,
			Rparen: end,
		}
	case token.LBRACK:
		//	'[' [ExprList] ']'
		pos := p.expect(token.LBRACK)
		exprs, end := p.parseExprList(token.RBRACK)
		return &cst.SetExpr{
			Lbrack: pos,
			Exprs:  exprs,
			Rbrack: end,
		}
	case token.LBRACE:
		//	'{' [RecInits] '}'
		pos := p.expect(token.LBRACE)
		exprs := p.parseReceiverInits()
		end := p.expect(token.RBRACE)
		return &cst.ReceiverInits{
			Lbrace: pos,
			Exprs:  exprs,
			Rbrace: end,
		}
	}

	entity := p.parseEntityOrPath(false, false)

	if ref, ok := entity.(*cst.EntityName); ok {
		// Entity
		return ref
	} else if ref, ok := entity.(*cst.Path); ok {
		// ExtFun '(' [ExprList]')'
		lpos := p.expect(token.LPAREN)
		exprs, rpos := p.parseExprList(token.RPAREN)

		funcName := ref.Path[len(ref.Path)-1].Value

		return &cst.FunctionCall{
			Name:   funcName,
			Ref:    ref,
			Lparen: lpos,
			Args:   exprs,
			Rparen: rpos,
		}
	}

	return nil
}

// ----------------------------------------------------------------------------
// Access ::= '.' IDENT ['(' [ExprList] ')'] | '[' STR ']'
func (p *parser) parseAccess() *cst.MemberAccess {
	if p.trace {
		defer un(trace(p, "Access"))
	}

	ident := p.parseIdent()
	lparenPos := p.pos
	rparenPos := p.pos
	var args []cst.Expr
	var index *cst.BasicLit

	isRef := false
	isFunc := false
	if p.tok == token.LPAREN {
		isFunc = true
		lparenPos = p.expect(token.LPAREN)
		args, rparenPos = p.parseExprList(token.RPAREN)
	} else if p.tok == token.LBRACK {
		isRef = true
		lparenPos = p.expect(token.LBRACK)
		if p.tok == token.STRINGLIT {
			value := p.parseString()
			index = &value
		}
		rparenPos = p.expect(token.RBRACK)
	}

	return &cst.MemberAccess{
		IsFunc:    isFunc,
		IsRef:     isRef,
		Ident:     ident,
		LparenPos: lparenPos,
		Args:      args,
		Index:     index,
		RparenPos: rparenPos,
	}
}

// ----------------------------------------------------------------------------
// Member ::= Primary {Access}
func (p *parser) parseMember() cst.Expr {
	if p.trace {
		defer un(trace(p, "Member"))
	}
	// if p.tok != token.PRINCIPAL && p.tok != token.ACTION && p.tok != token.RESOURCE && p.tok != token.CONTEXT {
	// 	return p.parsePrimary()
	// }
	primary := p.parsePrimary()
	if p.tok != token.PERIOD {
		return primary
	}

	var access []*cst.MemberAccess
	for p.tok == token.PERIOD {
		p.next()
		item := p.parseAccess()

		if item != nil {
			access = append(access, item)
		}
	}

	return &cst.MemberExpr{
		Primary: primary,
		Access:  access,
	}
}

// ----------------------------------------------------------------------------
// Unary ::= ['!' | '-']x4 Member
func (p *parser) parseUnary() cst.Expr {
	if p.trace {
		defer un(trace(p, "Unary"))
	}
	pos := p.pos
	if p.tok != token.NOT && p.tok != token.SUB {
		return p.parseMember()
	}

	tok := p.tok
	p.next()
	return &cst.UnaryExpr{
		OpPos: pos,
		Op:    tok,
		X:     p.parseUnary(),
	}
}

// ----------------------------------------------------------------------------
// Mult ::= Unary { '*' Unary}
func (p *parser) parseMult() cst.Expr {
	if p.trace {
		defer un(trace(p, "Mult"))
	}

	lhs := p.parseUnary()
	if p.tok != token.MUL {
		return lhs
	}
	tok := p.tok
	pos := p.pos
	p.next()
	rhs := p.parseMult()

	return &cst.BinaryExpr{
		X:     lhs,
		OpPos: pos,
		Op:    tok,
		Y:     rhs,
	}
}

// ----------------------------------------------------------------------------
// Add ::= Mult {('+' | '-') Mult}
func (p *parser) parseAdd() cst.Expr {
	if p.trace {
		defer un(trace(p, "Add"))
	}

	lhs := p.parseMult()
	if p.tok != token.ADD && p.tok != token.SUB {
		return lhs
	}
	tok := p.tok
	pos := p.pos
	p.next()
	rhs := p.parseAdd()

	return &cst.BinaryExpr{
		X:     lhs,
		OpPos: pos,
		Op:    tok,
		Y:     rhs,
	}
}

// ----------------------------------------------------------------------------
// Relation ::= Add [RELOP Add] | Add 'has' (IDENT | STR) | Add 'like' PAT
func (p *parser) parseRelation() cst.Expr {
	if p.trace {
		defer un(trace(p, "Relation"))
	}

	lhs := p.parseAdd()

	pos := p.pos
	tok := p.tok

	switch tok {
	case token.LSS, token.LEQ, token.GTR, token.GEQ,
		token.EQL, token.NEQ,
		token.IN:
		p.next()
		rhs := p.parseAdd()

		return &cst.BinaryExpr{
			X:     lhs,
			OpPos: pos,
			Op:    tok,
			Y:     rhs,
		}
	case token.HAS:
		p.next()
		if p.tok != token.STRINGLIT && p.tok != token.IDENTIFER {
			p.error(p.pos, "expected string")

			bad := &cst.BadExpr{
				From: lhs.Pos(),
				To:   p.pos,
			}
			p.next()

			return bad
		}
		lit := &cst.BasicLit{ValuePos: p.pos, Kind: p.tok, Value: p.lit}
		p.next()
		return &cst.BinaryExpr{
			X:     lhs,
			OpPos: pos,
			Op:    tok,
			Y:     lit,
		}
	case token.LIKE:
		p.next()
		if p.tok != token.STRINGLIT {
			p.next()
			p.error(p.pos, "expected string")
		}
		lit := &cst.BasicLit{ValuePos: p.pos, Kind: p.tok, Value: p.lit}
		p.next()
		return &cst.BinaryExpr{
			X:     lhs,
			OpPos: pos,
			Op:    tok,
			Y:     lit,
		}
	}

	return lhs
}

// ----------------------------------------------------------------------------
// And ::= Relation {'&&' Relation}
func (p *parser) parseAnd() cst.Expr {
	if p.trace {
		defer un(trace(p, "And"))
	}

	lhs := p.parseRelation()
	if p.tok != token.LAND {
		return lhs
	}
	tok := p.tok
	pos := p.pos
	p.next()
	rhs := p.parseAnd()

	return &cst.BinaryExpr{
		X:     lhs,
		OpPos: pos,
		Op:    tok,
		Y:     rhs,
	}
}

// ----------------------------------------------------------------------------
// Or ::= And {'||' And}
func (p *parser) parseOr() cst.Expr {
	if p.trace {
		defer un(trace(p, "Or"))
	}

	lhs := p.parseAnd()
	if p.tok != token.LOR {
		return lhs
	}
	tok := p.tok
	pos := p.pos
	p.next()
	rhs := p.parseOr()

	return &cst.BinaryExpr{
		X:     lhs,
		OpPos: pos,
		Op:    tok,
		Y:     rhs,
	}
}

func (p *parser) parseIf() cst.Expr {
	if p.trace {
		defer un(trace(p, "If"))
	}

	pos := p.pos
	p.next()
	condition := p.parseExpr()
	p.expect(token.THEN)
	thenExpr := p.parseExpr()
	p.expect(token.ELSE)
	elseExpr := p.parseExpr()

	return &cst.IfExpr{
		IfPos:     pos,
		Condition: condition,
		Then:      thenExpr,
		Else:      elseExpr,
	}
}

// ----------------------------------------------------------------------------
// Expr ::= Or | 'if' Expr 'then' Expr 'else' Expr
func (p *parser) parseExpr() cst.Expr {
	if p.trace {
		defer un(trace(p, "Expr"))
	}
	if p.tok == token.IF {
		return p.parseIf()
	} else {
		return p.parseOr()
	}

}

// ----------------------------------------------------------------------------
// Condition ::= ('when' | 'unless') '{' Expr '}'
func (p *parser) parseConditions() []*cst.Condition {
	if p.trace {
		defer un(trace(p, "Condition"))
	}

	var conditions []*cst.Condition

	for p.tok == token.WHEN || p.tok == token.UNLESS {
		node := cst.Condition{
			ConditionPos: p.pos,
			Condition:    p.tok,
		}
		p.next()

		node.Lbrace = p.expect(token.LBRACE)
		node.Expr = p.parseExpr()
		node.Rbrace = p.expect(token.RBRACE)

		conditions = append(conditions, &node)
	}

	return conditions
}

// ----------------------------------------------------------------------------
// Action ::= 'action' [( '==' Entity | 'in' ('[' EntList ']' | Entity) )]
func (p *parser) parseAction() cst.Variable {
	if p.trace {
		defer un(trace(p, "Action"))
	}

	pos := p.expect(token.ACTION)
	lit := cst.BasicLit{ValuePos: pos, Kind: token.ACTION, Value: p.lit}
	node := cst.Variable{
		NameLit: lit,
	}

	if p.tok != token.IN && p.tok != token.EQL {
		return node
	}
	node.RelOp = p.tok
	node.RelPos = p.expect(p.tok)

	if node.RelOp == token.EQL || p.tok != token.LBRACK {
		entity := p.parseEntity()
		if entity != nil {
			node.Entities = []*cst.EntityName{entity}
		}

		return node
	}
	pos = p.expect(token.LBRACK)
	exprs, end := p.parseExprList(token.RBRACK)
	node.SetExpr = &cst.SetExpr{
		Lbrack: pos,
		Exprs:  exprs,
		Rbrack: end,
	}
	node.PosEnd = node.SetExpr.End()

	return node
}

// ----------------------------------------------------------------------------
//
// V2.0
// Principal ::= 'principal' [('in' | '==') (Entity | '?principal')]
// Resource ::= 'resource' [('in' | '==') (Entity | '?resource')]
// V3.0
// Principal ::= 'principal' [(['is' PATH] ['in' (Entity | '?principal')]) | ('==' (Entity | '?principal'))]
// Resource  ::= 'resource'  [(['is' PATH] ['in' (Entity | '?resource')])  | ('==' (Entity | '?resource'))]
func (p *parser) parsePrincipalResource(expect token.Token) cst.Variable {
	if p.trace {
		defer un(trace(p, "Principal/Resource"))
	}

	pos := p.expect(expect)
	lit := cst.BasicLit{ValuePos: pos, Kind: expect, Value: p.lit}

	node := cst.Variable{
		NameLit: lit,
	}
	var wildcard token.Token
	if expect == token.PRINCIPAL {
		wildcard = token.PRINCIPAL_SLOT
	} else if expect == token.RESOURCE {
		wildcard = token.RESOURCE_SLOT
	}

	if p.tok != token.IN && p.tok != token.EQL && p.tok != token.IS {
		return node
	}

	if p.tok == token.IS {
		p.next()
		node.IsPos = p.pos
		expr := p.parseEntityOrPath(false, true)
		if expr != nil {
			return node
		}
		path, ok := expr.(*cst.Path)
		if !ok {
			return node
		}
		node.IsCheck = &cst.EntityName{Path: append(path.Path, cst.BasicLit{Kind: token.STRINGLIT})}
	}

	node.RelOp = p.tok
	node.RelPos = p.expect(p.tok)

	if p.tok == wildcard {
		node.Slot = p.tok
		node.PosEnd = p.pos + token.Pos(len(wildcard.String()))
		p.next()
	} else {
		entity := p.parseEntity()
		if entity != nil {
			node.Entities = []*cst.EntityName{entity}
		}
	}

	return node
}

// ----------------------------------------------------------------------------
// Scope ::= Principal ',' Action ',' Resource
func (p *parser) parseScope() *cst.ScopeNew {
	if p.trace {
		defer un(trace(p, "Scope"))
	}

	scope := &cst.ScopeNew{}

	scope.Lparen = p.expect(token.LPAREN)
	scope.Principal = p.parsePrincipalResource(token.PRINCIPAL)
	scope.Comma1 = p.expect(token.COMMA)
	scope.Action = p.parseAction()
	scope.Comma2 = p.expect(token.COMMA)
	scope.Resource = p.parsePrincipalResource(token.RESOURCE)
	scope.Rparen = p.expect(token.RPAREN)

	return scope
}

func (p *parser) parseIdent() *cst.BasicLit {
	if p.trace {
		defer un(trace(p, "Ident"))
	}

	pos := p.pos
	name := "_"
	if p.tok == token.IDENTIFER {
		name = p.lit
		p.next()
	} else {
		p.expect(token.IDENTIFER) // use expect() error handling
	}
	// return cst.Ident{NamePos: pos, Name: name}
	return &cst.BasicLit{ValuePos: pos, Kind: token.IDENTIFER, Value: name}
}

// Consume a string
func (p *parser) parseString() cst.BasicLit {
	if p.trace {
		defer un(trace(p, "Ident"))
	}

	pos := p.pos
	var name string
	if p.tok == token.STRINGLIT {
		name = p.lit
		p.next()
	} else {
		p.expect(token.STRINGLIT) // use expect() error handling
	}
	return cst.BasicLit{ValuePos: pos, Kind: token.STRINGLIT, Value: name}
}

// ----------------------------------------------------------------------------
// Annotation ::= '@'IDENT'('STR')'
func (p *parser) parseAnnotation() []*cst.AnnotationSpec {
	if p.trace {
		defer un(trace(p, "Annotations"))
	}

	var annotations []*cst.AnnotationSpec

	for p.tok == token.AT {
		node := &cst.AnnotationSpec{
			TokPos: p.pos,
		}
		p.next()
		node.Ident = p.parseIdent()
		node.Lparen = p.expect(token.LPAREN)

		if p.tok == token.STRINGLIT {
			node.Value = p.parseString()
		} else {
			p.expect(token.STRINGLIT) // use expect() error handling
		}

		node.Rparen = p.expect(token.RPAREN)

		annotations = append(annotations, node)
	}

	return annotations
}

// ----------------------------------------------------------------------------
// Policy ::= {Annotation} Effect '(' Scope ')' {Conditions} ';'
func (p *parser) parsePolicy() cst.Decl {
	if p.trace {
		defer un(trace(p, "Policy"))
	}

	fromPos := p.pos
	annotations := p.parseAnnotation()

	effect := p.tok
	if p.tok != token.PERMIT && p.tok != token.FORBID {
		p.error(p.pos, "expected either 'permit' or 'forbid'")
		p.next()
		// ?? should we move to the next ';' ?
		return nil
	}
	p.next()

	scope := p.parseScope()
	conditions := p.parseConditions()
	toPos := p.expect(token.SEMICOLON)

	return &cst.PolicyStmt{
		From:        fromPos,
		To:          toPos,
		Annotations: annotations,
		Effect:      effect,
		Scope:       scope,
		Conditions:  conditions,
	}
}

// ----------------------------------------------------------------------------
// Source files

func (p *parser) parseFile() *cst.File {
	if p.trace {
		defer un(trace(p, "File"))
	}

	var stmts []cst.Decl
	for p.tok != token.EOF {
		stmts = append(stmts, p.parsePolicy())
	}

	return &cst.File{
		Statements: stmts,
		Comments:   p.comments,
	}
}
