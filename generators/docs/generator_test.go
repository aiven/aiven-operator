package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrettyDigitFloat(t *testing.T) {
	cases := []struct {
		src      float64
		expected string
	}{
		{
			src:      0.90000,
			expected: "0.9",
		},
		{
			src:      0.99900,
			expected: "0.999",
		},
		{
			src:      0.99999,
			expected: "0.99999",
		},
		{
			src:      0.90900,
			expected: "0.909",
		},
		{
			src:      60.0000,
			expected: "60",
		},
		{
			src:      0.0000,
			expected: "0",
		},
	}

	for i, o := range cases {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			assert.Equal(t, o.expected, prettyDigit("number", o.src))
		})
	}
}
