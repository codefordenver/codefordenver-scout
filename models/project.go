package models

import "github.com/lib/pq"

type Project struct {
	ID               int `gorm:"primary_key;AUTO_INCREMENT"`
	BrigadeID        int `gorm:"not null"`
	Brigade          Brigade
	Name             string `gorm:"not null"`
	Description string
	Stack string
	HelpNeeded string
	DiscordChannelID string `gorm:"not null"`
	GithubDiscordChannelID string `gorm:"not null"`
	DiscordRoleID string `gorm:"not null"`
	DiscordChampionRoleID string `gorm:"not null"`
	GithubRepositoryNames pq.StringArray `gorm:"type:text[];not null"`
}
