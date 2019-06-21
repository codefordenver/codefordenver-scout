package global

import "google.golang.org/api/drive/v3"
import "github.com/google/go-github/github"

var (
	Token                  string
	NewRole                string
	OnboardingRole         string
	MemberRole             string
	OnboardingInviteCode   string
	CodeOfConductMessageID string
	AgendaFolderID         string
	LocationString         string
	PrivateKeyDir          string
	InviteCount            = make(map[string]int, 0)
	DriveClient            *drive.Service
	GithubClient           *github.Client
)
