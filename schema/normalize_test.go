package schema_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/koblas/cedar-go/engine"
	"github.com/koblas/cedar-go/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSmoke(t *testing.T) {
	data := struct {
		ONe string `cedar:"oNe"`
		Two int
	}{
		ONe: "test",
		Two: 2,
	}

	data2 := map[string]any{
		"oNe": "test",
		"two": 2,
	}

	data4 := map[string]any{
		"oNe": "test",
		"twO": 2,
	}

	schema := schema.NewEmptySchema()

	var1, err := schema.NormalizeContext(data, nil, nil, nil)
	assert.NoError(t, err)
	var2, err := schema.NormalizeContext(&data, nil, nil, nil)
	assert.NoError(t, err)
	var3, err := schema.NormalizeContext(data2, nil, nil, nil)
	assert.NoError(t, err)
	var4, err := schema.NormalizeContext(data4, nil, nil, nil)
	assert.NoError(t, err)

	assert.EqualValues(t, var1, var2)
	assert.EqualValues(t, var1, var2)
	assert.EqualValues(t, var1, var3)
	assert.NotEqualValues(t, var1, var4)
}

func TestNormalizerBase(t *testing.T) {
	entitesData := `
	[
		{
		"uid": {
		  "type": "Photo",
		  "id": "prototype_v0.jpg"
		},
		"attrs": {
		  "private": false,
		  "account": {
		    "type" : "Account",
		    "id": "ahmad"
		  },
		  "admins": []
		},
		"parents": [
		  {
		    "type": "Album",
		    "id": "device_prototypes"
		  }
		]
	      }
	]
	`

	schemaData := `
	{ "": {
		"entityTypes": {

	      "Photo": {
		"shape": {
		  "type": "Record",
		  "additionalAttributes": false,
		  "attributes": {
		    "private": {
		      "type": "Boolean",
		      "required": true
		    },
		    "account": {
		      "type": "Entity",
		      "name": "Account",
		      "required": true
		    },
		    "admins": {
		      "type": "Set",
		      "element": {
			"type": "Entity",
			"name": "User"
		      },
		      "required": true
		    }
		  }
		},
		"memberOfTypes": [
		]
	      }
		}

	} }
	`

	sch, err := schema.LoadSchema(strings.NewReader(schemaData))
	require.NoError(t, err)

	entites := schema.JsonEntities{}
	err = json.Unmarshal([]byte(entitesData), &entites)
	require.NoError(t, err)

	store, err := sch.NormalizeEntites(entites)
	require.NoError(t, err)

	value, err := store.Get(engine.NewEntityValue("Photo", "prototype_v0.jpg"))
	require.NoError(t, err)

	vval := value.(*engine.VarValue)
	eval, err := vval.OpLookup(engine.IdentifierValue("account"))
	require.NoError(t, err)

	require.EqualValues(t, "entity", eval.TypeName())
}
