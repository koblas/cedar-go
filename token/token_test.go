// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package token

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsIdentifier(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"Empty", "", false},
		{"Space", " ", false},
		{"SpaceSuffix", "foo ", false},
		{"Number", "123", false},
		{"Keyword", "forbid", false},

		{"LettersASCII", "foo", true},
		{"MixedASCII", "_bar123", true},
		{"UppercaseKeyword", "Func", true},
		{"LettersUnicode", "fóö", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := IsIdentifier(test.in)
			require.EqualValues(t, test.want, got)
		})
	}
}

func TestIsLiteral(t *testing.T) {
	require.True(t, INT.IsLiteral())
	require.False(t, MUL.IsLiteral())
	require.False(t, TRUE.IsLiteral())
}

func TestIsOperator(t *testing.T) {
	require.False(t, INT.IsOperator())
	require.True(t, MUL.IsOperator())
	require.False(t, TRUE.IsOperator())
}

func TestIsKeyword(t *testing.T) {
	require.False(t, INT.IsKeyword())
	require.False(t, MUL.IsKeyword())
	require.True(t, TRUE.IsKeyword())
}
