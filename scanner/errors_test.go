// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scanner_test

import (
	"fmt"

	"github.com/koblas/cedar-go/core/scanner"
	"github.com/koblas/cedar-go/core/token"
)

func ExampleScanner_Scan() {
	// src is the input that we want to tokenize.
	src := []byte(`
	permit(
		// TEST
		principal == User::"alice", 
	);`)

	// Initialize the scanner.
	var s scanner.Scanner
	fset := token.NewFileSet()                      // positions are relative to fset
	file := fset.AddFile("", fset.Base(), len(src)) // register input "file"
	s.Init(file, src, nil /* no error handler */, scanner.ScanComments)

	// Repeated calls to Scan yield the token sequence found in the input.
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		fmt.Printf("%s\t%s\t%q\n", fset.Position(pos), tok, lit)
	}

	// output:
	// 2:2	permit	"permit"
	// 2:8	(	""
	// 3:3	COMMENT	"// TEST"
	// 4:3	principal	"principal"
	// 4:13	==	""
	// 4:16	IDENTIFIER	"User"
	// 4:20	::	""
	// 4:22	STRINGLIT	"\"alice\""
	// 4:29	,	","
	// 5:2	)	""
	// 5:3	;	";"
}
