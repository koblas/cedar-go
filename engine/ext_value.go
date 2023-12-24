package ast

import (
	"fmt"
	"net"
	"strconv"
)

// IP Value
type IpValue struct {
	addr net.IP
	cidr *net.IPNet
}

var _ NamedType = (*IpValue)(nil)

func NewIpValue(arg string) (*IpValue, error) {
	if ip := net.ParseIP(arg); ip != nil {
		return &IpValue{addr: ip}, nil
	}

	ip, cidr, err := net.ParseCIDR(string(arg))
	if err != nil {
		return nil, fmt.Errorf("ip address is not valid: %w", ErrTypeMismatch)
	}
	if ip == nil {
		return nil, fmt.Errorf("ip address is not valid: %w", ErrTypeMismatch)
	}

	return &IpValue{addr: ip, cidr: cidr}, nil
}

func (v1 *IpValue) TypeName() string {
	return "ipaddr"
}

func (v1 *IpValue) String() string {
	if v1.cidr != nil {
		return v1.cidr.String()
	}
	return v1.addr.String()
}

func (v1 *IpValue) OpEqual(input NamedType) (BoolValue, error) {
	v2, ok := input.(*IpValue)
	if !ok {
		return false, fmt.Errorf("expected ip got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return BoolValue(v1.cidr == v2.cidr && v1.String() == v2.String()), nil
}

func (v1 *IpValue) AsJson() any {
	return map[string]any{
		"__extn": map[string]string{
			"fn":  "ip",
			"arg": v1.String(),
		},
	}
}

// Internal helper for function to get the type back
func asNetIp(input NamedType) (*IpValue, error) {
	if input == nil {
		return nil, fmt.Errorf("expected ip got nil: %w", ErrTypeMismatch)
	}
	val, ok := input.(*IpValue)
	if !ok {
		return nil, fmt.Errorf("expected ip got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return val, nil
}

// Decimal Value
type DecimalValue float64

var _ NamedType = (*DecimalValue)(nil)

func NewDecimalValue(arg string) (DecimalValue, error) {
	fval, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse decimal: %s: %w", err, ErrTypeMismatch)
	}

	return DecimalValue(fval), nil
}

func (v1 DecimalValue) TypeName() string {
	return "decimal"
}

func (v1 DecimalValue) String() string {
	return fmt.Sprintf("%f", v1)
}

func (v1 DecimalValue) OpEqual(input NamedType) (BoolValue, error) {
	v2, ok := input.(DecimalValue)
	if !ok {
		return false, fmt.Errorf("expected decimal got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return v1 == v2, nil
}

func (v1 DecimalValue) AsJson() any {
	return map[string]any{
		"__extn": map[string]string{
			"fn":  "decimal",
			"arg": v1.String(),
		},
	}
}

// Internal helper for function to get the type back
func asFloat(input NamedType) (float64, error) {
	if input == nil {
		return 0, fmt.Errorf("expected ip got nil: %w", ErrTypeMismatch)
	}
	val, ok := input.(DecimalValue)
	if !ok {
		return 0, fmt.Errorf("expected decimal got %s: %w", input.TypeName(), ErrTypeMismatch)
	}

	return float64(val), nil
}
