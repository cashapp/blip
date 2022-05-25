// Copyright 2022 Block, Inc.

package dbconn_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/dbconn"
)

func TestParseMyCnf(t *testing.T) {
	got, err := dbconn.ParseMyCnf("../test/mycnf/full-dsn")
	if err != nil {
		t.Error(err)
	}
	expect := blip.ConfigMySQL{
		Username: "U",
		Password: "P",
		Hostname: "H:33560",
		TLSCA:    "CA",
		TLSCert:  "CERT",
		TLSKey:   "KEY",
	}

	assert.Equal(t, got, expect)
}
