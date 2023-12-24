package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/koblas/cedar-go/engine"
)

var ErrInvalidSchema = errors.New("invalid schema definition")
var True = true

type Path []string

func (p Path) String() string {
	return strings.Join(p, "::")
}

func (schema JsonSchema) verifyEntityShape(path Path, value JsonEntityShape, lookup JsonCommonTypes) error {
	noAttributes := true
	noElement := true
	noName := true
	switch value.Type {
	case "String", "Long", "Boolean":
		// No to everything
	case "Set":
		if value.Element == nil {
			return fmt.Errorf("%s: element property required: %w", path.String(), ErrInvalidSchema)
		}
		noElement = false
	case "Entity", "Extension":
		if value.Name == nil || *value.Name == "" {
			return fmt.Errorf("%s: name property required: %w", path.String(), ErrInvalidSchema)
		}
		noName = false
	case "Record", "":
		// empty string "" is a non-set defintion assume Record
		// Attributes can be empty
		noAttributes = false
	default:
		found := false
		if lookup != nil {
			_, found = lookup[value.Type]
		}
		if !found {
			return fmt.Errorf("%s: unknown type name %s: %w", path.String(), value.Type, ErrInvalidSchema)
		}
	}

	if noAttributes && len(value.Attributes) != 0 {
		return fmt.Errorf("%s: attributes property defined: %w", path.String(), ErrInvalidSchema)
	}
	if noName && value.Name != nil && *value.Name != "" {
		return fmt.Errorf("%s: name property defined: %w", path.String(), ErrInvalidSchema)
	}
	if noElement && value.Element != nil {
		return fmt.Errorf("%s: element property defined: %w", path.String(), ErrInvalidSchema)
	}

	return nil
}

func (schema JsonSchema) verifyEntityType(path Path, value JsonEntityType, lookup JsonCommonTypes, etypes JsonEntityTypes) error {
	if err := schema.verifyEntityShape(path, value.Shape, lookup); err != nil {
		return err
	}

	for _, value := range value.MemberOfTypes {
		if _, found := etypes[value]; !found {
			return fmt.Errorf("%s: memberOf property non-existant: %s: %w", path.String(), value, ErrInvalidSchema)
		}
	}

	return nil
}

func (schema JsonSchema) verifyAction(path []string, value JsonAction, lookup JsonCommonTypes) error {
	if value.AppliesTo != nil && value.AppliesTo.Context != nil {
		if err := schema.verifyEntityShape(append(path, "context"), *value.AppliesTo.Context, lookup); err != nil {
			return err
		}
	}

	return nil
}

func hasWhiteSpace(str string) bool {
	for _, ch := range str {
		if unicode.IsSpace(ch) {
			return true
		}
	}

	return false
}

func (schema JsonSchema) verifyNamespace(path Path, value JsonSchemaEntry) error {
	// if len(value.EntityTypes) == 0 {
	// 	return fmt.Errorf("%s: no entityTypes defined: %w", path.String(), ErrInvalidSchema)
	// }
	// if len(value.Actions) == 0 {
	// 	return fmt.Errorf("%s: no actions defined: %w", path.String(), ErrInvalidSchema)
	// }

	for name, entity := range value.CommonTypes {
		path := append(path, "commonTypes", name)
		if hasWhiteSpace(name) {
			return fmt.Errorf("%s: whitespace in commonTypes name: %w", path.String(), ErrInvalidSchema)
		}
		err := schema.verifyEntityShape(path, entity, nil)
		if err != nil {
			return err
		}
	}
	for name, entity := range value.EntityTypes {
		path := append(path, "entityTypes", name)
		if hasWhiteSpace(name) {
			return fmt.Errorf("%s: whitespace in entityTypes name: %w", path.String(), ErrInvalidSchema)
		}
		err := schema.verifyEntityType(path, entity, value.CommonTypes, value.EntityTypes)
		if err != nil {
			return err
		}
	}
	for name, action := range value.Actions {
		path := append(path, "actions", name)
		//  action names can have whitespace
		err := schema.verifyAction(append(path, "actions", name), action, value.CommonTypes)
		if err != nil {
			return err
		}
	}

	return nil
}

func (schema *JsonSchema) VerifyConsistency() error {
	// Verify the common types first
	for namespace, value := range *schema {
		if hasWhiteSpace(namespace) {
			return fmt.Errorf("%s: whitespace in namespace: %w", namespace, ErrInvalidSchema)
		}
		if err := schema.verifyNamespace(Path{namespace}, value); err != nil {
			return err
		}
	}
	return nil
}

func NewEmptySchema() *Schema {
	return &Schema{}
}

type commonDefs map[string]*EntityShape

func namespaceName(namespace string, name string) string {
	if namespace == "" {
		return name
	}
	if strings.Contains(name, "::") {
		return name
	}
	return namespace + "::" + name
}

func namespaceTypes(namespace string, names []string) []string {
	if names == nil {
		return nil
	}
	output := []string{}
	if namespace == "" {
		return names
	}
	for _, item := range names {
		if strings.Contains(item, "::") {
			output = append(output, item)
		} else {
			output = append(output, namespace+"::"+item)
		}
	}
	return output
}

// processEntityShape converts the Json definition to a runtime definition, this will also complete
// all lookups of the type names to flatten out the schema
func processEntityShape(ekey string, namespace string, input JsonEntityShape, common commonDefs) (*EntityShape, error) {
	shape := EntityShape{}

	switch input.Type {
	case "String":
		shape.Type = SHAPE_STRING
	case "Long":
		shape.Type = SHAPE_LONG
	case "Boolean":
		shape.Type = SHAPE_BOOL
		// Do nothing
	case "Set":
		shape.Type = SHAPE_SET
		elem, err := processEntityShape(ekey, namespace, *input.Element, common)
		if err != nil {
			return nil, err
		}
		shape.Element = elem
	case "Entity":
		shape.Type = SHAPE_ENTITY
		shape.Name = namespaceName(namespace, *input.Name)
	case "Extension":
		shape.Type = SHAPE_EXTENSION
		shape.Name = *input.Name
	case "Record":
		shape.Type = SHAPE_RECORD
		shape.Attributes = map[string]*EntityShape{}
		for key, value := range input.Attributes {
			elem, err := processEntityShape(ekey, namespace, value, common)
			if err != nil {
				return nil, err
			}
			shape.Attributes[key] = elem
		}
	default:
		if input.Type == "" {
			shape.Type = SHAPE_RECORD
			shape.Attributes = map[string]*EntityShape{}
		} else {
			lookup, found := common[input.Type]
			if !found {
				return nil, fmt.Errorf("%s: unknown type name %s: %w", ekey, input.Type, ErrInvalidSchema)
			}
			return lookup, nil
		}
	}

	return &shape, nil
}

func processEntityType(ekey string, namespace string, input JsonEntityType, common commonDefs) (*EntityType, error) {
	output := EntityType{
		MemberOfTypes: namespaceTypes(namespace, input.MemberOfTypes),
	}

	shape, err := processEntityShape(ekey, namespace, input.Shape, common)
	if err != nil {
		return nil, err
	}
	output.Shape = shape

	return &output, nil
}

func processAction(ekey string, namespace string, input JsonAction, common commonDefs) (*Action, error) {
	output := Action{}

	for _, item := range input.MemberOf {
		output.MemberOf = append(output.MemberOf, MemberOf{
			Id:   item.Id,
			Type: namespaceName(namespace, item.Type),
		})
	}
	if input.AppliesTo != nil {
		if input.AppliesTo.PrincipalTypes != nil {
			output.HasPrincipalTypes = True
			output.PrincipalTypes = map[string]bool{}
			for _, item := range input.AppliesTo.PrincipalTypes {
				output.PrincipalTypes[namespaceName(namespace, item)] = true
			}
		}
		if input.AppliesTo.ResourceTypes != nil {
			output.HasResourceTypes = True
			output.ResourceTypes = map[string]bool{}
			for _, item := range input.AppliesTo.ResourceTypes {
				output.ResourceTypes[namespaceName(namespace, item)] = true
			}
		}

		if input.AppliesTo.Context != nil {
			shape, err := processEntityShape(ekey, namespace, *input.AppliesTo.Context, common)
			if err != nil {
				return nil, err
			}
			output.Context = shape
		}
	}

	return &output, nil
}

func processCommonDefs(ekey string, namespace string, common JsonCommonTypes) (commonDefs, error) {
	defs := commonDefs{}

	if len(common) == 0 {
		return defs, nil
	}

	// TODO - unclear from specification if there may be dependancies
	// within a type.
	for key, entry := range common {
		shape, err := processEntityShape(ekey, namespace, entry, defs)
		if err != nil {
			return nil, err
		}
		defs[key] = shape
	}

	return defs, nil
}

func processSchema(input *JsonSchema) (*Schema, error) {
	output := Schema{
		EntityTypes: map[string]*EntityType{},
		Actions:     map[string]map[string]*Action{},
	}

	for ns, item := range *input {
		prefix := ""
		if ns != "" {
			prefix = ns + "::"
		}

		common, err := processCommonDefs(ns+"::commonTypes", ns, item.CommonTypes)
		if err != nil {
			return nil, err
		}

		for key, value := range item.EntityTypes {
			entity, err := processEntityType(ns+"::entityTypes", ns, value, common)
			if err != nil {
				return nil, err
			}

			output.EntityTypes[prefix+key] = entity
		}

		acts := map[string]*Action{}
		for key, value := range item.Actions {
			action, err := processAction(ns+"::actions", ns, value, common)
			if err != nil {
				return nil, err
			}
			acts[prefix+key] = action
		}
		output.Actions[ns] = acts
	}

	return &output, nil
}

//
//

func LoadSchema(reader io.Reader) (*Schema, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	jschema := &JsonSchema{}

	if err := json.Unmarshal(data, &jschema); err != nil {
		return nil, err
	}
	if err := jschema.VerifyConsistency(); err != nil {
		return nil, err
	}

	schema, err := processSchema(jschema)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

func (schema *Schema) FindDef(entity engine.EntityValue) (*EntityShape, error) {
	length := len(entity)
	if length < 2 {
		return nil, fmt.Errorf("%s: %w", entity.String(), ErrInvalidEntityFormat)
	}

	path := entity[0 : length-1]

	def, ok := schema.EntityTypes[strings.Join(path, "::")]
	if !ok {
		return nil, nil
	}
	return def.Shape, nil
}
