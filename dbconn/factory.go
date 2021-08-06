package dbconn

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"

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
	awsSess  aws.SessionFactory
	modifyDB func(*sql.DB)
}

var _ Factory = factory{}

type repassFunc func() (string, error)

func NewConnFactory(awsSess aws.SessionFactory, modifyDB func(*sql.DB)) factory {
	return factory{
		awsSess:  awsSess,
		modifyDB: modifyDB,
	}
}

func (f factory) Make(cfg blip.ConfigMonitor) (*sql.DB, error) {
	reloadFunc, err := f.Password(cfg)
	if err != nil {
		return nil, err
	}
	password, err := reloadFunc(context.Background())
	if err != nil {
		// @todo
	}

	cred := cfg.Username
	if password != "" {
		cred += ":" + password
	}

	addr := ""
	if cfg.Socket != "" {
		addr = fmt.Sprintf("unix(%s)", cfg.Socket)
	} else {
		// add :3306 if missing
		addr = fmt.Sprintf("tcp(%s)", cfg.Hostname)
	}

	// @todo TLS

	dsn := fmt.Sprintf("%s@%s/", cred, addr)

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
		//token := ""
		return nil, nil
	}

	if cfg.AWS.PasswordSecret != "" {
		blip.Debug("password from AWS Secrets Manager")
		sess, err := f.awsSess.Make(cfg.AWS)
		if err != nil {
			return nil, err
		}
		secret := aws.NewSecret(cfg.AWS.PasswordSecret, sess)
		return secret.Get, nil
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
