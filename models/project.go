package models

type Project struct {
	ID               int `gorm:"primary_key"`
	BrigadeID        int `gorm:"type:int REFERENCES brigades(id);not null"`
	Brigade          Brigade `gorm:"foreignkey:BrigadeID"`
	Name             string `gorm:"not null"`
	Description string
	Stack string
	HelpNeeded string
	DiscordChannelID string `gorm:"not null"`
	GithubDiscordChannelID string `gorm:"not null"`
	DiscordRoleID string `gorm:"not null"`
	DiscordChampionRoleID string `gorm:"not null"`
	GithubRepositoryNames string `gorm:"not null"`
}
