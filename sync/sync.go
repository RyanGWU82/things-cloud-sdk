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

// Sync fetches new items from Things Cloud, updates local state,
// and returns the list of changes in order
func (s *Syncer) Sync() ([]Change, error) {
	// Ensure we have a history
	if s.history == nil {
		h, err := s.client.OwnHistory()
		if err != nil {
			return nil, err
		}
		s.history = h
	}

	// Get current sync state
	storedHistoryID, startIndex, err := s.getSyncState()
	if err != nil {
		return nil, err
	}

	// If history changed, start fresh
	if storedHistoryID != "" && storedHistoryID != s.history.ID {
		startIndex = 0
	}

	// Fetch items from server
	var allChanges []Change
	hasMore := true

	for hasMore {
		items, more, err := s.history.Items(things.ItemsOptions{StartIndex: startIndex})
		if err != nil {
			return nil, err
		}
		hasMore = more

		// Process each item
		changes, err := s.processItems(items, startIndex)
		if err != nil {
			return nil, err
		}
		allChanges = append(allChanges, changes...)

		startIndex = s.history.LoadedServerIndex
	}

	// Save sync state
	if err := s.saveSyncState(s.history.ID, s.history.LatestServerIndex); err != nil {
		return nil, err
	}

	return allChanges, nil
}

// LastSyncedIndex returns the server index we've synced up to
func (s *Syncer) LastSyncedIndex() int {
	_, idx, _ := s.getSyncState()
	return idx
}
