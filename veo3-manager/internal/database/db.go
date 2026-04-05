package database

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS tasks (
    id            TEXT PRIMARY KEY,
    prompt        TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    aspect_ratio  TEXT NOT NULL DEFAULT '16:9',
    model         TEXT NOT NULL DEFAULT 'veo_3_1_t2v_fast_ultra',
    output_count  INTEGER NOT NULL DEFAULT 4,
    media_ids     TEXT DEFAULT '[]',
    video_paths   TEXT DEFAULT '[]',
    error_message TEXT DEFAULT '',
    seed          TEXT DEFAULT '',
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at  DATETIME
);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT OR IGNORE INTO settings (key, value) VALUES
    ('chrome_path', ''),
    ('user_data_dir', ''),
    ('download_folder', ''),
    ('debug_port', '9222'),
    ('aspect_ratio', '16:9'),
    ('model', 'veo_3_1_t2v_fast_ultra'),
    ('output_count', '4'),
    ('delay_between_tasks', '5');
`

type DB struct {
	conn *sql.DB
}

func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for concurrent read/write
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, err
	}

	// Run schema migration
	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, err
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
