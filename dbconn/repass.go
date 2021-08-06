package dbconn

import (
	"context"
	"regexp"
	"sync"

	"github.com/square/blip"
)

type PasswordFunc func(context.Context) (string, error)

type dsnRepo struct {
	m *sync.Map
}

var Repo *dsnRepo

func init() {
	Repo = &dsnRepo{
		m: &sync.Map{},
	}
}

var addr = regexp.MustCompile(`\(([^\)]+)\)/`)

func (r *dsnRepo) ReloadPassword(ctx context.Context, dsn string) string {
	m := addr.FindStringSubmatch(dsn)
	if len(m) != 2 {
		return ""
	}
	v, ok := r.m.Load(m[1])
	if !ok {
		return ""
	}
	password, err := v.(PasswordFunc)(ctx)
	if err != nil {
		blip.Debug("password reload error: %s", err) // @todo event
	}
	return password
}

func (r *dsnRepo) Add(addr string, f PasswordFunc) error {
	r.m.Store(addr, f)
	return nil
}
