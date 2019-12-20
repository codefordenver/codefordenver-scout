package global

type Brigade struct {
	// Discord config
	GuildID                   string `yaml:"GuildID"`
	ActiveProjectCategoryID   string `yaml:"ActiveProjectCategoryID"`
	InactiveProjectCategoryID string `yaml:"InactiveProjectCategoryID"`
	EveryoneRole              string `yaml:"EveryoneRole"`
	NewRole                   string `yaml:"NewRole"`
	OnboardingRole            string `yaml:"OnboardingRole"`
	MemberRole                string `yaml:"MemberRole"`
	OnboardingInviteCode      string `yaml:"OnboardingInviteCode"`
	CodeOfConductMessageID    string `yaml:"CodeOfConductMessageID"`
	InviteCount               int    `yaml:"InviteCount"`
	AirtableBaseID            string `yaml:"AirtableBaseID"`
	// GDrive Config
	AgendaFolderID string `yaml:"AgendaFolderID"`
	LocationString string `yaml:"LocationString"`
	// Github Config
	GithubOrg  string `yaml:"GithubOrg"`
	IssueEmoji string `yaml:"IssueEmoji"`
}

var (
	AirtableKey    string
	Brigades       []Brigade
)
