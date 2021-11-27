package dbconn

import (
	"github.com/go-ini/ini"

	"github.com/square/blip"
)

func ParseMyCnf(file string) (blip.ConfigMonitor, error) {
	opts := ini.LoadOptions{AllowBooleanKeys: true}
	mycnf, err := ini.LoadSources(opts, file)
	if err != nil {
		return blip.ConfigMonitor{}, err
	}

	cfg := blip.ConfigMonitor{
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
		cfg.TLS = blip.ConfigTLS{
			CA:   ca,
			Cert: cert,
			Key:  key,
		}
	}

	return cfg, nil
}
