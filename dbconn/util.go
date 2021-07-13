package dbconn

import (
	"strings"

	"github.com/square/blip"
)

const (
	STATE_NONE      = ""
	STATE_OFFLINE   = "offline"
	STATE_STANDBY   = "standby"
	STATE_READ_ONLY = "read-only"
	STATE_ACTIVE    = "active"
)

func SanitizeTable(table string) string {
	v := strings.SplitN(table, ".", 2)
	if len(v) == 1 {
		return "`" + blip.DEFAULT_DATABASE + "`.`" + v[0] + "`"
	}
	return "`" + v[0] + "`.`" + v[1] + "`"
}
