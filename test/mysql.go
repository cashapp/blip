package test

import (
	"database/sql"
	"fmt"
)

var MySQLPort = map[string]string{
	"mysql56": "33560",
	"mysql57": "33570",
	"mysql80": "33800",
	"ps57":    "33900",
}

func Connection(distroVersion string) (string, *sql.DB, error) {
	port, ok := MySQLPort[distroVersion]
	if !ok {
		return "", nil, fmt.Errorf("invalid distro-version: %s (see dockerPorts in test/mysql.go)", distroVersion)
	}
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/?parseTime=true",
		"root",
		"test",
		"127.0.0.1",
		port,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return "", nil, err
	}
	return dsn, db, nil
}
