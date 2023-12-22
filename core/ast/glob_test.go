package ast_test

import (
	"fmt"
	"testing"

	"github.com/koblas/cedar-go/core/ast"
	"github.com/stretchr/testify/assert"
)

func TestGlob(t *testing.T) {
	cases := []struct {
		value   string
		pattern string
		expect  bool
	}{
		{"eggs", "ham*", false},
		{"eggs", "*ham", false},
		{"eggs", "*ham*", false},
		{"ham and eggs", "ham*", true},
		{"ham and eggs", "*ham", false},
		{"ham and eggs", "*ham*", true},
		{"ham and eggs", "*h*a*m*", true},
		{"eggs and ham", "ham*", false},
		{"eggs and ham", "*ham", true},
		{"eggs, ham, and spinach", "ham*", false},
		{"eggs, ham, and spinach", "*ham", false},
		{"eggs, ham, and spinach", "*ham*", true},
		{"Gotham", "ham*", false},
		{"Gotham", "*ham", true},
		{"ham", "ham", true},
		{"ham", "ham*", true},
		{"ham", "*ham", true},
		{"ham", "*h*a*m*", true},
		{"ham and ham", "ham*", true},
		{"ham and ham", "*ham", true},
		{"ham", "*ham and eggs*", false},
		{"\\afterslash", "\\\\*", true},
		{"string\\with\\backslashes", "string\\\\with\\\\backslashes", true},
		{"string\\with\\backslashes", "string*with*backslashes", true},
		{"string*with*stars", "string\\*with\\*stars", true},
	}

	for _, item := range cases {
		t.Run(fmt.Sprintf("%s like %s", item.value, item.pattern), func(t *testing.T) {
			res := ast.Glob(item.pattern, item.value)

			assert.Equal(t, item.expect, res)
		})
	}
}
