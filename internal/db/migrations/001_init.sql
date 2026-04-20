CREATE TABLE IF NOT EXISTS stores (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	category TEXT NOT NULL,
	terminal TEXT,
	floor TEXT,
	lat REAL NOT NULL,
	lng REAL NOT NULL,
	area_id TEXT NOT NULL,
	external_ref TEXT,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS status_reports (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
	busy_level TEXT NOT NULL,
	queue_length INTEGER,
	wait_minutes_est INTEGER,
	source TEXT NOT NULL,
	reporter_id TEXT,
	reported_at TEXT NOT NULL DEFAULT (datetime('now')),
	note TEXT
);

CREATE INDEX IF NOT EXISTS idx_status_reports_store_time
	ON status_reports (store_id, reported_at DESC);

CREATE TABLE IF NOT EXISTS store_status_latest (
	store_id TEXT PRIMARY KEY REFERENCES stores(id) ON DELETE CASCADE,
	busy_level TEXT NOT NULL,
	queue_length INTEGER,
	wait_minutes_est INTEGER,
	source TEXT NOT NULL,
	as_of TEXT NOT NULL
);
