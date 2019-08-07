package global

import (
	"github.com/bwmarrin/discordgo"
	"github.com/google/go-github/github"
	"google.golang.org/api/drive/v3"
)

type Brigade struct {
	// Discord config
	GuildID                string `yaml:"GuildID"`
	ProjectCategoryID      string `yaml:"ProjectCategoryID"`
	EveryoneRole           string `yaml:"EveryoneRole"`
	NewRole                string `yaml:"NewRole"`
	OnboardingRole         string `yaml:"OnboardingRole"`
	MemberRole             string `yaml:"MemberRole"`
	OnboardingInviteCode   string `yaml:"OnboardingInviteCode"`
	CodeOfConductMessageID string `yaml:"CodeOfConductMessageID"`
	InviteCount            int    `yaml:"InviteCount"`
	// GDrive Config
	AgendaFolderID string            `yaml:"AgendaFolderID"`
	LocationString string            `yaml:"LocationString"`
	Files          map[string]string `yaml:"Files"`
	// Github Config
	GithubOrg  string `yaml:"GithubOrg"`
	IssueEmoji string `yaml:"IssueEmoji"`
}

var (
	LocationString string
	Brigades       []Brigade
	DriveClient    *drive.Service
	GithubClient   *github.Client
	DiscordClient  *discordgo.Session
)
