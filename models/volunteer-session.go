package models

import (
	"database/sql"
	"time"
)

type VolunteerSession struct {
	ID            int `gorm:"PRIMARY_KEY"`
	BrigadeID     int `gorm:"type:int REFERENCES brigades(id);NOT NULL;"`
	Brigade       Brigade `gorm:"FOREIGNKEY:BrigadeID"`
	DiscordUserID string `gorm:"NOT NULL"`
	ProjectID     sql.NullInt64 `gorm:"type:int REFERENCES projects(id)"`
	Project       Project `gorm:"FOREIGNKEY:ProjectID"`
	StartTime     time.Time `gorm:"NOT NULL"`
	Duration      sql.NullInt64
}
