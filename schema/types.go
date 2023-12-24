package schema

import "encoding/json"

type JsonEntityShape struct {
	Type     string `json:"type"`
	Required bool   `json:"required"`
	// Record type (required)
	Attributes map[string]JsonEntityShape `json:"attributes"`
	// Entity or Extension type (required)
	Name *string `json:"name"`
	// Set type (required)
	Element *JsonEntityShape `json:"element"`
}

type JsonEntityType struct {
	MemberOfTypes []string        `json:"memberOfTypes"`
	Shape         JsonEntityShape `json:"shape"`
}

type JsonEntityTypes map[string]JsonEntityType

// Action
type JsonMemberOf struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type JsonAppliesTo struct {
	PrincipalTypes []string         `json:"principalTypes"`
	ResourceTypes  []string         `json:"resourceTypes"`
	Context        *JsonEntityShape `json:"context"`
}

type JsonAction struct {
	MemberOf  []JsonMemberOf `json:"memberOf"`
	AppliesTo *JsonAppliesTo `json:"appliesTo"`
}

type JsonActions map[string]JsonAction

type JsonCommonTypes map[string]JsonEntityShape

type JsonSchemaEntry struct {
	EntityTypes JsonEntityTypes `json:"entityTypes"`
	Actions     JsonActions     `json:"actions"`
	CommonTypes JsonCommonTypes `json:"commonTypes,omitempty"`
}

type JsonSchema map[string]JsonSchemaEntry

// EntityShape UnmarshalJSON -- just so we can default `Required: true`
func (es *JsonEntityShape) UnmarshalJSON(data []byte) error {
	type entityShape struct {
		Type     string `json:"type"`
		Required bool   `json:"required"`
		// Record type (required)
		Attributes map[string]JsonEntityShape `json:"attributes"`
		// Entity or Extension type (required)
		Name *string `json:"name"`
		// Set type (required)
		Element *JsonEntityShape `json:"element"`
	}

	entry := entityShape{
		Required: true,
	}

	if err := json.Unmarshal(data, &entry); err != nil {
		return err
	}

	*es = JsonEntityShape(entry)

	return nil
}

type ShapeType int

const (
	SHAPE_INVALID   ShapeType = iota
	SHAPE_BOOL      ShapeType = iota
	SHAPE_LONG      ShapeType = iota
	SHAPE_STRING    ShapeType = iota
	SHAPE_ENTITY    ShapeType = iota
	SHAPE_SET       ShapeType = iota
	SHAPE_EXTENSION ShapeType = iota
	SHAPE_RECORD    ShapeType = iota
)

type EntityShape struct {
	Type     ShapeType `json:"type"`
	Required bool      `json:"required"`
	// Record type (required)
	Attributes map[string]*EntityShape `json:"attributes"`
	// Entity or Extension type (required)
	Name string
	// Set type (required)
	Element *EntityShape `json:"element"`
}

type EntityType struct {
	MemberOfTypes []string     `json:"memberOfTypes"`
	Shape         *EntityShape `json:"shape"`
}

type MemberOf struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type Action struct {
	MemberOf []MemberOf `json:"memberOf"`

	// This is flatting up the AppliesTo block
	// to help differentiate between the empty set and not present
	HasPrincipalTypes bool
	HasResourceTypes  bool
	PrincipalTypes    map[string]bool
	ResourceTypes     map[string]bool
	Context           *EntityShape `json:"context"`
}

type Schema struct {
	EntityTypes map[string]*EntityType
	Actions     map[string]map[string]*Action
}
