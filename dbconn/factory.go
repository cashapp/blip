package dbconn

import (
	"database/sql"
	"fmt"

	//dsndriver "github.com/go-mysql/hotswap-dsn-driver"
	_ "github.com/go-sql-driver/mysql"

	"github.com/square/blip"
)

type Factory interface {
	Make(blip.ConfigMonitor) (*sql.DB, error)
}

var _ Factory = connFactory{}

type connFactory struct {
}

func NewConnFactory() connFactory {
	return connFactory{}
}

func (f connFactory) Make(cfg blip.ConfigMonitor) (*sql.DB, error) {
	cred := "root"

	// -------------------------------------------------------------------------
	// AWS RDS connection
	if cfg.AWS.AuthToken {
		// Password generated as IAM auth token (valid 15 min)
		token := ""
		cred = cred + ":" + token
	} else if cfg.AWS.PasswordSecret != "" {
		/*
			// Password from AWS Secrets Manager
			secret, serr = NewSecret(secretName)
			if serr != nil {
				log.Fatal(serr)
			}
			dsndriver.SetHotswapFunc(secret.SwapDSN)
			password, err := secret.Get(context.Background)
			if err != nil {
				return nil, err
			}
			cred = cred + ":" + password
		*/
	}

	cred = cfg.Username
	if cfg.Password != "" {
		cred += ":" + cfg.Password
	}

	addr := ""
	if cfg.Socket != "" {
		addr = fmt.Sprintf("unix(%s)", cfg.Socket)
	} else {
		// add :3306 if missing
		addr = fmt.Sprintf("tcp(%s)", cfg.Hostname)
	}
	dsn := fmt.Sprintf("%s@%s/", cred, addr)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)

	return db, nil
}
