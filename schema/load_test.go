package schema_test

import (
	"strings"
	"testing"

	"github.com/koblas/cedar-go/schema"
	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	reader := strings.NewReader(`
	{
		"": {
		  "commonTypes": {},
		  "entityTypes": {
		    "a": {
		      "memberOfTypes": [],
		      "shape": {
			"type": "Record",
			"attributes": {},
			"additionalAttributes": false
		      }
		    }
		  },
		  "actions": {
		    "action": {
		      "appliesTo": {
			"resourceTypes": [
			  "a"
			],
			"principalTypes": [
			  "a"
			],
			"context": {
			  "type": "Record",
			  "attributes": {},
			  "additionalAttributes": false
			}
		      },
		      "memberOf": null
		    }
		  }
		}
	}
	`)

	s, err := schema.NewFromJson(reader)

	assert.NoError(t, err)
	assert.EqualValues(t, len(s.EntityTypes), 1)
}

func TestBasicError(t *testing.T) {
	reader := strings.NewReader(`
	{
		"": {
		  "commonTypes": {},
		  "entityTypes": {
		    "a b": {
		      "memberOfTypes": [],
		      "shape": {
			"type": "Record",
			"attributes": {},
			"additionalAttributes": false
		      }
		    }
		  },
		  "actions": {
		    "action": {
		      "appliesTo": {
			"resourceTypes": [
			  "a"
			],
			"principalTypes": [
			  "a"
			],
			"context": {
			  "type": "Record",
			  "attributes": {},
			  "additionalAttributes": false
			}
		      },
		      "memberOf": null
		    }
		  }
		}
	}
	`)

	_, err := schema.NewFromJson(reader)

	assert.Error(t, err)
}
