package repository

import (
	"database/sql"
	"encoding/json"
	"time"
)

func parseDBTime(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC()
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC()
	}
	return time.Unix(0, 0).UTC()
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func nullIntPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}

func decodeJSONMap(v sql.NullString) any {
	if !v.Valid || v.String == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(v.String), &out); err != nil {
		return map[string]any{"raw": v.String}
	}
	return out
}
