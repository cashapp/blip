package dbconn

import (
	"io/fs"
	"os"
	"os/exec"
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

func MonitorId(cfg blip.ConfigMonitor) string {
	switch {
	case cfg.Id != "":
		return cfg.Id
	case cfg.Hostname != "":
		return cfg.Hostname
	case cfg.Socket != "":
		return cfg.Socket
	}
	return ""
}

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
