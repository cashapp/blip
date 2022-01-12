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

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/aws"
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
	// ----------------------------------------------------------------------
	// my.cnf

	// Set values in cfg blip.ConfigMonitor from values in my.cnf. This does
	// not overwrite any values in cfg already set. For exmaple, if username
	// is specified in both, the default my.cnf username is ignored and the
	// explicit cfg.Username is kept/used.
	if cfg.MyCnf != "" {
		def, err := ParseMyCnf(cfg.MyCnf)
		if err != nil {
			return nil, "", err
		}
		tls := blip.ConfigTLS{
			Cert: def.TLSCert, // ssl-cert in my.cnf
			Key:  def.TLSKey,  // ssl-key in my.cnf
			CA:   def.TLSCA,   // ssl-ca in my.cnf
		}
		cfg.ApplyDefaults(blip.Config{MySQL: def, TLS: tls})
	}

	// ----------------------------------------------------------------------
	// TCP or Unix socket

	net := ""
	addr := ""
	if cfg.Socket != "" {
		net = "unix"
		addr = cfg.Socket
	} else {
		net = "tcp"
		addr = cfg.Hostname
	}

	// ----------------------------------------------------------------------
	// Pasword reload func

	// Blip presumes that passwords are rotated for security. So we create
	// a callback that relaods the password based on its method: static, file,
	// Amazon IAM auth token, etc. The special mysql-hotswap-dsn driver (below)
	// calls this func when MySQL returns an authentication error.
	passwordFunc, err := f.Password(cfg)
	if err != nil {
		return nil, "", err
	}

	// Test the password reload func, i.e. get the current password, which
	// might just be a static password in the Blip config file or another file,
	// but it could be something dynamic like an Amazon IAM auth token.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	password, err := passwordFunc(ctx)
	if err != nil {
		return nil, "", err
	}

	// Credentials are username:password--part of the DSN created below
	cred := cfg.Username
	if password != "" {
		cred += ":" + password
	}

	// ----------------------------------------------------------------------
	// DSN params (including TLS)

	params := []string{"parseTime=true"}

	// Load and register TLS, if any
	tlsConfig, err := cfg.TLS.LoadTLS()
	if tlsConfig != nil && err != nil {
		mysql.RegisterTLSConfig(cfg.MonitorId, tlsConfig)
		params = append(params, "tls="+cfg.MonitorId)
		blip.Debug("TLS enabled for %s", cfg.MonitorId)
	}

	// Use built-in Amazon RDS CA if password is AWS IAM auth or Secrets Manager
	// and auto-TLS is still enabled (default) and user didn't provide an explicit
	// TLS config (above). This latter is really forward-looking: Amazon rotates
	// its certs, so eventually the Blip built-in will be out of date. But user
	// will never be blocked (waiting for a new Blip release) because they can
	// override the built-in Amazon cert.
	if (blip.True(cfg.AWS.AuthToken) || cfg.AWS.PasswordSecret != "") &&
		!blip.True(cfg.AWS.DisableAutoTLS) &&
		tlsConfig == nil {
		aws.RegisterRDSCA() // safe to call multiple times
		params = append(params, "tls=rds")
	}

	// IAM auto requires cleartext passwords (the auth token is already encrypted)
	if blip.True(cfg.AWS.AuthToken) {
		params = append(params, "allowCleartextPasswords=true")
	}

	// ----------------------------------------------------------------------
	// Create DSN and *sql.DB

	dsn := fmt.Sprintf("%s@%s(%s)/", cred, net, addr)
	if len(params) > 0 {
		dsn += "?" + strings.Join(params, "&")
	}

	// mysql-hotswap-dsn is a special driver; see reload_password.go.
	// Remember: this does NOT connect to MySQL; it only creates a valid
	// *sql.DB connection pool. Since the caller is Monitor.Run (indirectly
	// via the blip.DbFactory it was given), actually connecting to MySQL
	// happens (probably) by monitor/Engine.Prepare, or possibly by other
	// components (plan loader, LPA, heartbeat, etc.)
	db, err := sql.Open("mysql-hotswap-dsn", dsn)
	if err != nil {
		return nil, "", err
	}

	// ======================================================================
	// Valid db/DSN, do not return error past here
	// ======================================================================

	// Now that we know the DSN/DB are valid, registry the password reload func.
	// Don't do this earlier becuase there's no way to unregister it, which is
	// probably a bug/leak if/when Blip allows dyanmically unloading monitors.
	Repo.Add(addr, passwordFunc)

	// Limit Blip to 3 MySQL conn by default: 1 or 2 for metrics, and 1 for
	// LPA, heartbeat, etc. Since all metrics are supposed to collect in a
	// matter of milliseconds, 3 should be more than enough.
	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(3)

	// Let user-provided plugin set/change DB
	if f.modifyDB != nil {
		f.modifyDB(db, dsn)
	}

	return db, RedactedDSN(dsn), nil
}

// Password creates a password reload function (callback) based on the
// configured password method. This function is used by the mysql-hotswap-dsn
// driver (see reload_password.go). For a consistent abstraction, all
// passwords are fetched via a reload func, even a static password specified
// in the Blip config file.
func (f factory) Password(cfg blip.ConfigMonitor) (PasswordFunc, error) {

	// Amazon IAM auth token (valid 15 min)
	if blip.True(cfg.AWS.AuthToken) {
		blip.Debug("%s: AWS IAM auth token password", cfg.MonitorId)
		awscfg, err := f.awsConfg.Make(blip.AWS{Region: cfg.AWS.Region})
		if err != nil {
			return nil, err
		}
		token := aws.NewAuthToken(cfg.Username, cfg.Hostname, awscfg)
		return token.Password, nil
	}

	// Amazon Secrets Manager, could be rotated
	if cfg.AWS.PasswordSecret != "" {
		blip.Debug("%s: AWS Secrets Manager password", cfg.MonitorId)
		awscfg, err := f.awsConfg.Make(blip.AWS{Region: cfg.AWS.Region})
		if err != nil {
			return nil, err
		}
		secret := aws.NewSecret(cfg.AWS.PasswordSecret, awscfg)
		return secret.Password, nil
	}

	// Password file, could be "rotated" (new password written to file)
	if cfg.PasswordFile != "" {
		blip.Debug("%s: password file", cfg.MonitorId)
		return func(context.Context) (string, error) {
			bytes, err := ioutil.ReadFile(cfg.PasswordFile)
			if err != nil {
				return "", err
			}
			return string(bytes), err
		}, nil
	}

	// Static password in my.cnf file, could be rotated (like password file)
	if cfg.MyCnf != "" {
		blip.Debug("%s my.cnf password", cfg.MonitorId)
		return func(context.Context) (string, error) {
			cfg, err := ParseMyCnf(cfg.MyCnf)
			if err != nil {
				return "", err
			}
			return cfg.Password, err
		}, nil
	}

	// Static password in Blip config file, not rotated
	if cfg.Password != "" {
		blip.Debug("%s: static password", cfg.MonitorId)
		return func(context.Context) (string, error) { return cfg.Password, nil }, nil
	}

	blip.Debug("%s: no password", cfg.MonitorId)
	return func(context.Context) (string, error) { return "", nil }, nil
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

func RedactedDSN(dsn string) string {
	redactedPassword, err := mysql.ParseDSN(dsn)
	if err != nil { // ok to ignore
		blip.Debug("mysql.ParseDSN error: %s", err)
	}
	redactedPassword.Passwd = "..."
	return redactedPassword.FormatDSN()

}
