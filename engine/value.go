package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Basic type interface for all values
type NamedType interface {
	TypeName() string
	String() string
	AsJson() any
	OpEqual(input NamedType) (BoolValue, error)
}

// Types that can do math operations (presently only ints)
type MathType interface {
	OpUnaryMinus() (NamedType, error)
	OpAdd(input NamedType) (NamedType, error)
	OpSub(input NamedType) (NamedType, error)
	OpMul(input NamedType) (NamedType, error)
	OpQuo(input NamedType) (NamedType, error)
	OpRem(input NamedType) (NamedType, error)
}

// Types that support `<` and `<=` operations
// Note: tyis also covers `>` and `>=` via swapping left and right
type ComparisonType interface {
	OpLss(input NamedType) (BoolValue, error)
	OpLeq(input NamedType) (BoolValue, error)
}

// For `||“, `&&“ and `!“
type LogicType interface {
	OpNot() (BoolValue, error)
	OpLor(input NamedType) (BoolValue, error)
	OpLand(input NamedType) (BoolValue, error)
}

type IsType interface {
	OpIs(input NamedType) (BoolValue, error)
}

// For X in { ... }
type InType interface {
	OpIn(input NamedType, store Store) (BoolValue, error)
}

type LikeType interface {
	OpLike(input NamedType) (BoolValue, error)
}

type CallableType interface {
	// Fetch property
	OpCall(input NamedType, args []NamedType) (NamedType, error)
}

type VariableType interface {
	// Call function
	OpLookup(input NamedType, store Store) (EvalValue, error)
	// Check for property
	OpHas(input NamedType, store Store) (BoolValue, error)
}

var ErrTypeMismatch = errors.New("type mismatch")

type BoolValue bool

var _ NamedType = (*BoolValue)(nil)
var _ LogicType = (*BoolValue)(nil)

func (v1 BoolValue) TypeName() string {
	return "boolean"
}

func (v1 BoolValue) OpEqual(input NamedType) (BoolValue, error) {
	v2, ok := input.(BoolValue)
	if !ok {
		return false, fmt.Errorf("expected boolean got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return v1 == v2, nil
}

func (v1 BoolValue) String() string {
	if bool(v1) {
		return "true"
	}
	return "false"
}

func (v1 BoolValue) AsJson() any {
	return v1
}

func (v1 BoolValue) OpLand(input NamedType) (BoolValue, error) {
	v2, ok := input.(BoolValue)
	if !ok {
		return false, fmt.Errorf("expected boolean got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return BoolValue(v1 && v2), nil
}

func (v1 BoolValue) OpLor(input NamedType) (BoolValue, error) {
	v2, ok := input.(BoolValue)
	if !ok {
		return false, fmt.Errorf("expected boolean got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return BoolValue(v1 || v2), nil
}

func (v1 BoolValue) OpNot() (BoolValue, error) {
	return BoolValue(!v1), nil
}

// ---------

type IntValue int

var _ NamedType = (*IntValue)(nil)
var _ MathType = (*IntValue)(nil)
var _ ComparisonType = (*IntValue)(nil)

func (v1 IntValue) TypeName() string {
	return "long"
}

func (v1 IntValue) OpEqual(input NamedType) (BoolValue, error) {
	v2, ok := input.(IntValue)
	if !ok {
		return false, fmt.Errorf("expected long got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return BoolValue(v1 == v2), nil
}

func (v1 IntValue) String() string {
	val := int64(v1)

	return strconv.FormatInt(val, 10)
}

func (v1 IntValue) AsJson() any {
	return v1
}

// Integer Comparison

func (v1 IntValue) OpLss(input NamedType) (BoolValue, error) {
	v2, ok := input.(IntValue)
	if !ok {
		return (false), fmt.Errorf("expected long got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return (v1 < v2), nil
}

func (v1 IntValue) OpLeq(input NamedType) (BoolValue, error) {
	v2, ok := input.(IntValue)
	if !ok {
		return (false), fmt.Errorf("expected long got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return (v1 <= v2), nil
}

// Integer Math
func (v1 IntValue) OpUnaryMinus() (NamedType, error) {
	return IntValue(-int(v1)), nil
}

func (v1 IntValue) OpAdd(input NamedType) (NamedType, error) {
	v2, ok := input.(IntValue)
	if !ok {
		return nil, fmt.Errorf("expected long got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return IntValue(v1 + v2), nil
}

func (v1 IntValue) OpSub(input NamedType) (NamedType, error) {
	v2, ok := input.(IntValue)
	if !ok {
		return nil, fmt.Errorf("expected int got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return IntValue(v1 - v2), nil
}

func (v1 IntValue) OpMul(input NamedType) (NamedType, error) {
	v2, ok := input.(IntValue)
	if !ok {
		return nil, fmt.Errorf("expected int got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return IntValue(v1 * v2), nil
}

func (v1 IntValue) OpQuo(input NamedType) (NamedType, error) {
	v2, ok := input.(IntValue)
	if !ok {
		return nil, fmt.Errorf("expected boolean got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return IntValue(v1 / v2), nil
}

func (v1 IntValue) OpRem(input NamedType) (NamedType, error) {
	v2, ok := input.(IntValue)
	if !ok {
		return nil, fmt.Errorf("expected boolean got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return IntValue(v1 % v2), nil
}

// ---------

type StrValue string

var _ NamedType = (*StrValue)(nil)
var _ LikeType = (*StrValue)(nil)

func (v1 StrValue) TypeName() string {
	return "string"
}

func (v1 StrValue) OpEqual(input NamedType) (BoolValue, error) {
	v2, ok := input.(StrValue)
	if !ok {
		return false, fmt.Errorf("expected string got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return BoolValue(v1 == v2), nil
}

func (v1 StrValue) String() string {
	return string(v1)
}

func (v1 StrValue) AsJson() any {
	return v1
}

func (v1 StrValue) OpLike(input NamedType) (BoolValue, error) {
	v2, ok := input.(StrValue)
	if !ok {
		return false, fmt.Errorf("expected string got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return BoolValue(Glob(string(v2), string(v1))), nil
}

// ---------

type EntityValue []string

const ENTITY_PATH_SEP = "::"

var _ NamedType = (*EntityValue)(nil)
var _ InType = (*EntityValue)(nil)
var _ IsType = (*EntityValue)(nil)
var _ VariableType = (*EntityValue)(nil)

func NewEntityValue(kind string, id string) EntityValue {
	parts := strings.Split(kind, ENTITY_PATH_SEP)

	return append(parts, id)
}

func NewEntityFromString(value string) EntityValue {
	parts := strings.Split(value, ENTITY_PATH_SEP)

	id := parts[len(parts)-1]
	if len(id) > 1 && id[0] == '"' && id[len(id)-1] == '"' {
		id = id[1 : len(id)-1]
	}

	return append(parts[0:len(parts)-1], id)
}

// EntityType returns the namespaced type name for this entity
func (v1 EntityValue) EntityType() string {
	if v1 == nil {
		return ""
	}
	return strings.Join(v1[0:len(v1)-1], ENTITY_PATH_SEP)
}

// EntityType returns the Id value for this entity
func (v1 EntityValue) EntityId() string {
	if v1 == nil {
		return ""
	}
	return v1[len(v1)-1]
}

func (v1 EntityValue) TypeName() string {
	return "entity"
}

func (v1 EntityValue) OpEqual(input NamedType) (BoolValue, error) {
	v2, ok := input.(EntityValue)
	if !ok {
		return false, fmt.Errorf("expected entity got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	if len(v1) != len(v2) {
		return false, nil
	}
	for idx, val := range v1 {
		if v2[idx] != val {
			return BoolValue(false), nil
		}
	}

	return BoolValue(true), nil
}

func (v1 EntityValue) String() string {
	return v1.EntityType() + ENTITY_PATH_SEP + "\"" + v1.EntityId() + "\""
}

func (v1 EntityValue) AsJson() any {
	l := len(v1) - 1
	return map[string]any{
		"__entity": map[string]string{
			"type": v1.EntityType(),
			"id":   v1[l],
		},
	}
}

func (v1 *EntityValue) UnmarshalJSON(data []byte) error {
	type TypeId struct {
		Id   *string `json:"id"`
		Type *string `json:"type"`
	}
	type Record struct {
		TypeId
		Entity *TypeId `json:"__entity"`
	}

	var record *Record

	if err := json.Unmarshal(data, &record); err != nil {
		return err
	}

	if record == nil {
		return nil
	}

	id := record.Id
	kind := record.Type
	if record.Entity != nil {
		id = record.Id
		kind = record.Type
	}
	if id == nil {
		return fmt.Errorf("missing 'id' property: %w", ErrInvalidEntityFormat)
	}
	if kind == nil {
		return fmt.Errorf("missing 'type' property: %w", ErrInvalidEntityFormat)
	}

	*v1 = NewEntityValue(*kind, *id)

	return nil
}

func (v1 EntityValue) OpIn(input NamedType, store Store) (BoolValue, error) {
	parents, err := store.GetParents(v1)
	if err != nil {
		return false, err
	}

	entities := map[string]bool{}
	if rval, ok := input.(EntityValue); ok {
		entities[rval.String()] = true
	} else if rval, ok := input.(SetValue); ok {
		for _, item := range rval {
			val, ok := item.(EntityValue)
			if !ok {
				return false, fmt.Errorf("expected entity got %s: %w", item.TypeName(), ErrTypeMismatch)
			}
			entities[val.String()] = true
		}
	} else {
		return false, fmt.Errorf("expected entity or set got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	for _, item := range parents {
		if entities[item.String()] {
			return true, nil
		}
	}

	return false, nil
}

func (v1 EntityValue) OpIs(input NamedType) (BoolValue, error) {
	rval, ok := input.(EntityValue)
	if !ok {
		return false, fmt.Errorf("expected identifier got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	// For the moment the last portion of the "Is" value is a empty string, so drop it
	l := len(rval) - 1
	if rval[l] == "" {
		rval = rval[0:l]
	}
	for idx, part := range rval {
		if idx >= len(v1) {
			return false, nil
		}
		if part != v1[idx] {
			return false, nil
		}
	}

	return true, nil
}

func valueAsString(input NamedType) (string, error) {
	if val, ok := input.(StrValue); ok {
		return string(val), nil
	}
	if val, ok := input.(IdentifierValue); ok {
		return string(val), nil
	}
	return "", fmt.Errorf("expected identifier or string got %s: %w", input.TypeName(), ErrEvalError)
}

func (v1 EntityValue) OpHas(input NamedType, store Store) (BoolValue, error) {
	str, err := valueAsString(input)
	if err != nil {
		return false, err
	}

	_, err = store.Get(v1, str)
	if errors.Is(err, ErrValueNotFound) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("value not found: %w", err)
	}

	return true, nil
}

func (v1 EntityValue) OpLookup(input NamedType, store Store) (EvalValue, error) {
	str, err := valueAsString(input)
	if err != nil {
		return nil, err
	}
	val, err := store.Get(v1, str)
	if errors.Is(err, ErrValueNotFound) {
		return nil, err
	} else if err != nil {
		return nil, fmt.Errorf("value not found: %w", err)
	}

	return val, nil
}

// ---------

type SetValue []NamedType

var _ NamedType = (*SetValue)(nil)

func (v1 SetValue) TypeName() string {
	return "set"
}

func (v1 SetValue) OpEqual(input NamedType) (BoolValue, error) {
	v2, ok := input.(SetValue)
	if !ok {
		return false, fmt.Errorf("expected set got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	if len(v1) != len(v2) {
		return false, nil
	}

	// Just for completless
	for _, val := range v1 {
		found := false
		for _, bval := range v2 {
			if equal, err := val.OpEqual(bval); equal {
				found = true
				break
			} else if err != nil {
				return false, err
			}
		}
		if !found {
			return false, nil
		}
	}
	return BoolValue(true), nil
}

func (v1 SetValue) String() string {
	values := []string{}
	for _, item := range v1 {
		values = append(values, item.String())
	}
	return "{" + strings.Join(values, ",") + "}"
}

func (v1 SetValue) AsJson() any {
	result := []any{}
	for _, item := range v1 {
		result = append(result, item.AsJson())
	}
	return result
}

// ---------

type IdentifierValue string

var _ NamedType = (*IdentifierValue)(nil)

func (v1 IdentifierValue) TypeName() string {
	return "set"
}

func (v1 IdentifierValue) String() string {
	return string(v1)
}

func (v1 IdentifierValue) AsJson() any {
	return string(v1)
}

func (v1 IdentifierValue) OpEqual(input NamedType) (BoolValue, error) {
	v2, ok := input.(IdentifierValue)
	if !ok {
		return false, fmt.Errorf("expected set got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return string(v1) == string(v2), nil
}

// ---------

type VarValue struct {
	// self     NamedType
	children map[string]NamedType
}

var _ NamedType = (*VarValue)(nil)
var _ VariableType = (*VarValue)(nil)

func NewVarValue(data map[string]NamedType) *VarValue {
	return &VarValue{
		children: data,
	}
}

func (v1 *VarValue) Get(id string) (NamedType, bool) {
	val, ok := v1.children[id]

	return val, ok
}

func (v1 *VarValue) TypeName() string {
	return "variables"
}

func (v1 *VarValue) String() string {
	// TODO
	return "<<TBD>>"
}

func (v1 *VarValue) AsJson() any {
	result := map[string]any{}

	for k, v := range v1.children {
		result[k] = v.AsJson()
	}

	return result
}

// func (v1 *VarValue) UnmarshalJSON(data []byte) error {
// 	raw := map[string]any{}

// 	err := json.Unmarshal(data, &raw)
// 	if err != nil {
// 		return err
// 	}

// 	values, err := EntityStoreDataFromMap(raw)
// 	if err != nil {
// 		return err
// 	}
// 	val, ok := values.(VarValue)
// 	if !ok {
// 		return fmt.Errorf("return type is not correct: %w", ErrUnsupportedType)
// 	}

// 	*v1 = val

// 	return nil
// }

func (v1 *VarValue) OpEqual(input NamedType) (BoolValue, error) {
	return false, nil
}

func (v1 *VarValue) OpLookup(input NamedType, store Store) (EvalValue, error) {
	if val, ok := input.(StrValue); ok {
		child, found := v1.children[string(val)]
		if !found {
			return nil, fmt.Errorf("lookup key not found \"%s\": %w", val, ErrValueNotFound)
		}
		return child, nil
	}
	if val, ok := input.(IdentifierValue); ok {
		child, found := v1.children[string(val)]
		if !found {
			return nil, fmt.Errorf("lookup key not found \"%s\": %w", val, ErrValueNotFound)
		}
		return child, nil
	}

	return nil, fmt.Errorf("invalid type: %s: %w", input.TypeName(), ErrUnsupportedType)
}

func (v1 *VarValue) OpHas(input NamedType, store Store) (BoolValue, error) {
	if val, ok := input.(StrValue); ok {
		_, found := v1.children[string(val)]
		return BoolValue(found), nil
	}
	if val, ok := input.(IdentifierValue); ok {
		_, found := v1.children[string(val)]
		return BoolValue(found), nil
	}

	return BoolValue(false), fmt.Errorf("invalid type: %s: %w", input.TypeName(), ErrUnsupportedType)
}
