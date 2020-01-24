package models

import (
	"database/sql"
	"time"
)

type VolunteerSession struct {
	ID int `gorm:"not null"`
	BrigadeID string `gorm:"not null"`
	Brigade Brigade
	DiscordUserID string `gorm:"not null"`
	StartTime time.Time `gorm:"not null"`
	Duration sql.NullInt32 `gorm:"not null"`
}
