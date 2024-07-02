// Copyright 2024 Block, Inc.

package blip_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cashapp/blip"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

// --------------------------------------------------------------------------

func TestDefaultConfig(t *testing.T) {
	// The default config should be valid. Would be embarrassing if not.
	got := blip.DefaultConfig()
	if err := got.Validate(); err != nil {
		t.Errorf("default config is not valid, expected it to be valid: %s", err)
	}
}

// TestEnvInterpolation verifies that env vars with special characters, most notably $, are properly interpolated
func TestEnvInterpolation(t *testing.T) {
	envKey := "blip_test_TestEnvInterpolation"
	envVal := "a$1b!@#$%^&*()-+={};\""
	defer os.Unsetenv(envKey)

	err := os.Setenv(envKey, envVal)
	require.Nil(t, err)

	cfg := blip.Config{MySQL: blip.ConfigMySQL{Password: fmt.Sprintf("${%s}", envKey)}}
	cfg.InterpolateEnvVars()
	assert.Equal(t, envVal, cfg.MySQL.Password)
}

// TestEnvInterpolationEmpty verifies that a config like ${FOO:-bar} without FOO set, evaluates to "bar"
func TestEnvInterpolationEmpty(t *testing.T) {
	envKey := "blip_test_TestEnvInterpolation"
	_ = os.Unsetenv(envKey)

	cfg := blip.Config{MySQL: blip.ConfigMySQL{Password: fmt.Sprintf("${%s:-bar}", envKey)}}
	cfg.InterpolateEnvVars()
	assert.Equal(t, "bar", cfg.MySQL.Password)
}

func TestApplyDefaultConfig(t *testing.T) {
	// Defaults apply when the value isn't set
	df := blip.DefaultConfig()
	my := blip.Config{}
	my.ApplyDefaults(df)
	if my.API.Bind != blip.DEFAULT_API_BIND {
		t.Errorf("api.bind=%s, expected %s", my.API.Bind, blip.DEFAULT_API_BIND)
	}

	// But when a value is set, it overrides the default
	my = blip.Config{
		API: blip.ConfigAPI{
			Bind: ":1234",
		},
	}
	my.ApplyDefaults(df)
	if my.API.Bind != ":1234" {
		t.Errorf("api.bind=%s, expected :1234", my.API.Bind)
	}
}
