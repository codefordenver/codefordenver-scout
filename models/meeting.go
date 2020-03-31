package models

import "time"

type Meeting struct {
	ID              int `gorm:"PRIMARY_KEY"`
	BrigadeID       int `gorm:"type:int REFERENCES brigades(id);NOT NULL;"`
	Brigade         Brigade
	Date            time.Time `gorm:"NOT NULL"`
	AttendanceCount int       `gorm:"NOT NULL"`
}
