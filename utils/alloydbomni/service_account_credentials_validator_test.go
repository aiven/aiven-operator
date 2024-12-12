package alloydbomniUtils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateServiceAccountCredentials(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "valid",
			input: `{
				"private_key_id": "0",
				"private_key": "1",
				"client_email": "2",
				"client_id": "3"
			}`,
			expected: "",
		},
		{
			name:     "invalid, empty",
			input:    `{}`,
			expected: "(root): private_key_id is required\n(root): private_key is required\n(root): client_email is required\n(root): client_id is required\n",
		},
		{
			name: "missing private_key_id",
			input: `{
				"private_key": "1",
				"client_email": "2",
				"client_id": "3"
			}`,
			expected: "(root): private_key_id is required\n",
		},
		{
			name: "invalid type client_id",
			input: `{
				"private_key_id": "0",
				"private_key": "1",
				"client_email": "2",
				"client_id": 3
			}`,
			expected: "client_id: Invalid type. Expected: string, given: integer\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateServiceAccountCredentials(tc.input)
			if tc.expected == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expected)
			}
		})
	}
}
