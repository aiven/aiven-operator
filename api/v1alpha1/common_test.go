package v1alpha1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestBaseServiceFieldsExplicitEmpty: omitted and explicitly empty values must survive a typed
// encode as different things, because omitted is unmanaged and empty clears at Aiven.
func TestBaseServiceFieldsExplicitEmpty(t *testing.T) {
	t.Run("json round trip keeps empty values and drops omitted ones", func(t *testing.T) {
		in := `{"plan":"startup-4","projectVpcId":"","tags":{},"technicalEmails":[]}`
		var spec BaseServiceFields
		require.NoError(t, json.Unmarshal([]byte(in), &spec))
		assert.Equal(t, new(""), spec.ProjectVPCID)
		assert.NotNil(t, spec.Tags)
		assert.NotNil(t, spec.TechnicalEmails)

		out, err := json.Marshal(spec)
		require.NoError(t, err)
		assert.JSONEq(t, `{"plan":"startup-4","project":"","projectVpcId":"","tags":{},"technicalEmails":[]}`, string(out))

		out, err = json.Marshal(BaseServiceFields{Plan: "startup-4"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"plan":"startup-4","project":""}`, string(out))
	})

	t.Run("validate rejects projectVpcId with projectVPCRef, including an empty id", func(t *testing.T) {
		ref := &ResourceReference{Name: "vpc"}
		require.NoError(t, (&BaseServiceFields{ProjectVPCRef: ref}).Validate())
		require.NoError(t, (&BaseServiceFields{ProjectVPCID: new("vpc-1")}).Validate())
		require.Error(t, (&BaseServiceFields{ProjectVPCID: new("vpc-1"), ProjectVPCRef: ref}).Validate())
		require.Error(t, (&BaseServiceFields{ProjectVPCID: new(""), ProjectVPCRef: ref}).Validate())
	})
}
