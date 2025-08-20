package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type Activity struct {
	ID           int64          `json:"id" db:"id"`
	TenantID     string         `json:"tenant_id" db:"tenant_id"`
	Name         string         `json:"name" db:"name"`
	SKU          string         `json:"sku" db:"sku"`
	InitialStock int            `json:"initial_stock" db:"initial_stock"`
	StartAt      time.Time      `json:"start_at" db:"start_at"`
	EndAt        time.Time      `json:"end_at" db:"end_at"`
	Status       ActivityStatus `json:"status" db:"status"`
	Config       ActivityConfig `json:"config" db:"config_json"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

type ActivityStatus string

const (
	StatusDraft  ActivityStatus = "draft"
	StatusActive ActivityStatus = "active"
	StatusPaused ActivityStatus = "paused"
	StatusEnded  ActivityStatus = "ended"
)

type ActivityConfig struct {
	ReleaseRate    int  `json:"release_rate"`
	MaxConcurrent  int  `json:"max_concurrent"`
	EnableThrottle bool `json:"enable_throttle"`
	PollInterval   int  `json:"poll_interval"`
}

// 實作 database/sql/driver.Valuer 介面
func (ac ActivityConfig) Value() (driver.Value, error) {
	return json.Marshal(ac)
}

// 實作 database/sql.Scanner 介面
func (ac *ActivityConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into ActivityConfig", value)
	}

	return json.Unmarshal(bytes, ac)
}

type QueueEntry struct {
	ID          int64     `json:"id" db:"id"`
	ActivityID  int64     `json:"activity_id" db:"activity_id"`
	UserHash    string    `json:"user_hash" db:"user_hash"`
	SessionID   string    `json:"session_id" db:"session_id"`
	SeqNumber   int64     `json:"seq_number" db:"seq_number"`
	Fingerprint string    `json:"fingerprint" db:"fingerprint"`
	IPHash      string    `json:"ip_hash" db:"ip_hash"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
