package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Campaign struct {
	PID           string    `gorm:"primaryKey" json:"pid"`
	Name          string    `gorm:"not null" json:"name"`
	Blurb         string    `json:"blurb"`
	PhotoURL      string    `json:"photo_url"`
	GoalAmount    float64   `json:"goal_amount"`
	GoalCurrency  string    `json:"goal_currency"`
	PledgedAmount float64   `json:"pledged_amount"`
	Deadline      time.Time `json:"deadline"`
	State         string    `json:"state"`
	CategoryID    string    `json:"category_id"`
	CategoryName  string    `json:"category_name"`
	ProjectURL    string    `json:"project_url"`
	CreatorName   string    `json:"creator_name"`
	PercentFunded float64   `json:"percent_funded"`
	Slug          string    `json:"slug"`
	FirstSeenAt   time.Time `gorm:"not null;default:now()" json:"first_seen_at"`
	LastUpdatedAt time.Time `gorm:"not null;default:now()" json:"last_updated_at"`
}

type Category struct {
	ID       string `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"not null" json:"name"`
	ParentID string `json:"parent_id,omitempty"`
}

type Device struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	DeviceToken string    `gorm:"uniqueIndex;not null" json:"device_token"`
	CreatedAt   time.Time `json:"created_at"`
}

func (d *Device) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

type Alert struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	DeviceID      uuid.UUID  `gorm:"type:uuid;index;not null" json:"device_id"`
	Keyword       string     `gorm:"not null" json:"keyword"`
	CategoryID    string     `json:"category_id,omitempty"`
	MinPercent    float64    `gorm:"default:0" json:"min_percent"`
	IsEnabled     bool       `gorm:"default:true" json:"is_enabled"`
	CreatedAt     time.Time  `json:"created_at"`
	LastMatchedAt *time.Time `json:"last_matched_at,omitempty"`
}

func (a *Alert) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

type AlertMatch struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	AlertID     uuid.UUID `gorm:"type:uuid;index;not null" json:"alert_id"`
	CampaignPID string    `json:"campaign_pid"`
	MatchedAt   time.Time `gorm:"default:now()" json:"matched_at"`
}

func (am *AlertMatch) BeforeCreate(tx *gorm.DB) error {
	if am.ID == uuid.Nil {
		am.ID = uuid.New()
	}
	return nil
}
