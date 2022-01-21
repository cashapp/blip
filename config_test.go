// Copyright 2022 Block, Inc.

package blip_test

import (
	"os"
	"testing"

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
	got := blip.DefaultConfig(false) // strict=false
	if err := got.Validate(); err != nil {
		t.Errorf("default config is not valid, expected it to be valid: %s", err)
	}

	// The default config should have minimal config with default values.
	//expect := blip.Config{}
	//assert.Equal(t, got, expect)
}
