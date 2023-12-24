package main

import (
	"context"
	"fmt"

	"github.com/koblas/cedar-go"
	"github.com/koblas/cedar-go/engine"
	"github.com/koblas/cedar-go/parser"
)

var POLICY_SRC = `
permit(
	principal == User::"alice", 
	action == Action::"view", 
	resource == File::"93"
);
`

// This is the example from the Rust Crate
func main() {
	policy, err := parser.ParseRules(POLICY_SRC)
	if err != nil {
		panic(err)
	}

	alice := engine.NewEntityValue("User", "alice")
	action := engine.NewEntityValue("Action", "view")
	file := engine.NewEntityValue("File", "93")

	request := cedar.Request{
		Principal: alice,
		Resource:  file,
		Action:    action,
	}

	auth := cedar.NewAuthorizer(policy)

	answer, err := auth.IsAuthorized(context.TODO(), &request)
	if err != nil {
		panic(err)
	}

	// Should give us ALLOW
	fmt.Println(alice, answer)

	bob := engine.NewEntityValue("User", "bob")

	request = cedar.Request{
		Principal: bob,
		Resource:  file,
		Action:    action,
	}

	answer, err = auth.IsAuthorized(context.TODO(), &request)
	if err != nil {
		panic(err)
	}

	// Should give us DENY
	fmt.Println(bob, answer)
}
