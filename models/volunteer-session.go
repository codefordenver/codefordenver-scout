package models

import (
	"database/sql"
	"time"
)

type VolunteerSession struct {
	ID            int `gorm:"AUTO_INCREMENT"`
	BrigadeID     int `gorm:"not null"`
	Brigade       Brigade
	DiscordUserID string `gorm:"not null"`
	ProjectID     sql.NullInt64
	Project       Project
	StartTime     time.Time `gorm:"not null"`
	Duration      sql.NullInt64
}
