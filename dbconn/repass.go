package dbconn

import (
	"context"
	"sync"

	"github.com/go-sql-driver/mysql"

	"github.com/square/blip"
)

type PasswordFunc func(context.Context) (string, error)

type dsnRepo struct {
	m *sync.Map
}

var Repo = &dsnRepo{
	m: &sync.Map{},
}

func (r *dsnRepo) Add(addr string, f PasswordFunc) error {
	r.m.Store(addr, f)
	blip.Debug("added %s", addr)
	return nil
}

func (r *dsnRepo) ReloadPassword(ctx context.Context, currentDSN string) string {
	// Only return new DSN on success and password is different. Else, return
	// an empty string which makes the hotswap driver return the original driver
	// error, i.e. it's like this func was never called. Only when this func
	// returns a non-empty string does the hotswap driver use it to swap out
	// the low-level MySQL connection.
	blip.Debug("reloading %s", currentDSN)

	cfg, err := mysql.ParseDSN(currentDSN)
	if err != nil {
		blip.Debug("error parsing DSN %s: %s", currentDSN, err)
		return ""
	}
	blip.Debug("old dsn: %+v", cfg)

	v, ok := r.m.Load(cfg.Addr)
	if !ok {
		blip.Debug("no password func for %s", cfg.Addr)
		return ""
	}

	newPassword, err := v.(PasswordFunc)(ctx)
	if err != nil {
		blip.Debug("password reload error: %s", err) // @todo event
		return ""
	}

	if cfg.Passwd == newPassword {
		blip.Debug("password has not changed")
		return ""
	}

	blip.Debug("password reladed")
	cfg.Passwd = newPassword
	return cfg.FormatDSN()
}
