package dbconn

import (
	"context"
	"sync"

	dsndriver "github.com/go-mysql/hotswap-dsn-driver"
	"github.com/go-sql-driver/mysql"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/event"
)

func init() {
	dsndriver.SetHotswapFunc(Repo.ReloadPassword)
}

type PasswordFunc func(context.Context) (string, error)

type repo struct {
	m *sync.Map
}

var Repo = &repo{
	m: &sync.Map{},
}

func (r *repo) Add(addr string, f PasswordFunc) error {
	r.m.Store(addr, f)
	blip.Debug("added %s", addr)
	return nil
}

func (r *repo) ReloadPassword(ctx context.Context, currentDSN string) string {
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
		event.Sendf(event.DB_RELOAD_PASSWORD_ERROR, "%s: %s", RedactedDSN(currentDSN), err.Error())
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
