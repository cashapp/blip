package heartbeat_test

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

var (
	db *sql.DB
)

// First Method that gets run before all tests.
func TestMain(m *testing.M) {
	var err error

	// Read plans from files

	// Connect to MySQL
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/?parseTime=true",
		"root",
		"test",
		"localhost",
		"33570",
	)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	code := m.Run() // run tests
	os.Exit(code)
}
