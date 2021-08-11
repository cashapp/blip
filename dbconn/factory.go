package dbconn

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	dsndriver "github.com/go-mysql/hotswap-dsn-driver"
	_ "github.com/go-sql-driver/mysql"

	"github.com/square/blip"
	"github.com/square/blip/aws"
)

func init() {
	dsndriver.SetHotswapFunc(Repo.ReloadPassword)
}

type Factory interface {
	Make(blip.ConfigMonitor) (*sql.DB, error)
}

var _ Factory = factory{}

type factory struct {
	awsConfg aws.ConfigFactory
	modifyDB func(*sql.DB)
}

var _ Factory = factory{}

type repassFunc func() (string, error)

func NewConnFactory(awsConfg aws.ConfigFactory, modifyDB func(*sql.DB)) factory {
	return factory{
		awsConfg: awsConfg,
		modifyDB: modifyDB,
	}
}

func (f factory) Make(cfg blip.ConfigMonitor) (*sql.DB, error) {
	passwordFunc, err := f.Password(cfg)
	if err != nil {
		return nil, err
	}

	net := ""
	addr := ""
	if cfg.Socket != "" {
		net = "unix"
		addr = cfg.Socket
	} else {
		net = "tcp"
		addr = cfg.Hostname
	}
	Repo.Add(addr, passwordFunc)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	password, err := passwordFunc(ctx)
	if err != nil {
		return nil, err
	}

	cred := cfg.Username
	if password != "" {
		cred += ":" + password
	}

	params := []string{}
	if (cfg.AWS.AuthToken || cfg.AWS.PasswordSecret != "") && !cfg.AWS.DisableAutoTLS {
		params = append(params, "tls=rds")
	}
	if cfg.AWS.AuthToken {
		params = append(params, "allowCleartextPasswords=true")
	}
	if cfg.TLS.Cert != "" && cfg.TLS.Key != "" {
		// @todo
	}

	dsn := fmt.Sprintf("%s@%s(%s)/", cred, net, addr)
	if len(params) > 0 {
		dsn += "?" + strings.Join(params, "&")
	}

	db, err := sql.Open("mysql-hotswap-dsn", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)

	if f.modifyDB != nil {
		f.modifyDB(db)
	}

	return db, nil
}

func (f factory) Password(cfg blip.ConfigMonitor) (PasswordFunc, error) {
	if cfg.AWS.AuthToken {
		// Password generated as IAM auth token (valid 15 min)
		blip.Debug("password from AWS IAM auth token")
		if !cfg.AWS.DisableAutoTLS {
			aws.RegisterRDSCA()
		}
		awscfg, err := f.awsConfg.Make(cfg.AWS)
		if err != nil {
			return nil, err
		}
		token := aws.NewAuthToken(cfg.Username, cfg.Hostname, awscfg)
		return token.Password, nil
	}

	if cfg.AWS.PasswordSecret != "" {
		blip.Debug("password from AWS Secrets Manager")
		if !cfg.AWS.DisableAutoTLS {
			aws.RegisterRDSCA()
		}
		awscfg, err := f.awsConfg.Make(cfg.AWS)
		if err != nil {
			return nil, err
		}
		secret := aws.NewSecret(cfg.AWS.PasswordSecret, awscfg)
		return secret.Password, nil
	}

	if cfg.PasswordFile != "" {
		blip.Debug("password from file %s", cfg.PasswordFile)
		return func(context.Context) (string, error) {
			bytes, err := ioutil.ReadFile(cfg.PasswordFile)
			if err != nil {
				return "", err
			}
			return string(bytes), err
		}, nil
	}

	if cfg.Password != "" {
		blip.Debug("password from config")
		return func(context.Context) (string, error) { return cfg.Password, nil }, nil
	}

	if !blip.Strict {
		blip.Debug("password blank")
		return func(context.Context) (string, error) { return "", nil }, nil
	}

	return nil, fmt.Errorf("no password")
}
