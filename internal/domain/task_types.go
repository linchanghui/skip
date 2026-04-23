package domain

import "time"

type TaskType string

const (
	TaskTypeQueueForMe TaskType = "queue_for_me"
)

type TaskStatus string

const (
	TaskStatusCreated   TaskStatus = "created"
	TaskStatusMatching  TaskStatus = "matching"
	TaskStatusAccepted  TaskStatus = "accepted"
	TaskStatusArrived   TaskStatus = "arrived"
	TaskStatusQueuing   TaskStatus = "queuing"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type RunnerStatus string

const (
	RunnerCandidate  RunnerStatus = "candidate"
	RunnerApproved   RunnerStatus = "approved"
	RunnerProbation  RunnerStatus = "probation"
	RunnerActive     RunnerStatus = "active"
	RunnerSuspended  RunnerStatus = "suspended"
	RunnerOffboarded RunnerStatus = "offboarded"
)

type ReporterType string

const (
	ReporterRunner   ReporterType = "runner"
	ReporterUser     ReporterType = "user"
	ReporterOperator ReporterType = "operator"
)

type Task struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	StoreID          string     `json:"store_id"`
	TaskType         TaskType   `json:"task_type"`
	Status           TaskStatus `json:"status"`
	AcceptedRunnerID *string    `json:"accepted_runner_id"`
	QuotedPriceCents *int       `json:"quoted_price_cents"`
	RequestedAt      time.Time  `json:"requested_at"`
	SLAAcceptBy      time.Time  `json:"sla_accept_by"`
	SLAArriveBy      *time.Time `json:"sla_arrive_by"`
	FailReason       *string    `json:"fail_reason"`
	CancelledBy      *string    `json:"cancelled_by"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type TaskEvent struct {
	ID         int64       `json:"id"`
	TaskID     string      `json:"task_id"`
	FromStatus *TaskStatus `json:"from_status"`
	ToStatus   TaskStatus  `json:"to_status"`
	ActorType  string      `json:"actor_type"`
	ActorID    *string     `json:"actor_id"`
	Payload    any         `json:"payload"`
	CreatedAt  time.Time   `json:"created_at"`
}

type TaskDetail struct {
	Task
	Events []TaskEvent `json:"events"`
}

type Runner struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Phone            string       `json:"phone"`
	Status           RunnerStatus `json:"status"`
	ServiceArea      string       `json:"service_area"`
	ReliabilityScore float64      `json:"reliability_score"`
	AgreementVersion *string      `json:"agreement_version"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

type RunnerAvailability struct {
	RunnerID     string    `json:"runner_id"`
	IsOnline     bool      `json:"is_online"`
	Location     *LatLng   `json:"location"`
	ActiveTaskID *string   `json:"active_task_id"`
	LastPingAt   time.Time `json:"last_ping_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TaskProof struct {
	ID         int64     `json:"id"`
	TaskID     string    `json:"task_id"`
	RunnerID   *string   `json:"runner_id"`
	ProofType  string    `json:"proof_type"`
	MediaURL   *string   `json:"media_url"`
	Note       *string   `json:"note"`
	CapturedAt time.Time `json:"captured_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type QueueReport struct {
	ID             int64        `json:"id"`
	StoreID        string       `json:"store_id"`
	ReporterType   ReporterType `json:"reporter_type"`
	ReporterID     *string      `json:"reporter_id"`
	QueueLength    *int         `json:"queue_length"`
	WaitMinutesEst *int         `json:"wait_minutes_est"`
	BusyLevel      BusyLevel    `json:"busy_level"`
	EvidenceURL    *string      `json:"evidence_url"`
	ConfidenceFlag string       `json:"confidence_flag"`
	ReportedAt     time.Time    `json:"reported_at"`
	ExpiresAt      time.Time    `json:"expires_at"`
	CreatedAt      time.Time    `json:"created_at"`
}

type QueueSignal struct {
	StoreID             string       `json:"store_id"`
	StatusExpired       bool         `json:"status_expired"`
	LastUpdatedAt       time.Time    `json:"last_updated_at"`
	LastUpdatedXMinsAgo int          `json:"last_updated_x_mins_ago"`
	Signal              *QueueReport `json:"signal"`
}

type MetricsSummary struct {
	TotalTasks            int  `json:"total_tasks"`
	CompletedTasks        int  `json:"completed_tasks"`
	CompletionRatePct     int  `json:"completion_rate_pct"`
	ReassignedTasks       int  `json:"reassigned_tasks"`
	ReassignmentRatePct   int  `json:"reassignment_rate_pct"`
	AcceptP50Seconds      *int `json:"accept_p50_seconds"`
	TotalQueueReports     int  `json:"total_queue_reports"`
	ExpiredQueueReports   int  `json:"expired_queue_reports"`
	ExpiredSignalRatioPct int  `json:"expired_signal_ratio_pct"`
}

type CreateTaskInput struct {
	UserID   string   `json:"user_id"`
	StoreID  string   `json:"store_id"`
	TaskType TaskType `json:"task_type"`
	Note     *string  `json:"note"`
}

type CancelTaskInput struct {
	UserID string `json:"user_id"`
}

type RunnerApplyInput struct {
	Name        string  `json:"name"`
	Phone       string  `json:"phone"`
	ServiceArea *string `json:"service_area"`
}

type RunnerAvailabilityInput struct {
	IsOnline bool    `json:"is_online"`
	Location *LatLng `json:"location"`
}

type TaskActionByRunnerInput struct {
	RunnerID string  `json:"runner_id"`
	Note     *string `json:"note"`
}

type CreateTaskProofInput struct {
	RunnerID  string  `json:"runner_id"`
	ProofType string  `json:"proof_type"`
	MediaURL  *string `json:"media_url"`
	Note      *string `json:"note"`
}

type CreateQueueReportInput struct {
	ReporterType   ReporterType `json:"reporter_type"`
	ReporterID     *string      `json:"reporter_id"`
	QueueLength    *int         `json:"queue_length"`
	WaitMinutesEst *int         `json:"wait_minutes_est"`
	BusyLevel      BusyLevel    `json:"busy_level"`
	EvidenceURL    *string      `json:"evidence_url"`
	TTLMinutes     *int         `json:"ttl_minutes"`
}

type AssignTaskInput struct {
	RunnerID string `json:"runner_id"`
	OpsID    string `json:"ops_id"`
}
