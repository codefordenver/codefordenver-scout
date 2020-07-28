package models

import (
	"database/sql"
	"time"
)

type VolunteerSession struct {
	ID            int `gorm:"primary_key"`
	BrigadeID     int `gorm:"type:int REFERENCES brigades(id);not null"`
	Brigade       Brigade `gorm:"foreignkey:BrigadeID"`
	DiscordUserID string `gorm:"not null"`
	ProjectID     sql.NullInt64 `gorm:"type:int REFERENCES projects(id)"`
	Project       Project `gorm:"foreignkey:ProjectID"`
	StartTime     time.Time `gorm:"not null"`
	Duration      sql.NullInt64
}
