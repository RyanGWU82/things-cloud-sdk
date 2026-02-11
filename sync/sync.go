// Package sync provides a persistent sync engine for Things Cloud.
// It stores state in SQLite and surfaces semantic change events.
package sync

import (
	"database/sql"
	"time"

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

// ChangesSince returns changes that occurred after the given timestamp
func (s *Syncer) ChangesSince(timestamp time.Time) ([]Change, error) {
	rows, err := s.db.Query(`
		SELECT id, server_index, synced_at, change_type, entity_type, entity_uuid, payload
		FROM change_log
		WHERE synced_at > ?
		ORDER BY id
	`, timestamp.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanChangeLog(rows)
}

// ChangesSinceIndex returns changes that occurred after the given server index
func (s *Syncer) ChangesSinceIndex(serverIndex int) ([]Change, error) {
	rows, err := s.db.Query(`
		SELECT id, server_index, synced_at, change_type, entity_type, entity_uuid, payload
		FROM change_log
		WHERE server_index > ?
		ORDER BY id
	`, serverIndex)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanChangeLog(rows)
}

// ChangesForEntity returns all changes for a specific entity
func (s *Syncer) ChangesForEntity(entityUUID string) ([]Change, error) {
	rows, err := s.db.Query(`
		SELECT id, server_index, synced_at, change_type, entity_type, entity_uuid, payload
		FROM change_log
		WHERE entity_uuid = ?
		ORDER BY id
	`, entityUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanChangeLog(rows)
}

func (s *Syncer) scanChangeLog(rows *sql.Rows) ([]Change, error) {
	var changes []Change

	for rows.Next() {
		var id int
		var serverIndex int
		var syncedAt int64
		var changeType, entityType, entityUUID string
		var payload sql.NullString

		if err := rows.Scan(&id, &serverIndex, &syncedAt, &changeType, &entityType, &entityUUID, &payload); err != nil {
			return nil, err
		}

		base := baseChange{
			serverIndex: serverIndex,
			timestamp:   time.Unix(syncedAt, 0),
		}

		// Return UnknownChange with the change type as details
		// A more complete implementation would reconstruct full typed changes
		changes = append(changes, UnknownChange{
			baseChange: base,
			entityType: entityType,
			entityUUID: entityUUID,
			Details:    changeType,
		})
	}

	return changes, nil
}
