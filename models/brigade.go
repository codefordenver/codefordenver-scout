package models

type Brigade struct {
	// Discord config
	ID                        int   `gorm:"PRIMARY_KEY"`
	Name                      string `gorm:"NOT NULL;UNIQUE"`
	DisplayName               string `gorm:"NOT NULL;UNIQUE"`
	GuildID                   string `gorm:"NOT NULL;UNIQUE"`
	ActiveProjectCategoryID   string `gorm:"NOT NULL;UNIQUE"`
	InactiveProjectCategoryID string `gorm:"NOT NULL;UNIQUE"`
	NewUserRole               string `gorm:"NOT NULL;UNIQUE"`
	OnboardingRole            string `gorm:"NOT NULL;UNIQUE"`
	MemberRole                string `gorm:"NOT NULL;UNIQUE"`
	OnboardingInviteCode      string `gorm:"NOT NULL;UNIQUE"`
	OnboardingInviteCount     int   `gorm:"NOT NULL"`
	CodeOfConductMessageID    string `gorm:"NOT NULL;UNIQUE"`
	// GDrive Config
	AgendaFolderID string `gorm:"NOT NULL;UNIQUE"`
	TimezoneString string `gorm:"NOT NULL"`
	// Github Config
	GithubOrganization string `gorm:"NOT NULL;UNIQUE"`
	IssueEmoji         string `gorm:"NOT NULL"`
}
