package storage

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"log"
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

	log.Printf("[info] connected to sqlite database, file: %s", file)

	return conn, err
}
