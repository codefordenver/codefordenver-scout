package models

import "time"

type Meeting struct {
	ID              int `gorm:"primary_key"`
	BrigadeID       int `gorm:"type:int REFERENCES brigades(id);not null"`
	Brigade         Brigade `gorm:"foreignkey:BrigadeID"`
	Date            time.Time `gorm:"not null"`
	AttendanceCount int       `gorm:"not null"`
}
