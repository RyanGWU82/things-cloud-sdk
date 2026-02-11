package sync

const schemaVersion = 1

const schema = `
-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY
);

-- Sync metadata (singleton row)
CREATE TABLE IF NOT EXISTS sync_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    history_id TEXT NOT NULL,
    server_index INTEGER NOT NULL DEFAULT 0,
    last_sync_at INTEGER
);

-- Core entities
CREATE TABLE IF NOT EXISTS areas (
    uuid TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    "index" INTEGER DEFAULT 0,
    deleted INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS tags (
    uuid TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    shortcut TEXT DEFAULT '',
    parent_uuid TEXT,
    "index" INTEGER DEFAULT 0,
    deleted INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS tasks (
    uuid TEXT PRIMARY KEY,
    type INTEGER NOT NULL DEFAULT 0,
    title TEXT NOT NULL DEFAULT '',
    note TEXT DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    schedule INTEGER NOT NULL DEFAULT 0,
    scheduled_date INTEGER,
    deadline_date INTEGER,
    completion_date INTEGER,
    creation_date INTEGER,
    modification_date INTEGER,
    "index" INTEGER DEFAULT 0,
    today_index INTEGER DEFAULT 0,
    in_trash INTEGER DEFAULT 0,
    area_uuid TEXT,
    project_uuid TEXT,
    heading_uuid TEXT,
    alarm_time_offset INTEGER,
    recurrence_rule TEXT,
    deleted INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS checklist_items (
    uuid TEXT PRIMARY KEY,
    task_uuid TEXT,
    title TEXT NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    "index" INTEGER DEFAULT 0,
    creation_date INTEGER,
    completion_date INTEGER,
    deleted INTEGER DEFAULT 0
);

-- Junction tables
CREATE TABLE IF NOT EXISTS task_tags (
    task_uuid TEXT NOT NULL,
    tag_uuid TEXT NOT NULL,
    PRIMARY KEY (task_uuid, tag_uuid)
);

CREATE TABLE IF NOT EXISTS area_tags (
    area_uuid TEXT NOT NULL,
    tag_uuid TEXT NOT NULL,
    PRIMARY KEY (area_uuid, tag_uuid)
);

-- Change log
CREATE TABLE IF NOT EXISTS change_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_index INTEGER NOT NULL,
    synced_at INTEGER NOT NULL,
    change_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_uuid TEXT NOT NULL,
    payload TEXT
);

CREATE INDEX IF NOT EXISTS idx_change_log_synced_at ON change_log(synced_at);
CREATE INDEX IF NOT EXISTS idx_change_log_entity ON change_log(entity_type, entity_uuid);
`

func (s *Syncer) migrate() error {
	// Check current version
	var version int
	err := s.db.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&version)
	if err != nil {
		// Table doesn't exist or is empty, run full migration
		if _, err := s.db.Exec(schema); err != nil {
			return err
		}
		_, err = s.db.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES (?)", schemaVersion)
		return err
	}

	// Already at current version
	if version >= schemaVersion {
		return nil
	}

	// Future: handle incremental migrations here
	return nil
}
