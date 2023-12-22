package ast_test

import (
	"fmt"
	"testing"

	"github.com/koblas/cedar-go/core/ast"
	"github.com/koblas/cedar-go/core/parser"
	"github.com/stretchr/testify/assert"
)

func jsonTestRunner(t *testing.T, rules string, expect string) {
	policy, err := parser.ParseRules(rules)
	assert.NoError(t, err, "parse rules")

	data, err := ast.ToJson(policy)
	assert.NoError(t, err, "to json")

	fmt.Println("=====")
	fmt.Println(string(data))
	fmt.Println("=====")
	assert.JSONEq(t, expect, string(data))
}

func TestBasic(t *testing.T) {
	jsonTestRunner(t,
		`
	permit(principal, action, resource);
	`,
		`
[
   {
      "effect" : "permit",
      "action" : null,
      "principal" : null,
      "resource" : null
   }
]
	`)
}
