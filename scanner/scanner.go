// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package scanner implements a scanner for Go source text.
// It takes a []byte as source which can then be tokenized
// through repeated calls to the Scan method.
package scanner

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/koblas/cedar-go/token"
)

// An ErrorHandler may be provided to [Scanner.Init]. If a syntax error is
// encountered and a handler was installed, the handler is called with a
// position and an error message. The position points to the beginning of
// the offending token.
type ErrorHandler func(pos token.Position, msg string)

// A Scanner holds the scanner's internal state while processing
// a given text. It can be allocated as part of another data
// structure but must be initialized via [Scanner.Init] before use.
type Scanner struct {
	// immutable state
	file *token.File  // source file handle
	dir  string       // directory portion of file.Name()
	src  []byte       // source
	err  ErrorHandler // error reporting; or nil
	mode Mode         // scanning mode

	// scanning state
	ch         rune      // current character
	offset     int       // character offset
	rdOffset   int       // reading offset (position after current character)
	lineOffset int       // current line offset
	nlPos      token.Pos // position of newline in preceding comment

	// public state - ok to modify
	ErrorCount int // number of errors encountered
}

const (
	bom = 0xFEFF // byte order mark, only permitted as very first character
	eof = -1     // end of file
)

// Read the next Unicode char into s.ch.
// s.ch < 0 means end-of-file.
//
// For optimization, there is some overlap between this method and
// s.scanIdentifier.
func (s *Scanner) next() {
	if s.rdOffset < len(s.src) {
		s.offset = s.rdOffset
		if s.ch == '\n' {
			s.lineOffset = s.offset
			s.file.AddLine(s.offset)
		}
		r, w := rune(s.src[s.rdOffset]), 1
		switch {
		case r == 0:
			s.error(s.offset, "illegal character NUL")
		case r >= utf8.RuneSelf:
			// not ASCII
			r, w = utf8.DecodeRune(s.src[s.rdOffset:])
			if r == utf8.RuneError && w == 1 {
				s.error(s.offset, "illegal UTF-8 encoding")
			} else if r == bom && s.offset > 0 {
				s.error(s.offset, "illegal byte order mark")
			}
		}
		s.rdOffset += w
		s.ch = r
	} else {
		s.offset = len(s.src)
		if s.ch == '\n' {
			s.lineOffset = s.offset
			s.file.AddLine(s.offset)
		}
		s.ch = eof
	}
}

// peek returns the byte following the most recently read character without
// advancing the scanner. If the scanner is at EOF, peek returns 0.
func (s *Scanner) peek() byte {
	if s.rdOffset < len(s.src) {
		return s.src[s.rdOffset]
	}
	return 0
}

// A mode value is a set of flags (or 0).
// They control scanner behavior.
type Mode uint

const (
	ScanComments    Mode = 1 << iota // return comments as COMMENT tokens
	dontInsertSemis                  // do not automatically insert semicolons - for testing only
)

// Init prepares the scanner s to tokenize the text src by setting the
// scanner at the beginning of src. The scanner uses the file set file
// for position information and it adds line information for each line.
// It is ok to re-use the same file when re-scanning the same file as
// line information which is already present is ignored. Init causes a
// panic if the file size does not match the src size.
//
// Calls to [Scanner.Scan] will invoke the error handler err if they encounter a
// syntax error and err is not nil. Also, for each error encountered,
// the [Scanner] field ErrorCount is incremented by one. The mode parameter
// determines how comments are handled.
//
// Note that Init may call err if there is an error in the first character
// of the file.
func (s *Scanner) Init(file *token.File, src []byte, err ErrorHandler, mode Mode) {
	// Explicitly initialize all fields since a scanner may be reused.
	if file.Size() != len(src) {
		panic(fmt.Sprintf("file size (%d) does not match src len (%d)", file.Size(), len(src)))
	}
	s.file = file
	s.dir, _ = filepath.Split(file.Name())
	s.src = src
	s.err = err
	s.mode = mode

	s.ch = ' '
	s.offset = 0
	s.rdOffset = 0
	s.lineOffset = 0
	s.ErrorCount = 0

	s.next()
	if s.ch == bom {
		s.next() // ignore BOM at file beginning
	}
}

func (s *Scanner) error(offs int, msg string) {
	if s.err != nil {
		s.err(s.file.Position(s.file.Pos(offs)), msg)
	}
	s.ErrorCount++
}

func (s *Scanner) errorf(offs int, format string, args ...any) {
	s.error(offs, fmt.Sprintf(format, args...))
}

// scanComment returns the text of the comment and (if nonzero)
// the offset of the first newline within it, which implies a
// /*...*/ comment.
func (s *Scanner) scanComment() (string, int) {
	// initial '/' already consumed; s.ch == '/' || s.ch == '*'
	offs := s.offset - 1 // position of initial '/'
	next := -1           // position immediately following the comment; < 0 means invalid comment
	numCR := 0
	nlOffset := 0 // offset of first newline within /*...*/ comment

	if s.ch == '/' {
		//-style comment
		// (the final '\n' is not considered part of the comment)
		s.next()
		for s.ch != '\n' && s.ch >= 0 {
			if s.ch == '\r' {
				numCR++
			}
			s.next()
		}
		// if we are at '\n', the position following the comment is afterwards
		next = s.offset
		if s.ch == '\n' {
			next++
		}
		goto exit
	}

	/*-style comment */
	s.next()
	for s.ch >= 0 {
		ch := s.ch
		if ch == '\r' {
			numCR++
		} else if ch == '\n' && nlOffset == 0 {
			nlOffset = s.offset
		}
		s.next()
		if ch == '*' && s.ch == '/' {
			s.next()
			next = s.offset
			goto exit
		}
	}

	s.error(offs, "comment not terminated")

exit:
	lit := s.src[offs:s.offset]

	// On Windows, a (//-comment) line may end in "\r\n".
	// Remove the final '\r' before analyzing the text for
	// line directives (matching the compiler). Remove any
	// other '\r' afterwards (matching the pre-existing be-
	// havior of the scanner).
	if numCR > 0 && len(lit) >= 2 && lit[1] == '/' && lit[len(lit)-1] == '\r' {
		lit = lit[:len(lit)-1]
		numCR--
	}

	// interpret line directives
	// (//line directives must start at the beginning of the current line)
	if next >= 0 /* implies valid comment */ && (lit[1] == '*' || offs == s.lineOffset) && bytes.HasPrefix(lit[2:], prefix) {
		s.updateLineInfo(next, offs, lit)
	}

	if numCR > 0 {
		lit = stripCR(lit, lit[1] == '*')
	}

	return string(lit), nlOffset
}

var prefix = []byte("line ")

// updateLineInfo parses the incoming comment text at offset offs
// as a line directive. If successful, it updates the line info table
// for the position next per the line directive.
func (s *Scanner) updateLineInfo(next, offs int, text []byte) {
	// extract comment text
	if text[1] == '*' {
		text = text[:len(text)-2] // lop off trailing "*/"
	}
	text = text[7:] // lop off leading "//line " or "/*line "
	offs += 7

	i, n, ok := trailingDigits(text)
	if i == 0 {
		return // ignore (not a line directive)
	}
	// i > 0

	if !ok {
		// text has a suffix :xxx but xxx is not a number
		s.error(offs+i, "invalid line number: "+string(text[i:]))
		return
	}

	// Put a cap on the maximum size of line and column numbers.
	// 30 bits allows for some additional space before wrapping an int32.
	// Keep this consistent with cmd/compile/syntax.PosMax.
	const maxLineCol = 1 << 30
	var line, col int
	i2, n2, ok2 := trailingDigits(text[:i-1])
	if ok2 {
		//line filename:line:col
		i, i2 = i2, i
		line, col = n2, n
		if col == 0 || col > maxLineCol {
			s.error(offs+i2, "invalid column number: "+string(text[i2:]))
			return
		}
		text = text[:i2-1] // lop off ":col"
	} else {
		//line filename:line
		line = n
	}

	if line == 0 || line > maxLineCol {
		s.error(offs+i, "invalid line number: "+string(text[i:]))
		return
	}

	// If we have a column (//line filename:line:col form),
	// an empty filename means to use the previous filename.
	filename := string(text[:i-1]) // lop off ":line", and trim white space
	if filename == "" && ok2 {
		filename = s.file.Position(s.file.Pos(offs)).Filename
	} else if filename != "" {
		// Put a relative filename in the current directory.
		// This is for compatibility with earlier releases.
		// See issue 26671.
		filename = filepath.Clean(filename)
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(s.dir, filename)
		}
	}

	s.file.AddLineColumnInfo(next, filename, line, col)
}

func trailingDigits(text []byte) (int, int, bool) {
	i := bytes.LastIndexByte(text, ':') // look from right (Windows filenames may contain ':')
	if i < 0 {
		return 0, 0, false // no ":"
	}
	// i >= 0
	n, err := strconv.ParseUint(string(text[i+1:]), 10, 0)
	return i + 1, int(n), err == nil
}

func isLetter(ch rune) bool {
	return 'a' <= lower(ch) && lower(ch) <= 'z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return isDecimal(ch) || ch >= utf8.RuneSelf && unicode.IsDigit(ch)
}

// scanIdentifier reads the string of valid identifier characters at s.offset.
// It must only be called when s.ch is known to be a valid letter.
//
// Be careful when making changes to this function: it is optimized and affects
// scanning performance significantly.
func (s *Scanner) scanIdentifier() string {
	offs := s.offset

	// Optimize for the common case of an ASCII identifier.
	//
	// Ranging over s.src[s.rdOffset:] lets us avoid some bounds checks, and
	// avoids conversions to runes.
	//
	// In case we encounter a non-ASCII character, fall back on the slower path
	// of calling into s.next().
	for rdOffset, b := range s.src[s.rdOffset:] {
		if 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z' || b == '_' || '0' <= b && b <= '9' {
			// Avoid assigning a rune for the common case of an ascii character.
			continue
		}
		s.rdOffset += rdOffset
		if 0 < b && b < utf8.RuneSelf {
			// Optimization: we've encountered an ASCII character that's not a letter
			// or number. Avoid the call into s.next() and corresponding set up.
			//
			// Note that s.next() does some line accounting if s.ch is '\n', so this
			// shortcut is only possible because we know that the preceding character
			// is not '\n'.
			s.ch = rune(b)
			s.offset = s.rdOffset
			s.rdOffset++
			goto exit
		}
		// We know that the preceding character is valid for an identifier because
		// scanIdentifier is only called when s.ch is a letter, so calling s.next()
		// at s.rdOffset resets the scanner state.
		s.next()
		for isLetter(s.ch) || isDigit(s.ch) {
			s.next()
		}
		goto exit
	}
	s.offset = len(s.src)
	s.rdOffset = len(s.src)
	s.ch = eof

exit:
	return string(s.src[offs:s.offset])
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= lower(ch) && lower(ch) <= 'f':
		return int(lower(ch) - 'a' + 10)
	}
	return 16 // larger than any legal digit val
}

func lower(ch rune) rune     { return ('a' - 'A') | ch } // returns lower-case ch iff ch is ASCII letter
func isDecimal(ch rune) bool { return '0' <= ch && ch <= '9' }
func isHex(ch rune) bool     { return '0' <= ch && ch <= '9' || 'a' <= lower(ch) && lower(ch) <= 'f' }

// digits accepts the sequence { digit | '_' }.
// digits returns a bitset describing whether the sequence contained
// digits (bit 0 is set), or separators '_' (bit 1 is set).
func (s *Scanner) digits(base int, invalid *int) int {
	digsep := 0

	for isDecimal(s.ch) || s.ch == '_' {
		ds := 1
		if s.ch == '_' {
			ds = 2
		} else if s.ch > '9' && *invalid < 0 {
			*invalid = s.offset // record invalid rune offset
		}
		digsep |= ds
		s.next()
	}

	return digsep
}

func (s *Scanner) scanNumber() (token.Token, string) {
	offs := s.offset
	tok := token.ILLEGAL

	base := 10        // number base
	prefix := rune(0) // one of 0 (decimal), '0' (0-octal), 'x', 'o', or 'b'
	digsep := 0       // bit 0: digit present, bit 1: '_' present
	invalid := -1     // index of invalid digit in literal, or < 0

	// integer part
	tok = token.INT
	digsep |= s.digits(base, &invalid)

	if digsep&1 == 0 {
		s.error(s.offset, litname(prefix)+" has no digits")
	}

	lit := string(s.src[offs:s.offset])
	if tok == token.INT && invalid >= 0 {
		s.errorf(invalid, "invalid digit %q in %s", lit[invalid-offs], litname(prefix))
	}
	if digsep&2 != 0 {
		if i := invalidSep(lit); i >= 0 {
			s.error(offs+i, "'_' must separate successive digits")
		}
	}

	return tok, lit
}

func litname(prefix rune) string {
	switch prefix {
	case 'x':
		return "hexadecimal literal"
	case 'o', '0':
		return "octal literal"
	case 'b':
		return "binary literal"
	}
	return "decimal literal"
}

// invalidSep returns the index of the first invalid separator in x, or -1.
func invalidSep(x string) int {
	x1 := ' ' // prefix char, we only care if it's 'x'
	d := '.'  // digit, one of '_', '0' (a digit), or '.' (anything else)
	i := 0

	// a prefix counts as a digit
	if len(x) >= 2 && x[0] == '0' {
		x1 = lower(rune(x[1]))
		if x1 == 'x' || x1 == 'o' || x1 == 'b' {
			d = '0'
			i = 2
		}
	}

	// mantissa and exponent
	for ; i < len(x); i++ {
		p := d // previous digit
		d = rune(x[i])
		switch {
		case d == '_':
			if p != '0' {
				return i
			}
		case isDecimal(d) || x1 == 'x' && isHex(d):
			d = '0'
		default:
			if p == '_' {
				return i - 1
			}
			d = '.'
		}
	}
	if d == '_' {
		return len(x) - 1
	}

	return -1
}

// scanEscape parses an escape sequence where rune is the accepted
// escaped quote. In case of a syntax error, it stops at the offending
// character (without consuming it) and returns false. Otherwise
// it returns true.
func (s *Scanner) scanEscape(quote rune) bool {
	offs := s.offset

	var base, max uint32
	switch s.ch {
	case 'n', 'r', 't', '\\', '\'', '*', '0', quote:
		s.next()
		return true
	// case 'x':
	case 'u':
		base = 16
		max = unicode.MaxRune
		s.next()
		if s.ch != '{' {
			s.error(offs, "expected { to start unicode escape")
			return false
		}
		s.next()
	default:
		msg := "unknown escape sequence '\\" + string(s.ch) + "' in string"
		if s.ch < 0 {
			msg = "escape sequence not terminated"
		}
		s.error(offs, msg)
		return false
	}

	var x uint32
	for isHex(s.ch) {
		d := uint32(digitVal(s.ch))
		if d >= base {
			msg := fmt.Sprintf("illegal character %#U in escape sequence", s.ch)
			if s.ch < 0 {
				msg = "escape sequence not terminated"
			}
			s.error(s.offset, msg)
			return false
		}
		x = x*base + d
		s.next()
	}

	if s.ch != '}' {
		s.error(offs, "expected } to end unicode escape")
		return false
	}
	s.next()

	if x > max || 0xD800 <= x && x < 0xE000 {
		s.error(offs, "escape sequence is invalid Unicode code point")
		return false
	}

	return true
}

func (s *Scanner) scanString() string {
	// '"' opening already consumed
	offs := s.offset - 1

	for {
		ch := s.ch
		if ch == '\n' || ch < 0 {
			s.error(offs, "string literal not terminated")
			break
		}
		s.next()
		if ch == '"' {
			break
		}
		if ch == '\\' {
			s.scanEscape('"')
		}
	}

	// return string(s.src[offs:s.offset])
	return string(s.src[offs:s.offset])
}

func stripCR(b []byte, comment bool) []byte {
	c := make([]byte, len(b))
	i := 0
	for j, ch := range b {
		// In a /*-style comment, don't strip \r from *\r/ (incl.
		// sequences of \r from *\r\r...\r/) since the resulting
		// */ would terminate the comment too early unless the \r
		// is immediately following the opening /* in which case
		// it's ok because /*/ is not closed yet (issue #11151).
		if ch != '\r' || comment && i > len("/*") && c[i-1] == '*' && j+1 < len(b) && b[j+1] == '/' {
			c[i] = ch
			i++
		}
	}
	return c[:i]
}

func (s *Scanner) scanRawString() string {
	// '`' opening already consumed
	offs := s.offset - 1

	hasCR := false
	for {
		ch := s.ch
		if ch < 0 {
			s.error(offs, "raw string literal not terminated")
			break
		}
		s.next()
		if ch == '`' {
			break
		}
		if ch == '\r' {
			hasCR = true
		}
	}

	lit := s.src[offs:s.offset]
	if hasCR {
		lit = stripCR(lit, false)
	}

	return string(lit)
}

func (s *Scanner) skipWhitespace() {
	for s.ch == ' ' || s.ch == '\t' || s.ch == '\n' || s.ch == '\r' {
		s.next()
	}
}

// Scan scans the next token and returns the token position, the token,
// and its literal string if applicable. The source end is indicated by
// [token.EOF].
//
// If the returned token is a literal ([token.IDENT], [token.INT], [token.FLOAT],
// [token.IMAG], [token.CHAR], [token.STRING]) or [token.COMMENT], the literal string
// has the corresponding value.
//
// If the returned token is a keyword, the literal string is the keyword.
//
// If the returned token is [token.SEMICOLON], the corresponding
// literal string is ";" if the semicolon was present in the source,
// and "\n" if the semicolon was inserted because of a newline or
// at EOF.
//
// If the returned token is [token.ILLEGAL], the literal string is the
// offending character.
//
// In all other cases, Scan returns an empty literal string.
//
// For more tolerant parsing, Scan will return a valid token if
// possible even if a syntax error was encountered. Thus, even
// if the resulting token sequence contains no illegal tokens,
// a client may not assume that no error occurred. Instead it
// must check the scanner's ErrorCount or the number of calls
// of the error handler, if there was one installed.
//
// Scan adds line information to the file added to the file
// set with Init. Token positions are relative to that file
// and thus relative to the file set.
func (s *Scanner) Scan() (pos token.Pos, tok token.Token, lit string) {
scanAgain:
	if s.nlPos.IsValid() {
		// Return artificial ';' token after /*...*/ comment
		// containing newline, at position of first newline.
		pos, tok, lit = s.nlPos, token.SEMICOLON, "\n"
		s.nlPos = token.NoPos
		return
	}

	s.skipWhitespace()

	// current token start
	pos = s.file.Pos(s.offset)

	// determine token value
	switch ch := s.ch; {
	case isLetter(ch) || ch == '?':
		lit = s.scanIdentifier()
		if len(lit) > 1 {
			// keywords are longer than one letter - avoid lookup otherwise
			tok = token.Lookup(lit)
		} else {
			tok = token.IDENTIFER
		}
	case isDecimal(ch):
		tok, lit = s.scanNumber()
	default:
		s.next() // always make progress
		switch ch {
		case eof:
			tok = token.EOF
		case '\n':
			// we only reach here if s.insertSemi was
			// set in the first place and exited early
			// from s.skipWhitespace()
			return pos, token.SEMICOLON, "\n"
		case '"':
			tok = token.STRINGLIT
			lit = s.scanString()
		case '`':
			tok = token.STRINGLIT
			lit = s.scanRawString()
		case ':':
			if s.ch == ':' {
				s.next()
				tok = token.PATH
			} else {
				tok = token.COLON
			}
		case '.':
			// fractions starting with a '.' are handled by outer switch
			tok = token.PERIOD
		case ',':
			tok = token.COMMA
			lit = ","
		case ';':
			tok = token.SEMICOLON
			lit = ";"
		case '(':
			tok = token.LPAREN
		case ')':
			tok = token.RPAREN
		case '[':
			tok = token.LBRACK
		case ']':
			tok = token.RBRACK
		case '{':
			tok = token.LBRACE
		case '}':
			tok = token.RBRACE
		case '+':
			tok = token.ADD
		case '-':
			tok = token.SUB
		case '*':
			tok = token.MUL
		case '/':
			if s.ch == '/' || s.ch == '*' {
				// comment
				comment, _ := s.scanComment()
				if s.mode&ScanComments == 0 {
					// skip comment
					goto scanAgain
				}
				tok = token.COMMENT
				lit = comment
			} else {
				tok = token.QUO
			}
		case '@':
			tok = token.AT
		case '%':
			tok = token.REM
		case '<':
			if s.ch == '=' {
				s.next()
				tok = token.LEQ
			} else {
				tok = token.LSS
			}
		case '>':
			if s.ch == '=' {
				s.next()
				tok = token.GEQ
			} else {
				tok = token.GTR
			}
		case '=':
			if s.ch == '=' {
				s.next()
				tok = token.EQL
			} else {
				tok = token.ILLEGAL
				lit = string(ch)
			}
		case '!':
			if s.ch == '=' {
				s.next()
				tok = token.NEQ
			} else {
				tok = token.NOT
			}
		case '&':
			if s.ch == '&' {
				s.next()
				tok = token.LAND
			} else {
				tok = token.ILLEGAL
				lit = string(ch)
			}
		case '|':
			if s.ch == '|' {
				s.next()
				tok = token.LOR
			} else {
				tok = token.ILLEGAL
				lit = string(ch)
			}
		default:
			// next reports unexpected BOMs - don't repeat
			if ch != bom {
				// Report an informative error for U+201[CD] quotation
				// marks, which are easily introduced via copy and paste.
				if ch == '“' || ch == '”' {
					s.errorf(s.file.Offset(pos), "curly quotation mark %q (use neutral %q)", ch, '"')
				} else {
					s.errorf(s.file.Offset(pos), "illegal character %#U", ch)
				}
			}
			tok = token.ILLEGAL
			lit = string(ch)
		}
	}

	return
}
