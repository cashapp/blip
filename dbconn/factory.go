// Package dbconn provides a Factory that makes *sql.DB connections to MySQL.
package dbconn

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/square/blip"
	"github.com/square/blip/aws"
)

// factory is the internal implementation of blip.DbFactory.
type factory struct {
	awsConfg blip.AWSConfigFactory
	modifyDB func(*sql.DB, string)
}

// NewConnFactory returns a blip.NewConnFactory that connects to MySQL.
// This is the only blip.NewConnFactor. It is created in Server.Defaults.
func NewConnFactory(awsConfg blip.AWSConfigFactory, modifyDB func(*sql.DB, string)) factory {
	return factory{
		awsConfg: awsConfg,
		modifyDB: modifyDB,
	}
}

// Make makes a *sql.DB for the given monitor config. On success, it also returns
// a print-safe DSN (with any password replaced by "..."). The config must be
// copmlete: defaults, env var, and monitor var interpolations already applied,
// which is done by the monitor.Loader in its private merge method.
func (f factory) Make(cfg blip.ConfigMonitor) (*sql.DB, string, error) {
	passwordFunc, err := f.Password(cfg)
	if err != nil {
		return nil, "", err
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
		return nil, "", err
	}

	cred := cfg.Username
	if password != "" {
		cred += ":" + password
	}

	params := []string{"parseTime=true"}
	if (blip.True(cfg.AWS.AuthToken) || cfg.AWS.PasswordSecret != "") && !blip.True(cfg.AWS.DisableAutoTLS) {
		params = append(params, "tls=rds")
	}
	if blip.True(cfg.AWS.AuthToken) {
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
		return nil, "", err
	}
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(3)

	if f.modifyDB != nil {
		f.modifyDB(db, dsn)
	}

	dsncfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		// @todo
	}
	if dsncfg.Passwd != "" {
		dsncfg.Passwd = "..."
	}

	return db, dsncfg.FormatDSN(), nil
}

func (f factory) Password(cfg blip.ConfigMonitor) (PasswordFunc, error) {
	if blip.True(cfg.AWS.AuthToken) {
		// Password generated as IAM auth token (valid 15 min)
		blip.Debug("password from AWS IAM auth token")
		if !blip.True(cfg.AWS.DisableAutoTLS) {
			aws.RegisterRDSCA()
		}
		awscfg, err := f.awsConfg.Make(blip.AWS{Region: cfg.AWS.Region})
		if err != nil {
			return nil, err
		}
		token := aws.NewAuthToken(cfg.Username, cfg.Hostname, awscfg)
		return token.Password, nil
	}

	if cfg.AWS.PasswordSecret != "" {
		blip.Debug("password from AWS Secrets Manager")
		if !blip.True(cfg.AWS.DisableAutoTLS) {
			aws.RegisterRDSCA()
		}
		awscfg, err := f.awsConfg.Make(blip.AWS{Region: cfg.AWS.Region})
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

// --------------------------------------------------------------------------

const (
	default_mysql_socket  = "/tmp/mysql.sock"
	default_distro_socket = "/var/lib/mysql/mysql.sock"
)

func Sockets() []string {
	sockets := []string{}
	seen := map[string]bool{}
	for _, socket := range strings.Split(socketList(), "\n") {
		socket = strings.TrimSpace(socket)
		if socket == "" {
			continue
		}
		if seen[socket] {
			continue
		}
		seen[socket] = true
		if !isSocket(socket) {
			continue
		}
		sockets = append(sockets, socket)
	}

	if len(sockets) == 0 {
		blip.Debug("no sockets, using defaults")
		if isSocket(default_mysql_socket) {
			sockets = append(sockets, default_mysql_socket)
		}
		if isSocket(default_distro_socket) {
			sockets = append(sockets, default_distro_socket)
		}
	}

	blip.Debug("sockets: %v", sockets)
	return sockets
}

func socketList() string {
	cmd := exec.Command("sh", "-c", "netstat -f unix | grep mysql | grep -v mysqlx | awk '{print $NF}'")
	output, err := cmd.Output()
	if err != nil {
		blip.Debug(err.Error())
	}
	return string(output)
}

func isSocket(file string) bool {
	fi, err := os.Stat(file)
	if err != nil {
		return false
	}
	return fi.Mode()&fs.ModeSocket != 0
}
