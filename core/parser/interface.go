// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file contains the exported entry points for invoking the parser.

package parser

import (
	"bytes"
	"errors"
	"io"
	"os"

	"github.com/koblas/cedar-go/core/ast"
	"github.com/koblas/cedar-go/core/cst"
	"github.com/koblas/cedar-go/core/token"
)

// If src != nil, readSource converts src to a []byte if possible;
// otherwise it returns an error. If src == nil, readSource returns
// the result of reading the file specified by filename.
func readSource(filename string, src interface{}) ([]byte, error) {
	if src != nil {
		switch s := src.(type) {
		case string:
			return []byte(s), nil
		case []byte:
			return s, nil
		case *bytes.Buffer:
			// is io.Reader, but src is already available in []byte form
			if s != nil {
				return s.Bytes(), nil
			}
		case io.Reader:
			return io.ReadAll(s)
		}
		return nil, errors.New("invalid source")
	}
	return os.ReadFile(filename)
}

// A Mode value is a set of flags (or 0).
// They control the amount of source code parsed and other optional
// parser functionality.
type Mode uint

const (
	ParseComments     Mode = 1 << iota // parse comments and add them to AST
	Trace                              // print a trace of parsed productions
	DeclarationErrors                  // report declaration errors
	AllErrors                          // report all errors (not just the first 10 on different lines)
)

// ParseFile parses the source code of a single Go source file and returns
// the corresponding cst.File node. The source code may be provided via
// the filename of the source file, or via the src parameter.
//
// If src != nil, ParseFile parses the source from src and the filename is
// only used when recording position information. The type of the argument
// for the src parameter must be string, []byte, or io.Reader.
// If src == nil, ParseFile parses the file specified by filename.
//
// The mode parameter controls the amount of source text parsed and other
// optional parser functionality. Position information is recorded in the
// file set fset, which must not be nil.
//
// If the source couldn't be read, the returned AST is nil and the error
// indicates the specific failure. If the source was read but syntax
// errors were found, the result is a partial AST (with cst.Bad* nodes
// representing the fragments of erroneous source code). Multiple errors
// are returned via a scanner.ErrorList which is sorted by source position.
func ParseFile(fset *token.FileSet, filename string, src interface{}, mode Mode) (f *cst.File, err error) {
	if fset == nil {
		panic("parser.ParseFile: no token.FileSet provided (fset == nil)")
	}

	// get source
	text, err := readSource(filename, src)
	if err != nil {
		return nil, err
	}

	var p parser
	defer func() {
		if e := recover(); e != nil {
			// resume same panic if it's not a bailout
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

		// set result values
		if f == nil {
			// source is not a valid Go source file - satisfy
			// ParseFile API and return a valid (but) empty
			// *cst.File
			f = &cst.File{
				// Name:  new(cst.Ident),
				// Scope: cst.NewScope(nil),
			}
		}

		p.errors.Sort()
		err = p.errors.Err()
	}()

	// parse source
	p.init(fset, filename, text, mode)
	f = p.parseFile()

	return
}

func ParseRules(src string) (ast.PolicyList, error) {
	fset := token.NewFileSet()
	data, err := ParseFile(fset, "", src, 0)
	if err != nil {
		return nil, err
	}

	var policies ast.PolicyList
	// There is only one
	fset.Iterate(func(file *token.File) bool {
		policies, err = cst.ToAst(file, data)

		return true
	})

	return policies, err
}

func ParseRulesTrace(src string) (ast.PolicyList, error) {
	fset := token.NewFileSet()
	data, err := ParseFile(fset, "", src, Trace)
	if err != nil {
		return nil, err
	}

	var policies ast.PolicyList
	// There is only one
	fset.Iterate(func(file *token.File) bool {
		policies, err = cst.ToAst(file, data)

		return true
	})

	return policies, err
}
