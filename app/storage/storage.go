package storage

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite" // sqlite driver loaded here
)

// NewSqliteDB creates a new sqlite database
func NewSqliteDB(file string) (*sqlx.DB, error) {
	conn, err := sqlx.Connect("sqlite", file)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite database: %v", err)
	}

	if conn.Ping() != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %v", err)
	}

	// WAL mode allows concurrent reads during writes
	if _, err := conn.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %v", err)
	}
	// Wait up to 5s for the lock instead of failing immediately
	if _, err := conn.Exec(`PRAGMA busy_timeout=5000`); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %v", err)
	}

	log.Printf("[info] connected to sqlite database, file: %s", file)

	return conn, err
}
