# Cedar Go

[![Go Report Card](https://goreportcard.com/badge/github.com/koblas/cedar-go)](https://goreportcard.com/report/github.com/koblas/cedar-go)
[![Build Status](https://github.com/koblas/cedar-go/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/koblas/cedar-go/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/koblas/cedar-go)](https://pkg.go.dev/github.com/koblas/cedar-go)

<!-- [![CII Best Practices](https://bestpractices.coreinfrastructure.org/TODO)](https://bestpractices.coreinfrastructure.org/TODO) -->

Cedar Go is a pure Go implementation of the [Cedar](https://www.cedarpolicy.com/) policy language.

## TL;DR

- This implements the policy runtime completly based on the 3.0 specification
- The schema validator is focused on converting JSON reprentations, the deep validation of policies with entites is not supported
- The full suite of corpus tests is passing

## Quick Start -- integration

```
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/koblas/cedar-go"
)

const ENTITIES = `
[
   {
      "uid" : {
         "id" : "vacation.jpg",
         "type" : "Photo"
      },
      "attrs" : {
         "owner" : {
            "__entity" : {
               "id" : "alice",
               "type" : "User"
            }
         }
      }
   }
]
`

const POLICIES = `
    permit(
        principal,
        action == Action::"view",
        resource is Photo)
    when {
        principal == resource.owner
    };
`

func main() {
	policies, err := cedar.ParsePolicies(POLICIES)
	if err != nil {
		panic(fmt.Errorf("unable to parse policies: %w", err))
	}

	store, err := cedar.StoreFromJson(strings.NewReader(ENTITIES), nil)
	if err != nil {
		panic(fmt.Errorf("unable to parse entities: %w", err))
	}

	req := cedar.Request{
		Principal: cedar.NewEntity("User", "alice"),
		Action:    cedar.NewEntity("Action", "view"),
		Resource:  cedar.NewEntity("Photo", "vacation.jpg"),
	}

	auth := cedar.NewAuthorizer(policies, cedar.WithStore(store))

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
```

## Quick Start -- command line

Let's put the policy in `policy.cedar` and the entities in `entities.json`.

`policy.cedar`:

```cedar
permit (
  principal == User::"alice",
  action == Action::"view",
  resource in Album::"jane_vacation"
);
```

This policy specifies that `alice` is allowed to view the photos in the `"jane_vacation"` album.

`entities.json`:

```json
[
  {
    "uid": { "type": "User", "id": "alice" },
    "attrs": { "age": 18 },
    "parents": []
  },
  {
    "uid": { "type": "Photo", "id": "VacationPhoto94.jpg" },
    "attrs": {},
    "parents": [{ "type": "Album", "id": "jane_vacation" }]
  }
]
```

Cedar represents principals, resources, and actions as entities. An entity has a type (e.g., `User`) and an id (e.g., `alice`). They can also have attributes (e.g., `User::"alice"`'s `age` attribute is the integer `18`).

Now, let's test our policy with the CLI:

```sh
go run cmd/main.go \
  --policies example/data/policy.cedar \
  --entities example/data/entities.json \
  --principal 'User::"alice"' \
  --action 'Action::"view"' \
  --resource 'Photo::"VacationPhoto94.jpg"'
```

CLI output:

```
ALLOW
```

This request is allowed because `VacationPhoto94.jpg` belongs to `Album::"jane_vacation"`, and `alice` can view photos in `Album::"jane_vacation"`.

If you'd like to see more details on what can be expressed as Cedar policies, see the [documentation](https://docs.cedarpolicy.com).

Examples of how to use Cedar in an application are contained in the repository [cedar-examples](https://github.com/cedar-policy/cedar-examples). [TinyTodo](https://github.com/cedar-policy/cedar-examples/tree/main/tinytodo) is a simple task list management app whose users' requests, sent as HTTP messages, are authorized by Cedar. It shows how you can integrate Cedar into your own Rust program.

## Extending

One of the original objectives for this project was to make it easier to extend the entity store rather than rely on the
JSON input format from the Rust project. This is accomplished via having a standard Go interface for type resolution.

### Store

There is a standard interface that can be implemented to provide custom storage solutions for
entities rather than JSON based formats

### Types

The type system and functions can be extended as well by implemention some basic interfaces. This
has the ability to also implement operator overloads for types (why did cedar not solve this when
then added `decimal`?)

## Differences from Rust implementation

- Error messages are similar but different due to compiler and runtime differences
- Schema validation is not as strict as the the standard requires
- Transitive dependancies (`in`) are computed at runtime rather than loading

# License

This code is licensed as per the `LICENSE` file, however the integration tests are
copied from the cedar projects and covered by the `LICENSE_cedar.txt` and some parts
of the parser are based on the Go parser covered by `LICENSE_go.txt`
