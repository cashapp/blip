package dbconn

import (
	"github.com/go-ini/ini"

	"github.com/cashapp/blip"
)

func ParseMyCnf(file string) (blip.ConfigMySQL, error) {
	opts := ini.LoadOptions{AllowBooleanKeys: true}
	mycnf, err := ini.LoadSources(opts, file)
	if err != nil {
		return blip.ConfigMySQL{}, err
	}

	cfg := blip.ConfigMySQL{
		Username: mycnf.Section("client").Key("user").String(),
		Password: mycnf.Section("client").Key("password").String(),
		Hostname: mycnf.Section("client").Key("host").String(),
		Socket:   mycnf.Section("client").Key("socket").String(),
	}

	port := mycnf.Section("client").Key("port").String()
	if port != "" {
		cfg.Hostname += ":" + port
	}

	ca := mycnf.Section("client").Key("ssl-ca").String()
	cert := mycnf.Section("client").Key("ssl-cert").String()
	key := mycnf.Section("client").Key("ssl-key").String()
	if ca != "" || cert != "" || key != "" {
		cfg.TLSCA = ca
		cfg.TLSCert = cert
		cfg.TLSKey = key
	}

	return cfg, nil
}
