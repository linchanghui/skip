package domain

import "time"

type BusyLevel string

const (
	BusyQuiet    BusyLevel = "quiet"
	BusyModerate BusyLevel = "moderate"
	BusyBusy     BusyLevel = "busy"
	BusyClosed   BusyLevel = "closed"
)

type StatusSource string

const (
	SourceMerchant    StatusSource = "merchant"
	SourceOperator    StatusSource = "operator"
	SourceIntegration StatusSource = "integration"
	SourceCrowd       StatusSource = "crowd"
)

type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type BBox struct {
	South float64 `json:"south"`
	West  float64 `json:"west"`
	North float64 `json:"north"`
	East  float64 `json:"east"`
}

type Area struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Center      LatLng  `json:"center"`
	DefaultZoom int     `json:"default_zoom"`
	BBox        BBox    `json:"bbox"`
}

type LatestStatus struct {
	BusyLevel      BusyLevel    `json:"busy_level"`
	QueueLength    *int         `json:"queue_length"`
	WaitMinutesEst *int         `json:"wait_minutes_est"`
	AsOf           time.Time    `json:"as_of"`
	Source         StatusSource `json:"source"`
}

type Store struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	AreaID        string         `json:"area_id"`
	Terminal      *string        `json:"terminal"`
	Floor         *string        `json:"floor"`
	Category      string         `json:"category"`
	Location      LatLng         `json:"location"`
	ExternalRef   *string        `json:"external_ref"`
	LatestStatus  *LatestStatus  `json:"latest_status"`
}

type StoreDetail struct {
	Store
	StatusHistory []StatusReport `json:"status_history"`
}

type StatusReport struct {
	ID             int64          `json:"id"`
	StoreID        string         `json:"store_id"`
	BusyLevel      BusyLevel      `json:"busy_level"`
	QueueLength    *int           `json:"queue_length"`
	WaitMinutesEst *int           `json:"wait_minutes_est"`
	Source         StatusSource   `json:"source"`
	ReporterID     *string        `json:"reporter_id"`
	ReportedAt     time.Time      `json:"reported_at"`
	Note           *string        `json:"note"`
}

type StatusReportInput struct {
	BusyLevel      BusyLevel    `json:"busy_level"`
	QueueLength    *int         `json:"queue_length"`
	WaitMinutesEst *int        `json:"wait_minutes_est"`
	Source         StatusSource `json:"source"`
	Note           *string      `json:"note"`
}
