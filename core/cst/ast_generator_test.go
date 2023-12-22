package cst

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var stringCases = []struct {
	input  string
	expect string
}{
	{`"abc"`, "abc"},
	{`"\n"`, "\n"},
	{`"\0"`, "\000"},
	{`"\u{6}"`, "\u0006"},
	{`"\u{6"`, "\\u{6"},
	{`"\u{2}1\u{1b}\"\u{2}\u{2}\u{2}\u{2}"`, "\u00021\u001b\"\u0002\u0002\u0002\u0002"},
}

func TestUnquote(t *testing.T) {
	for _, item := range stringCases {
		output := unquote(item.input)
		assert.Equal(t, item.expect, output)
	}
}
