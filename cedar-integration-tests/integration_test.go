package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	cedar "github.com/koblas/cedar-go"
	"github.com/koblas/cedar-go/engine"
	"github.com/koblas/cedar-go/parser"
	"github.com/koblas/cedar-go/schema"
	"github.com/stretchr/testify/require"
)

//go:embed corpus_tests/* tests/* sample-data/*
var content embed.FS

func findAllJson() ([]string, error) {
	var paths []string

	err := fs.WalkDir(content, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		paths = append(paths, path)

		return nil
	})

	return paths, err
}

type SpecQuery struct {
	Description string `json:"desc"`
	// TODO context
	Decision string   `json:"decision"`
	Errors   []string `json:"errors"`
	Reasons  []string `json:"reasons"`

	Principal engine.EntityValue `json:"principal"`
	Resource  engine.EntityValue `json:"resource"`
	Action    engine.EntityValue `json:"action"`
	Context   map[string]any     `json:"context"`
}

type SpecDef struct {
	Schema         string `json:"schema"`
	Policies       string `json:"policies"`
	ShouldValidate bool   `json:"should_validate"`
	Entities       string `json:"entities"`
	Queries        []SpecQuery
}

func runTests(t *testing.T, dir string) {
	paths, err := findAllJson()
	require.NoError(t, err)

	passCount := 0
	total := 0
	for _, path := range paths {
		if !strings.Contains(path, dir) {
			continue
		}
		if strings.Contains(path, "schema_") || strings.Contains(path, "schema.") {
			continue
		}
		total += 1
		t.Run(path, func(t *testing.T) {
			data, err := content.ReadFile(path)
			require.NoError(t, err)

			spec := SpecDef{}
			err = json.Unmarshal(data, &spec)
			require.NoError(t, err)

			if spec.Policies == "" {
				fmt.Println("SKIP === ", path)
				// Ignore anything that isn't a valid defintion
				return
			}

			// Ignored tests
			// This test fails parsing, but is ok in Cedar-Rust
			// if strings.Contains(spec.Policies, "policies_0197f1506d505e1f9364e4379bbeeb8d8b38d482") {
			// 	return
			// }

			readFile := func(name string) ([]byte, error) {
				name = strings.TrimPrefix(name, "./")
				return content.ReadFile(name)
			}

			policyData, err := readFile(spec.Policies)
			require.NoError(t, err, "failed to read policies")
			entityData, err := readFile(spec.Entities)
			require.NoError(t, err, "failed to read entities")
			schemaData, err := readFile(spec.Schema)
			require.NoError(t, err, "failed to read schema")

			// debugDump := func(desc string) {
			// 	fmt.Println("=============================")
			// 	fmt.Println("Spec: ", path)
			// 	fmt.Println("Policy: ", spec.Policies)
			// 	fmt.Println("Schema: ", spec.Schema)
			// 	fmt.Println("Entities: ", spec.Entities)
			// 	fmt.Println("TestCase: ", desc)
			// 	fmt.Println("POLICY", string(policyData))
			// 	fmt.Println("")
			// }

			entities := schema.JsonEntities{}
			err = json.Unmarshal(entityData, &entities)
			require.NoError(t, err, "failed to parse entities")

			policy, err := parser.ParseRules(string(policyData))
			require.NoError(t, err, "failed to parse policies")

			schema, err := schema.NewFromJson(bytes.NewReader(schemaData))
			require.NoError(t, err, "failed to parse schema")

			store, err := schema.NormalizeEntites(entities)
			require.NoError(t, err, "failed to load store - parse entities")

			auth := cedar.NewAuthorizer(policy, cedar.WithSchema(schema), cedar.WithStore(store))

			for _, query := range spec.Queries {
				t.Run(query.Description, func(t *testing.T) {
					principal := query.Principal
					resource := query.Resource
					action := query.Action

					qcontext, err := schema.NormalizeContext(query.Context, principal, action, resource)
					require.NoError(t, err, "normalize context")

					// fmt.Println("==== ", query.Description, path)
					request := cedar.Request{
						Principal: principal,
						Resource:  resource,
						Action:    action,
						Context:   qcontext,
					}

					result, err := auth.IsAuthorizedDetail(context.TODO(), &request)

					// if err == nil && !spec.ShouldValidate {
					// 	debugDump()
					// 	fmt.Println("POLICY: SHOULD HAVE NOT VALIDATED", spec.Queries[0].Errors)
					// 	assert.Error(t, policyErr, "policy should not have validated")
					// }

					// if hasErr != expectErr {
					// 	debugDump(query.Description)
					// 	if expectErr {
					// 		fmt.Println("EXPECTED ERRORS: ", query.Errors)
					// 	} else {
					// 		fmt.Println("GOT ERROR", err)
					// 	}
					// }

					resultStr := "Allow"
					if result == nil || !result.IsAllowed {
						resultStr = "Deny"
					}
					require.Equal(t, query.Decision, resultStr)
					if resultStr == "Allow" {
						require.NoError(t, err)
						require.NotNil(t, result, "Result is nil")

						require.EqualValues(t, len(query.Reasons), len(result.Matches), "policy matches")
					}

					// Whether the given policies are expected to pass the validator with this schema, or not
					// if spec.ShouldValidate {
				})

			}

			passCount += 1
		})
	}

	// fmt.Printf("pass=%d  fail=%d  total=%d\n", passCount, total-passCount, total)
}

// Directory by directory runners to make it easier to debug specific failing tests

func TestCorpus(t *testing.T) {
	runTests(t, "corpus_tests/")
}

func TestDecimal(t *testing.T) {
	runTests(t, "tests/decimal/")
}

func TestExampleUseCases(t *testing.T) {
	runTests(t, "tests/example_use_cases_doc/")
}

func TestIp(t *testing.T) {
	runTests(t, "tests/ip/")
}

func TestMulti(t *testing.T) {
	runTests(t, "tests/multi/")
}

// ------ pull outs for debugging

func TestExampe4c(t *testing.T) {
	runTests(t, "tests/example_use_cases_doc/4c.json")
}

func TestExampe3(t *testing.T) {
	runTests(t, "tests/multi/3.json")
}

func TestNullTest(t *testing.T) {
	runTests(t, "corpus_tests/226abd401950859a4fc3fe5f82182ed5cd403d17.json")
}
