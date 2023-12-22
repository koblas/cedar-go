package ast

import (
	"fmt"
)

type Function func(left EvalValue, args []EvalValue) (EvalValue, error)

var functionTable = map[string]Function{
	//
	// IP Functions
	//
	"ip": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("func[ip]: expected one argument got %d: %w", len(args), ErrTypeMismatch)
		}
		input := args[0]

		val, ok := input.(StrValue)
		if !ok {
			return nil, fmt.Errorf("func[ip]: expected string got %s: %w", input.TypeName(), ErrTypeMismatch)
		}

		return NewIpValue(string(val))
	},
	"isIpv4": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("expected no arguments got %d: %w", len(args), ErrTypeMismatch)
		}

		val, err := asNetIp(left)
		if err != nil {
			return nil, err
		}

		return BoolValue(val.addr.To4() != nil), nil
	},
	"isIpv6": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("expected no arguments got %d: %w", len(args), ErrTypeMismatch)
		}

		// val, ok := left.(IpValue)
		// if !ok {
		// 	return nil, fmt.Errorf("expected ip got %s: %w", left.TypeName(), ErrTypeMismatch)
		// }

		// If it parsed, it's v6
		return BoolValue(true), nil
	},
	"isLoopback": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("expected no arguments got %d: %w", len(args), ErrTypeMismatch)
		}

		val, err := asNetIp(left)
		if err != nil {
			return nil, err
		}

		return BoolValue(val.addr.IsLoopback()), nil
	},
	"isMulticast": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("expected no arguments got %d: %w", len(args), ErrTypeMismatch)
		}

		val, err := asNetIp(left)
		if err != nil {
			return nil, err
		}

		return BoolValue(val.addr.IsMulticast()), nil
	},
	"isInRange": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("func[isInRange]: expected one argument got %d: %w", len(args), ErrTypeMismatch)
		}

		arg, ok := args[0].(*IpValue)
		if !ok {
			return nil, fmt.Errorf("func[isInRange]: expected string got %s: %w", arg.TypeName(), ErrTypeMismatch)
		}

		if arg.cidr == nil {
			return BoolValue(false), nil
		}

		val, err := asNetIp(left)
		if err != nil {
			return nil, err
		}

		if val.cidr != nil {
			return nil, fmt.Errorf("func[isInRange]: cannot compare cidr address on left: %w", ErrTypeMismatch)
		}

		return BoolValue(arg.cidr.Contains(val.addr)), nil
	},
	//
	// Decmial Functions
	//
	"decimal": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expected one argument got %d: %w", len(args), ErrTypeMismatch)
		}
		input := args[0]

		val, ok := input.(StrValue)
		if !ok {
			return nil, fmt.Errorf("expected string got %s: %w", input.TypeName(), ErrTypeMismatch)
		}

		return NewDecimalValue(string(val))
	},
	"lessThan": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expected one argument got %d: %w", len(args), ErrTypeMismatch)
		}

		arg, ok := args[0].(DecimalValue)
		if !ok {
			return nil, fmt.Errorf("expected decimal got %s: %w", arg.TypeName(), ErrTypeMismatch)
		}

		lval, err := asFloat(left)
		if err != nil {
			return nil, err
		}
		rval, err := asFloat(arg)
		if err != nil {
			return nil, err
		}

		return BoolValue(lval < rval), nil
	},
	"lessThanOrEqual": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expected one argument got %d: %w", len(args), ErrTypeMismatch)
		}

		arg, ok := args[0].(DecimalValue)
		if !ok {
			return nil, fmt.Errorf("expected decimal got %s: %w", arg.TypeName(), ErrTypeMismatch)
		}

		lval, err := asFloat(left)
		if err != nil {
			return nil, err
		}
		rval, err := asFloat(arg)
		if err != nil {
			return nil, err
		}

		return BoolValue(lval <= rval), nil
	},
	"greaterThan": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expected one argument got %d: %w", len(args), ErrTypeMismatch)
		}

		arg, ok := args[0].(DecimalValue)
		if !ok {
			return nil, fmt.Errorf("expected decimal got %s: %w", arg.TypeName(), ErrTypeMismatch)
		}

		lval, err := asFloat(left)
		if err != nil {
			return nil, err
		}
		rval, err := asFloat(arg)
		if err != nil {
			return nil, err
		}

		return BoolValue(lval > rval), nil
	},
	"greaterThanOrEqual": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expected one argument got %d: %w", len(args), ErrTypeMismatch)
		}

		arg, ok := args[0].(DecimalValue)
		if !ok {
			return nil, fmt.Errorf("expected decimal got %s: %w", arg.TypeName(), ErrTypeMismatch)
		}

		lval, err := asFloat(left)
		if err != nil {
			return nil, err
		}
		rval, err := asFloat(arg)
		if err != nil {
			return nil, err
		}

		return BoolValue(lval >= rval), nil
	},

	// Set operators
	"contains": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expected one argument got %d: %w", len(args), ErrTypeMismatch)
		}

		lval, ok := left.(SetValue)
		if !ok {
			return nil, fmt.Errorf("expected set got %s: %w", left.TypeName(), ErrTypeMismatch)
		}

		for _, item := range lval {
			cmp, _ := item.OpEqual(args[0])
			if cmp {
				return BoolValue(true), nil
			}
		}

		return BoolValue(false), nil
	},

	"containsAll": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expected one argument got %d: %w", len(args), ErrTypeMismatch)
		}

		lval, ok := left.(SetValue)
		if !ok {
			return nil, fmt.Errorf("expected set got %s: %w", left.TypeName(), ErrTypeMismatch)
		}
		rval, ok := args[0].(SetValue)
		if !ok {
			return nil, fmt.Errorf("expected set argument got %s: %w", left.TypeName(), ErrTypeMismatch)
		}

		looking := map[NamedType]bool{}
		for _, item := range rval {
			looking[item] = true
		}

		for _, item := range lval {
			for look := range looking {
				// Ignore type errors
				if found, _ := item.OpEqual(look); found {
					delete(looking, item)
					break
				}
			}

			if len(looking) == 0 {
				return BoolValue(true), nil
			}
		}

		return BoolValue(false), nil
	},

	"containsAny": func(left EvalValue, args []EvalValue) (EvalValue, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("expected one argument got %d: %w", len(args), ErrTypeMismatch)
		}

		lval, ok := left.(SetValue)
		if !ok {
			return nil, fmt.Errorf("expected set got %s: %w", left.TypeName(), ErrTypeMismatch)
		}
		rval, ok := args[0].(SetValue)
		if !ok {
			return nil, fmt.Errorf("expected set argument got %s: %w", left.TypeName(), ErrTypeMismatch)
		}

		looking := map[NamedType]bool{}
		for _, item := range rval {
			looking[item] = true
		}

		for _, item := range lval {
			for _, look := range rval {
				if found, _ := item.OpEqual(look); found {
					return BoolValue(true), nil
				}
			}
		}

		return BoolValue(false), nil
	},
}
