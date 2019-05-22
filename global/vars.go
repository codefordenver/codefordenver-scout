package global

import "google.golang.org/api/drive/v3"

var (
	Token                  string
	NewRole                string
	OnboardingRole         string
	MemberRole             string
	OnboardingInviteCode   string
	CodeOfConductMessageID string
	AgendaFolderID         string
	InviteCount            = make(map[string]int, 0)
	DriveClient            *drive.Service
)
