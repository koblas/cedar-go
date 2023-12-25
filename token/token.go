package token

import (
	"strconv"
	"unicode"
)

type Token int

// The list of tokens
const (
	ILLEGAL Token = iota
	EOF

	COMMENT

	keyword_beg
	// Special Identifiers (begin expression)
	TRUE
	FALSE
	IF

	// Common Identifiers
	PERMIT
	FORBID
	WHEN
	UNLESS
	IN
	HAS
	LIKE
	IS
	THEN
	ELSE

	// main idents
	PRINCIPAL
	ACTION
	RESOURCE
	CONTEXT

	// Valid slots, hardcoded for now, may be generalized later
	PRINCIPAL_SLOT
	RESOURCE_SLOT
	keyword_end

	// data input
	literal_beg
	IDENTIFER
	INT
	STRINGLIT
	literal_end

	// Other Tokens
	operator_beg
	AT // @

	PERIOD    // .
	COMMA     // ,
	SEMICOLON // ;
	COLON     // :
	PATH      // ::

	EQL // ==
	LSS // <
	GTR // >
	NEQ // !=
	LEQ // <=
	GEQ // >=

	LPAREN // (
	LBRACK // [
	LBRACE // {
	RPAREN // )
	RBRACK // ]
	RBRACE // }

	LAND // &&
	LOR  // ||

	ADD // +
	SUB // -
	MUL // *
	QUO // /
	REM // %

	NOT // !
	operator_end
)

var tokens = [...]string{
	AT: "@",

	EOF:     "EOF",
	COMMENT: "COMMENT",

	// Special Identifiers (begin expression)
	TRUE:  "true",
	FALSE: "false",
	IF:    "if",

	// Common Identifiers
	PERMIT: "permit",
	FORBID: "forbid",
	WHEN:   "when",
	UNLESS: "unless",
	IN:     "in",
	HAS:    "has",
	LIKE:   "like",
	IS:     "is",
	THEN:   "then",
	ELSE:   "else",

	// main idents
	PRINCIPAL: "principal",
	ACTION:    "action",
	RESOURCE:  "resource",
	CONTEXT:   "context",

	// Valid slots, hardcoded for now, may be generalized later
	PRINCIPAL_SLOT: "?principal",
	RESOURCE_SLOT:  "?resource",

	IDENTIFER: "IDENTIFIER",
	INT:       "INT",
	STRINGLIT: "STRINGLIT",

	PERIOD:    ".",
	COMMA:     ",",
	SEMICOLON: ";",
	COLON:     ":",
	PATH:      "::",

	EQL: "==",
	LSS: "<",
	GTR: ">",
	NEQ: "!=",
	LEQ: "<=",
	GEQ: ">=",

	LPAREN: "(",
	LBRACK: "[",
	LBRACE: "{",
	RPAREN: ")",
	RBRACK: "]",
	RBRACE: "}",

	LAND: "&&",
	LOR:  "||",

	ADD: "+",
	SUB: "-",
	MUL: "*",
	QUO: "/",
	REM: "%",

	NOT: "!",
}

// String returns the string corresponding to the token tok.
// For operators, delimiters, and keywords the string is the actual
// token character sequence (e.g., for the token [ADD], the string is
// "+"). For all other tokens the string corresponds to the token
// constant name (e.g. for the token [IDENTIFER], the string is "IDENTIFER").
func (tok Token) String() string {
	if 0 <= tok && tok < Token(len(tokens)) {
		return tokens[tok]
	}
	return "token(" + strconv.Itoa(int(tok)) + ")"
}

var keywords map[string]Token

func init() {
	keywords = make(map[string]Token, keyword_end-(keyword_beg+1))
	for i := keyword_beg + 1; i < keyword_end; i++ {
		keywords[tokens[i]] = i
	}
}

// Lookup maps an identifier to its keyword token or [IDENTIFIER] (if not a keyword).
func Lookup(ident string) Token {
	if tok, is_keyword := keywords[ident]; is_keyword {
		return tok
	}
	return IDENTIFER
}

// Predicates

// IsLiteral returns true for tokens corresponding to identifiers
// and basic type literals; it returns false otherwise.
func (tok Token) IsLiteral() bool { return literal_beg < tok && tok < literal_end }

// IsOperator returns true for tokens corresponding to operators and
// delimiters; it returns false otherwise.
func (tok Token) IsOperator() bool {
	return (operator_beg < tok && tok < operator_end)
}

// IsKeyword returns true for tokens corresponding to keywords;
// it returns false otherwise.
func (tok Token) IsKeyword() bool { return keyword_beg < tok && tok < keyword_end }

// IsKeyword reports whether name is a Go keyword, such as "func" or "return".
func IsKeyword(name string) bool {
	// TODO: opt: use a perfect hash function instead of a global map.
	_, ok := keywords[name]
	return ok
}

// IsIdentifier reports whether name is a Go identifier, that is, a non-empty
// string made up of letters, digits, and underscores, where the first character
// is not a digit. Keywords are not identifiers.
func IsIdentifier(name string) bool {
	if name == "" || IsKeyword(name) {
		return false
	}
	for i, c := range name {
		if !unicode.IsLetter(c) && c != '_' && (i == 0 || !unicode.IsDigit(c)) {
			return false
		}
	}
	return true
}
