package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertDiskSpace(t *testing.T) {
	cases := map[string]int{
		"10GiB": 10240,
		"10gib": 10240,
		"10G":   10240,
		"10g":   10240,
		"1g":    1024,
		"2g":    2048,
		"":      0,
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			result := ConvertDiskSpace(k)
			assert.Equal(t, v, result)
		})
	}
}
