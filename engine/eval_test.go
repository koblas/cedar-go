package engine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/koblas/cedar-go"
	ast "github.com/koblas/cedar-go/engine"
	"github.com/koblas/cedar-go/parser"
	"github.com/koblas/cedar-go/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var emptyRequest = &cedar.Request{
	Principal: ast.EntityValue{},
	Resource:  ast.EntityValue{},
	Action:    ast.EntityValue{},
}

var bobRequest = &cedar.Request{
	Principal: ast.NewEntityValue("User", "bob"),
}

func evalTestRunner(t *testing.T, rules string, req *cedar.Request, allow bool) {
	policy, err := parser.ParseRules(rules)
	assert.NoError(t, err, "parse rules")

	data, err := cedar.NewAuthorizer(policy).IsAuthorized(context.TODO(), req)
	assert.NoError(t, err)

	if allow {
		assert.Equal(t, true, data)
	} else {
		assert.Equal(t, false, data)
	}
}

func evalTestError(t *testing.T, rules string, req *cedar.Request) {
	policy, err := parser.ParseRules(rules)
	assert.NoError(t, err, "parse rules")

	data, err := cedar.NewAuthorizer(policy).IsAuthorized(context.TODO(), req)
	assert.Error(t, err)
	assert.Equal(t, false, data)
}

func TestEvalBasic(t *testing.T) {
	evalTestRunner(t,
		`
	permit(principal, action, resource);
	`, emptyRequest, true)
}

func TestEvalBool(t *testing.T) {
	evalTestRunner(t,
		`
	permit(principal, action, resource) 
	unless {
		!true || false && (8 > 9)
	};
	`, emptyRequest, true)
}

func TestEvalEntity(t *testing.T) {
	evalTestRunner(t,
		`
	permit(principal, action, resource) 
	unless {
		User::"bob" == User::"alice"
	} when {
		User::"bob" == User::"bob"
	};
	`, emptyRequest, true)
}

func TestEvalEntity2(t *testing.T) {
	evalTestRunner(t,
		`
	permit(principal, action, resource) 
	unless {
		principal == User::"alice"
	} when {
		principal == User::"bob"
	};
	`, bobRequest, true)
}

func TestEvalEntityMathFalse(t *testing.T) {
	request := cedar.Request{
		Principal: ast.NewEntityValue("a", "ff\000"),
		Resource:  ast.NewEntityValue("a", "ff\000"),
		Action:    ast.NewEntityValue("Action", `action`),
	}
	evalTestError(t, `
	forbid(
		principal,
		action == Action::"action",
		resource
	      ) when {
		true && (((3941264106 * 0) * 0) * 0)
	      };
	`, &request)
}

func TestEvalEntityEval(t *testing.T) {
	entityData := schema.JsonEntities{
		{
			Uid: schema.JsonEntityValue{
				"type": "User",
				"id":   "stacy",
			},
			Attrs: map[string]any{
				"jobLevel": 8,
			},
		},
	}

	principal := ast.NewEntityValue("User", "stacy")
	resource := ast.NewEntityValue("Photo", "vacation.jpg")
	action := ast.NewEntityValue("Action", "view")

	schema := schema.NewEmptySchema()
	store, _ := schema.NormalizeEntites(entityData)
	ctxData, _ := schema.NormalizeContext(map[string]any{
		"authenticated": true,
	}, principal, action, resource)

	req := cedar.Request{
		Principal: ast.NewEntityValue("User", "stacy"),
		Resource:  ast.NewEntityValue("Photo", "vacation.jpg"),
		Action:    ast.NewEntityValue("Action", "view"),
		Context:   ctxData,
	}

	rules := `
	      permit (
		principal,
		action,
		resource in Album::"alice_vacation"
	      )
	      when { principal.department == "Sales" };
	      
	      // Deny all requests that have context.authenticated set to false
	      forbid (principal, action, resource)
	      unless { context.authenticated };
	      
	      // Users with job level >= 7 can view any resource
	      permit (
		principal,
		action == Action::"view",
		resource
	      )
	      when { principal.jobLevel >= 7 };
	`

	policy, err := parser.ParseRules(rules)
	assert.NoError(t, err, "parse rules")

	data, err := cedar.NewAuthorizer(policy, cedar.WithStore(store)).IsAuthorized(context.TODO(), &req)
	assert.NoError(t, err)

	assert.Equal(t, true, data)
}

func TestOps(t *testing.T) {
	base := `permit(principal is User, action, resource) when { %s };`

	entityData := schema.JsonEntities{
		{
			Uid: schema.JsonEntityValue{
				"type": "User",
				"id":   "alice",
			},
			Attrs: map[string]any{
				"active":    true,
				"disabled":  false,
				"level":     5,
				"breakfast": "ham and eggs",
			},
		},
	}

	// schema, err := schema.LoadSchema(bytes.NewReader(entityData))
	// require.NoError(t, err, "load schema")
	schema := schema.NewEmptySchema()

	store, _ := schema.NormalizeEntites(entityData)

	req := cedar.Request{
		Principal: ast.NewEntityValue("User", "alice"),
		Resource:  ast.NewEntityValue("Photo", "vacation.jpg"),
		Action:    ast.NewEntityValue("Action", "view"),
	}

	expressions := []struct {
		expr   string
		expect bool
	}{
		{"true", true},
		{"false", false},
		{"!false", true},
		{"1 == 1", true},
		{"1 != 2", true},
		{"1 < 2", true},
		{"1 <= 2", true},
		{"2 <= 2", true},
		{"2 > 1", true},
		{"2 >= 1", true},
		{"2 >= 2", true},
		{"true && true", true},
		{"true && false", false},
		{"true || true", true},
		{"true || false", true},
		{"1 + 1 == 2", true},
		{"1 - 1 == 0", true},
		{"1 * 1 == 1", true},
		{"1 - 2 == -1", true},
		// Not in language
		// {"1 / 1 == 1", true},
		// {"3 % 2 == 1", true},
		{"if 1 == 1 then true else false", true},
		{"if 1 != 1 then false else true", true},
		{`User::"alice" == User::"alice"`, true},
		{`User::"alice" in User::"alice"`, true},
		{`User::"alice" in [ User::"alice", User::"bob" ]`, true},
		{"principal has active", true},
		{"!(principal has madeup)", true},
		{"principal.active", true},
		// {`principal.breakfast like "*zz*"`, true},
		{`"ham and eggs" like "ham*"`, true},
		// Make sure strings work
		{`principal.breakfast == "ham and eggs"`, true},
		// Make sure integers parse
		{`principal.level <= 7`, true},
		{"[1,2] == [2,1]", true},
	}

	for _, item := range expressions {
		t.Run(item.expr, func(t *testing.T) {
			policy, err := parser.ParseRules(fmt.Sprintf(base, item.expr))
			require.NoError(t, err, "parse rules")

			auth := cedar.NewAuthorizer(policy, cedar.WithStore(store))
			result, err := auth.IsAuthorized(context.TODO(), &req)
			require.NoError(t, err, "authorizer")

			require.EqualValues(t, item.expect, result)
		})
	}
}

func TestFuncCall(t *testing.T) {
	entityData := schema.JsonEntities{
		{
			Uid: schema.JsonEntityValue{
				"type": "User",
				"id":   "alice",
			},
			Attrs: map[string]any{
				"active":       true,
				"disabled":     false,
				"level":        5,
				"breakfengine": "ham and eggs",
			},
		},
	}

	// schema, err := schema.LoadSchema(bytes.NewReader(entityData))
	// require.NoError(t, err, "load schema")
	schema := schema.NewEmptySchema()

	store, _ := schema.NormalizeEntites(entityData)

	req := cedar.Request{
		Principal: ast.NewEntityValue("User", "alice"),
		Resource:  ast.NewEntityValue("Photo", "vacation.jpg"),
		Action:    ast.NewEntityValue("Action", "view"),
	}

	policy, err := parser.ParseRules(`
	permit(principal is User, action, resource) 
	when { 
		ip("1.2.3.4").isIpv4()
	};
	`)
	require.NoError(t, err, "parse rules")

	auth := cedar.NewAuthorizer(policy, cedar.WithStore(store))
	result, err := auth.IsAuthorized(context.TODO(), &req)
	require.NoError(t, err, "authorizer")

	require.True(t, result)
}
