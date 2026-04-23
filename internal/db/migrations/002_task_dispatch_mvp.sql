CREATE TABLE IF NOT EXISTS runners (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	phone TEXT NOT NULL UNIQUE,
	status TEXT NOT NULL CHECK (status IN (
		'candidate','approved','probation','active','suspended','offboarded'
	)),
	service_area TEXT NOT NULL DEFAULT 'changi',
	reliability_score REAL NOT NULL DEFAULT 0.5,
	agreement_version TEXT,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_runners_status_area
	ON runners (status, service_area);

CREATE TABLE IF NOT EXISTS runner_availability (
	runner_id TEXT PRIMARY KEY REFERENCES runners(id) ON DELETE CASCADE,
	is_online INTEGER NOT NULL CHECK (is_online IN (0,1)),
	last_ping_at TEXT NOT NULL DEFAULT (datetime('now')),
	current_lng REAL,
	current_lat REAL,
	active_task_id TEXT,
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_runner_availability_online
	ON runner_availability (is_online, updated_at DESC);

CREATE TABLE IF NOT EXISTS tasks (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
	task_type TEXT NOT NULL CHECK (task_type IN ('queue_for_me')),
	status TEXT NOT NULL CHECK (status IN (
		'created','matching','accepted','arrived','queuing','completed','failed','cancelled'
	)),
	requested_at TEXT NOT NULL,
	accepted_runner_id TEXT REFERENCES runners(id) ON DELETE SET NULL,
	quoted_price_cents INTEGER,
	sla_accept_by TEXT NOT NULL,
	sla_arrive_by TEXT,
	fail_reason TEXT,
	cancelled_by TEXT,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_tasks_status_created
	ON tasks (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tasks_runner_status
	ON tasks (accepted_runner_id, status);

CREATE TABLE IF NOT EXISTS task_attempts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	attempt_no INTEGER NOT NULL,
	strategy TEXT NOT NULL CHECK (strategy IN ('auto_batch','manual_assign')),
	candidate_runner_ids TEXT,
	selected_runner_id TEXT REFERENCES runners(id) ON DELETE SET NULL,
	result TEXT NOT NULL CHECK (result IN ('pending','accepted','timeout','rejected','cancelled')),
	started_at TEXT NOT NULL DEFAULT (datetime('now')),
	ended_at TEXT,
	UNIQUE(task_id, attempt_no)
);

CREATE INDEX IF NOT EXISTS idx_task_attempts_task
	ON task_attempts (task_id, attempt_no DESC);

CREATE TABLE IF NOT EXISTS task_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	from_status TEXT,
	to_status TEXT NOT NULL,
	actor_type TEXT NOT NULL CHECK (actor_type IN ('user','runner','system','ops')),
	actor_id TEXT,
	payload TEXT,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_task_events_task_time
	ON task_events (task_id, created_at DESC);

CREATE TABLE IF NOT EXISTS task_proofs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	runner_id TEXT REFERENCES runners(id) ON DELETE SET NULL,
	proof_type TEXT NOT NULL CHECK (proof_type IN (
		'arrived_photo','queue_progress_photo','completion_photo','text_note'
	)),
	media_url TEXT,
	note TEXT,
	captured_at TEXT NOT NULL DEFAULT (datetime('now')),
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_task_proofs_task_time
	ON task_proofs (task_id, captured_at DESC);

CREATE TABLE IF NOT EXISTS queue_reports (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	store_id TEXT NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
	reporter_type TEXT NOT NULL CHECK (reporter_type IN ('runner','user','operator')),
	reporter_id TEXT,
	queue_length INTEGER,
	wait_minutes_est INTEGER,
	busy_level TEXT NOT NULL CHECK (busy_level IN ('quiet','moderate','busy','closed')),
	evidence_url TEXT,
	confidence_flag TEXT NOT NULL DEFAULT 'normal' CHECK (confidence_flag IN ('normal','low')),
	reported_at TEXT NOT NULL DEFAULT (datetime('now')),
	expires_at TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_queue_reports_store_time
	ON queue_reports (store_id, reported_at DESC);

CREATE INDEX IF NOT EXISTS idx_queue_reports_store_expiry
	ON queue_reports (store_id, expires_at DESC);
