package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/koblas/cedar-go"
	"github.com/koblas/cedar-go/engine"
	"github.com/koblas/cedar-go/parser"
	"github.com/koblas/cedar-go/schema"
)

func main() {
	policyFile := flag.String("policies", "", "file for policy data")
	entityFile := flag.String("entities", "", "file for entities data")
	// contextFile := flag.String("context", "", "file for context data")
	schemaFile := flag.String("schema", "", "file for schema definition")
	principalStr := flag.String("principal", "", "principal entity e.g. User::\"alice\"")
	actionStr := flag.String("action", "", "action entity e.g. Action::\"view\"")
	resourceStr := flag.String("resource", "", "resource entity e.g. Photo::\"VacationPhoto94.jpg\"")

	flag.Parse()

	if *policyFile == "" {
		panic(fmt.Errorf("policy file must provided"))
	}

	policyData, err := os.ReadFile(*policyFile)
	if err != nil {
		panic(fmt.Errorf("unable to read policy file: %w", err))
	}
	policy, err := parser.ParseRules(string(policyData))
	if err != nil {
		panic(fmt.Errorf("unable to parse policies: %w", err))
	}

	var opts []cedar.Option
	var sdef *schema.Schema
	if *schemaFile != "" {
		fd, err := os.Open(*schemaFile)
		if err != nil {
			panic(fmt.Errorf("unable to open schema file: %w", err))
		}
		defer fd.Close()
		sdef, err = schema.NewFromJson(fd)
		if err != nil {
			panic(fmt.Errorf("unable to read schema file: %w", err))
		}

		opts = append(opts, cedar.WithSchema(sdef))
	}

	if *entityFile != "" {
		fd, err := os.Open(*entityFile)
		if err != nil {
			panic(fmt.Errorf("unable to open entity file: %w", err))
		}
		defer fd.Close()

		store, err := cedar.StoreFromJson(fd, sdef)
		if err != nil {
			panic(fmt.Errorf("unable load entities: %w", err))
		}

		opts = append(opts, cedar.WithStore(store))
	}

	auth := cedar.NewAuthorizer(policy, opts...)

	req := cedar.Request{}
	if *principalStr != "" {
		req.Principal = engine.NewEntityFromString(*principalStr)
	}
	if *actionStr != "" {
		req.Action = engine.NewEntityFromString(*actionStr)
	}
	if *resourceStr != "" {
		req.Resource = engine.NewEntityFromString(*resourceStr)
	}

	result, err := auth.IsAuthorized(context.Background(), &req)
	if err != nil {
		panic(fmt.Errorf("unable authorize: %w", err))
	}

	if result {
		fmt.Println("ALLOW")
	} else {
		fmt.Println("DENY")
	}
}
