package models

type Project struct {
	ID               int `gorm:"AUTO_INCREMENT"`
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
	GithubRepository string `gorm:"not null"`
}
