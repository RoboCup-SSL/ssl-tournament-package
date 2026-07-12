// Package store owns the SQLite tournament database: opening it with the
// right pragmas, applying migrations, and providing typed accessors.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Store wraps the tournament database.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at path. Every pooled connection
// gets foreign-key enforcement, WAL journaling, and a 5s busy timeout.
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)",
		path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// Ping verifies the database is reachable.
func (s *Store) Ping() error { return s.db.Ping() }
