package controllers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidatePollInterval(t *testing.T) {
	cases := []struct {
		name    string
		give    time.Duration
		wantErr bool
	}{
		{name: "zero means use the default", give: 0},
		{name: "at the floor", give: 10 * time.Minute},
		{name: "at the ceiling", give: 60 * time.Minute},
		{name: "within the range", give: 30 * time.Minute},
		{name: "below the floor", give: time.Minute, wantErr: true},
		{name: "just below the floor", give: 10*time.Minute - time.Second, wantErr: true},
		// The floor equals the default, so anything faster than the default is rejected.
		{name: "faster than the default", give: 5 * time.Minute, wantErr: true},
		{name: "just above the ceiling", give: 60*time.Minute + time.Second, wantErr: true},
		{name: "above the ceiling", give: 2 * time.Hour, wantErr: true},
		{name: "negative", give: -time.Minute, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePollInterval(tc.give)
			if !tc.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.give.String())
			require.Contains(t, err.Error(), "10m0s-1h0m0s")
		})
	}
}

func TestSetupConfigNormalizePollInterval(t *testing.T) {
	cases := []struct {
		name string
		give time.Duration
		want time.Duration
	}{
		{name: "unset falls back to the default", give: 0, want: DefaultPollInterval},
		{name: "negative falls back to the default", give: -time.Minute, want: DefaultPollInterval},
		{name: "a set value is left alone", give: 30 * time.Minute, want: 30 * time.Minute},
		{name: "an out-of-range value is not clamped", give: time.Minute, want: time.Minute},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := SetupConfig{PollInterval: tc.give}
			cfg.normalize()
			require.Equal(t, tc.want, cfg.PollInterval)
		})
	}
}
