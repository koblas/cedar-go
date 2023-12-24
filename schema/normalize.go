package schema

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/koblas/cedar-go/engine"
)

func unwrapInterface(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Interface {
		return v.Elem()
	}
	return v
}

func walkSlice(path string, v reflect.Value, shape *EntityShape) (engine.NamedType, error) {
	// Prefer empty list over nil
	result := engine.SetValue{}
	for i := 0; i < v.Len(); i++ {
		v, err := walkValue(fmt.Sprintf("%s.%d", path, i), v.Index(i), shape)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

func specialExtension(path string, name string, v reflect.Value) (engine.NamedType, error) {
	v = unwrapInterface(v)

	var fn string
	var value string

	if v.Kind() == reflect.String && name != "" {
		fn = name
		value = v.String()
	} else if v.Kind() != reflect.Map {
		return nil, fmt.Errorf("%s: expected map got %s for entity: %w", path, v.Kind().String(), ErrInvalidEntityFormat)
	} else {
		byKey := map[string]reflect.Value{}

		// only check for __entity once at the top level, don't allow for
		// __entity.__entity.__entity.id
		for iter := v.MapRange(); iter.Next(); {
			byKey[unwrapInterface(iter.Key()).String()] = iter.Value()
		}

		if name != "" {
			if val, found := byKey["__extn"]; found {
				return specialExtension(path, "", val)
			}
		}

		fnKey, found := byKey["fn"]
		if !found {
			return nil, fmt.Errorf("%s: missing fn field for extension: %w", path, ErrInvalidEntityFormat)
		}
		arg, found := byKey["arg"]
		if !found {
			return nil, fmt.Errorf("%s: missing arg field for extension: %w", path, ErrInvalidEntityFormat)
		}
		fnKey = unwrapInterface(fnKey)
		arg = unwrapInterface(arg)

		if fnKey.Kind() != reflect.String {
			return nil, fmt.Errorf("%s: 'fn' type not string got %s for extension: %w", path, fnKey.Kind().String(), ErrInvalidEntityFormat)
		}
		if arg.Kind() != reflect.String {
			return nil, fmt.Errorf("%s: 'arg' type not string got %s for extension: %w", path, arg.Kind().String(), ErrInvalidEntityFormat)
		}

		fn = fnKey.String()
		value = arg.String()
	}

	switch fn {
	case "ip", "ipaddr":
		return engine.NewIpValue(value)
	case "decimal":
		return engine.NewDecimalValue(value)
	}
	return nil, fmt.Errorf("unknown extension type %s: %w", fn, ErrInvalidEntityFormat)
}

func specialEntity(path string, v reflect.Value, allowUnderscore bool) (engine.EntityValue, error) {
	v = unwrapInterface(v)
	if v.Kind() != reflect.Map {
		return nil, fmt.Errorf("%s: expected map got %s for entity: %w", path, v.Kind().String(), ErrInvalidEntityFormat)
	}

	byKey := map[string]reflect.Value{}

	// only check for __entity once at the top level, don't allow for
	// __entity.__entity.__entity.id
	for iter := v.MapRange(); iter.Next(); {
		byKey[unwrapInterface(iter.Key()).String()] = iter.Value()
	}

	if allowUnderscore {
		if val, found := byKey["__entity"]; found {
			return specialEntity(path, val, false)
		}
	}

	id, found := byKey["id"]
	if !found {
		return nil, fmt.Errorf("%s: missing id field for entity: %w", path, ErrInvalidEntityFormat)
	}
	kind, found := byKey["type"]
	if !found {
		return nil, fmt.Errorf("%s: missing type field for entity: %w", path, ErrInvalidEntityFormat)
	}
	id = unwrapInterface(id)
	kind = unwrapInterface(kind)

	if id.Kind() != reflect.String {
		return nil, fmt.Errorf("%s: 'id' type not string got %s for entity: %w", path, id.Kind().String(), ErrInvalidEntityFormat)
	}
	if kind.Kind() != reflect.String {
		return nil, fmt.Errorf("%s: 'type' type not string got %s for entity: %w", path, kind.Kind().String(), ErrInvalidEntityFormat)
	}

	return engine.NewEntityValue(kind.String(), id.String()), nil
}

func walkMap(path string, v reflect.Value, shape map[string]*EntityShape) (engine.NamedType, error) {
	children := map[string]engine.NamedType{}
	iter := v.MapRange()

	for iter.Next() {
		k := iter.Key()
		if k.Kind() != reflect.String {
			return nil, fmt.Errorf("invalid map key %v: %w", k.Kind(), ErrInvalidMapKey)
		}

		key := k.String()
		/*
			if key == "__entity" {
				val, err = specialEntity(val)
				if err != nil {
					return nil, err
				}
				return val, nil
			}
		*/

		var sub *EntityShape
		if shape != nil {
			if sval, found := shape[key]; found {
				sub = sval
			}
		}
		if (sub != nil && sub.Type == SHAPE_ENTITY) || key == "__entity" {
			val, err := specialEntity(path, iter.Value(), key != "__entity")
			if err != nil {
				return nil, err
			}
			if sub != nil && val.EntityType() != sub.Name {
				return nil, fmt.Errorf("%s: entity of the wrong type got %s expected %s: %w",
					path, val[len(val)-1], sub.Name, ErrInvalidEntityFormat)
			}
			if key == "__entity" {
				return val, nil
			}
			children[key] = val
			continue
		}
		if (sub != nil && sub.Type == SHAPE_EXTENSION) || key == "__extn" {
			kind := ""
			if sub != nil {
				kind = sub.Name
			}
			val, err := specialExtension(path, kind, iter.Value())
			if err != nil {
				return nil, err
			}
			if key == "__extn" {
				return val, nil
			}
			children[key] = val
			continue
		}

		val, err := walkValue(path+"."+key, iter.Value(), sub)
		if err != nil {
			return nil, err
		}

		children[key] = val
	}

	for key, item := range shape {
		if !item.Required {
			continue
		}
		if _, found := children[key]; found {
			continue
		}
		return nil, fmt.Errorf("%s: required field %s not provided: %w", path, key, ErrInvalidEntityFormat)
	}

	return engine.NewVarValue(children), nil
}

func walkStruct(path string, v reflect.Value, shape map[string]*EntityShape) (engine.NamedType, error) {
	children := map[string]engine.NamedType{}

	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		name := field.Name
		tag := field.Tag.Get("cedar")
		if tag != "" {
			name = tag
		} else {
			// TODO - Technically this is a byte :(
			name = string(unicode.ToLower(rune(name[0]))) + name[1:]
		}

		var sub *EntityShape
		if shape != nil {
			if sval, found := shape[name]; found {
				sub = sval
			}
		}

		val, err := walkValue(path+"."+name, v.Field(i), sub)
		if err != nil {
			return nil, err
		}

		children[name] = val
	}

	return engine.NewVarValue(children), nil
}

func walkValue(path string, v reflect.Value, shape *EntityShape) (engine.NamedType, error) {
	// fmt.Printf("Visiting %v\n", v)
	// Indirect through pointers and interfaces
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Interface:
		// Ignore
	case reflect.Array, reflect.Slice:
		var sub *EntityShape
		if shape != nil {
			if shape.Type != SHAPE_SET {
				return nil, fmt.Errorf("unexpected type at key %s expected record: %w", "", ErrInvalidEntityFormat)
			}
			sub = shape.Element
		}
		v, err := walkSlice(path, v, sub)
		if err != nil {
			return nil, err
		}
		return v, nil
	case reflect.Map:
		var sub map[string]*EntityShape
		if shape == nil {
			// nothing
		} else if shape.Type == SHAPE_ENTITY {
			return specialEntity(path, v, true)
		} else if shape.Type == SHAPE_EXTENSION {
			// TODO
		} else if shape.Type == SHAPE_RECORD {
			sub = shape.Attributes
		} else {
			return nil, fmt.Errorf("unexpected type at key %s expected record: %w", "", ErrInvalidEntityFormat)
		}
		v, err := walkMap(path, v, sub)
		if err != nil {
			return nil, err
		}
		return v, nil
	case reflect.Struct:
		var sub map[string]*EntityShape
		if shape == nil {
			// nothing
		} else if shape.Type == SHAPE_ENTITY {
			// TODO
		} else if shape.Type == SHAPE_EXTENSION {
			// TODO
		} else if shape.Type == SHAPE_RECORD {
			sub = shape.Attributes
		} else {
			return nil, fmt.Errorf("unexpected type at key %s expected record: %w", "", ErrInvalidEntityFormat)
		}
		v, err := walkStruct(path, v, sub)
		if err != nil {
			return nil, err
		}
		return v, nil

	case reflect.Bool:
		if shape != nil && shape.Type != SHAPE_BOOL {
			return nil, fmt.Errorf("unexpected type at key %s expected bool: %w", "", ErrInvalidEntityFormat)
		}
		return engine.BoolValue(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if shape != nil && shape.Type != SHAPE_LONG {
			return nil, fmt.Errorf("unexpected type at key %s expected long: %w", "", ErrInvalidEntityFormat)
		}
		return engine.IntValue(v.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if shape != nil && shape.Type != SHAPE_LONG {
			return nil, fmt.Errorf("unexpected type at key %s expected long: %w", "", ErrInvalidEntityFormat)
		}
		return engine.IntValue(int(v.Uint())), nil
	case reflect.Float32, reflect.Float64:
		if shape != nil && shape.Type != SHAPE_LONG {
			return nil, fmt.Errorf("unexpected type at key %s expected long: %w", "", ErrInvalidEntityFormat)
		}
		return engine.IntValue(int(v.Float())), nil
		// return engine.StrValue(fmt.Sprintf("%f", v.Float())), nil
	case reflect.String:
		if shape != nil && shape.Type != SHAPE_STRING {
			return nil, fmt.Errorf("unexpected type at key %s expected string: %w", "", ErrInvalidEntityFormat)
		}
		return engine.StrValue(v.String()), nil

		// Not supported
		// case reflect.Uintptr:
		// case reflect.Complex64:
		// case reflect.Complex128:
		// case reflect.Chan:
		// case reflect.Func:
		// case reflect.Pointer:
		// case reflect.UnsafePointer:

	}
	return nil, fmt.Errorf("unexpected type %s: %w", v.Kind().String(), ErrUnsupportedType)
}

func (schema *Schema) findActionShape(action, principal, resource engine.EntityValue) *EntityShape {
	var namespace string
	nlen := 0
	if len(action) > 2 {
		nlen = len(action) - 2
		namespace = strings.Join(action[0:nlen], "::")
	}
	nsrules, found := schema.Actions[namespace]
	if !found {
		return nil
	}
	rules, found := nsrules[action[len(action)-1]]
	if !found {
		return nil
	}
	if rules.HasPrincipalTypes {
		if len(principal) < nlen {
			return nil
		}
		if !rules.PrincipalTypes[principal.EntityType()] {
			return nil
		}
	}
	if rules.HasResourceTypes {
		if len(resource) < nlen {
			return nil
		}
		if !rules.ResourceTypes[resource.EntityType()] {
			return nil
		}
	}

	return rules.Context
}

func (schema *Schema) NormalizeContext(input any, principal, action, resource engine.EntityValue) (*engine.VarValue, error) {
	shape := schema.findActionShape(action, principal, resource)

	output, err := walkValue("", reflect.ValueOf(input), shape)
	if err != nil {
		return nil, fmt.Errorf("unable to parse context: %s", err)
	}
	varval, ok := output.(*engine.VarValue)
	if !ok {
		return nil, fmt.Errorf("expected variable type got=%s: %w", output.TypeName(), ErrUnsupportedType)
	}

	return varval, nil
}

func (schema *Schema) NormalizeEntites(input JsonEntities) (EntityStore, error) {
	collection := EntityStore{}

	for _, item := range input {
		uid, err := specialEntity("", reflect.ValueOf(item.Uid), true)
		if err != nil {
			return nil, err
		}

		var parents []engine.EntityValue
		for _, item := range item.Parents {
			ent, err := specialEntity("", reflect.ValueOf(item), true)

			if err != nil {
				return nil, err
			}

			parents = append(parents, ent)
		}

		shape, err := schema.FindDef(uid)
		if err != nil {
			return nil, err
		}

		output, err := walkValue(uid.String(), reflect.ValueOf(item.Attrs), shape)
		if err != nil {
			return nil, err
		}
		varval, ok := output.(*engine.VarValue)
		if !ok {
			return nil, fmt.Errorf("expected variable type got=%s: %w", output.TypeName(), ErrUnsupportedType)
		}

		collection[uid.String()] = EntityStoreItem{
			entity:  uid,
			values:  varval,
			parents: parents,
		}
	}

	return collection, nil
}
