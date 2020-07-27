package models

import (
	"database/sql"
	"github.com/lib/pq"
)

type Project struct {
	ID                     int `gorm:"PRIMARY_KEY"`
	BrigadeID              int `gorm:"type:int REFERENCES brigades(id);NOT NULL;"`
	Brigade                Brigade
	Name                   string `gorm:"NOT NULL"`
	DisplayName            string `gorm:"NOT NULL"`
	Description            sql.NullString
	Stack                  sql.NullString
	HelpNeeded             sql.NullString
	DiscordChannelID       string         `gorm:"NOT NULL"`
	GithubDiscordChannelID string         `gorm:"NOT NULL"`
	DiscordRoleID          string         `gorm:"NOT NULL"`
	DiscordChampionRoleID  string         `gorm:"NOT NULL"`
	GithubRepositoryNames  pq.StringArray `gorm:"type:text[];NOT NULL"`
}
