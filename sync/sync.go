// Package sync provides a persistent sync engine for Things Cloud.
// It stores state in SQLite and surfaces semantic change events.
package sync

import (
	"database/sql"

	things "github.com/nicolai86/things-cloud-sdk"
	_ "modernc.org/sqlite"
)

// Syncer manages persistent sync with Things Cloud
type Syncer struct {
	db      *sql.DB
	client  *things.Client
	history *things.History
}

// Open creates or opens a sync database and connects to Things Cloud
func Open(dbPath string, client *things.Client) (*Syncer, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	s := &Syncer{
		db:     db,
		client: client,
	}

	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// Close closes the database connection
func (s *Syncer) Close() error {
	return s.db.Close()
}
