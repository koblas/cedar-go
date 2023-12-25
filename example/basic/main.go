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
