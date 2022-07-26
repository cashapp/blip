// Copyright 2022 Block, Inc.

package dbconn_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/dbconn"
)

func TestParseMyCnf(t *testing.T) {
	gotCfg, gotTLS, err := dbconn.ParseMyCnf("../test/mycnf/full-dsn")
	if err != nil {
		t.Error(err)
	}
	expectCfg := blip.ConfigMySQL{
		Username: "U",
		Password: "P",
		Hostname: "H:33560",
	}
	expectTLS := blip.ConfigTLS{
		CA:        "CA",
		Cert:      "CERT",
		Key:       "KEY",
		MySQLMode: "PREFERRED",
	}
	assert.Equal(t, gotCfg, expectCfg)
	assert.Equal(t, gotTLS, expectTLS)
}

func TestParseMyCnfSocketTLS(t *testing.T) {
	gotCfg, gotTLS, err := dbconn.ParseMyCnf("../test/mycnf/socket-tls")
	if err != nil {
		t.Error(err)
	}
	expectCfg := blip.ConfigMySQL{
		Username: "U",
		Password: "P",
		Socket:   "socketFile",
	}
	expectTLS := blip.ConfigTLS{
		MySQLMode: "PREFERRED",
	}
	assert.Equal(t, gotCfg, expectCfg)
	assert.Equal(t, gotTLS, expectTLS)
}
