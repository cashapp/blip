// Copyright 2022 Block, Inc.

package blip_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	//"github.com/stretchr/testify/assert"

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

	// The default config should have minimal config with default values.
	//expect := blip.Config{}
	//assert.Equal(t, got, expect)
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
