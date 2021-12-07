package dbconn_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/square/blip"
	"github.com/square/blip/dbconn"
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
