package global

import (
	"github.com/bwmarrin/discordgo"
	"github.com/google/go-github/github"
	"google.golang.org/api/drive/v3"
)

var (
	Token                  string
	NewRole                string
	OnboardingRole         string
	MemberRole             string
	EveryoneRole           = make(map[string]string, 0)
	OnboardingInviteCode   string
	CodeOfConductMessageID string
	AgendaFolderID         string
	LocationString         string
	PrivateKeyDir          string
	DiscordGuildId         string
	ProjectCategoryId      string
	InviteCount            = make(map[string]int, 0)
	DriveClient            *drive.Service
	GithubClient           *github.Client
	DiscordClient          *discordgo.Session
)
