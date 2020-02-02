package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/bwmarrin/discordgo"
	"github.com/codefordenver/codefordenver-scout/models"
	"github.com/codefordenver/codefordenver-scout/pkg/discord"
	"github.com/codefordenver/codefordenver-scout/pkg/shared"
	"github.com/google/go-github/github"
	"github.com/jinzhu/gorm"
	"github.com/teacat/noire"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type Repository struct {
	Name  string `json:"name"`
	Owner struct {
		Name string `json:"login"`
	} `json:"owner"`
}

type RepositoryEvent struct {
	Action          string     `json:"action"`
	EventRepository Repository `json:"repository"`
}

type AddMemberData struct {
	Team  string
	Owner string
}

type AddChampionData struct {
	Project string
	Owner   string
}

var colorGenerator *rand.Rand

var teamWaitlist map[string]AddMemberData
var championWaitlist map[string]AddChampionData

var client *github.Client
var discordClient *discordgo.Session
var db *gorm.DB

func New(dbConnection *gorm.DB, dg *discordgo.Session) error {
	db = dbConnection

	colorGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

	tr := http.DefaultTransport
	credsEnv := os.Getenv("GITHUB_CREDS")
	creds, err := base64.StdEncoding.DecodeString(credsEnv)
	if err != nil {
		fmt.Println("error reading GitHub client secret file,", err)
		return err
	}
	itr, err := ghinstallation.New(tr, 31388, 1101679, creds)
	if err != nil {
		fmt.Println("error creating GitHub key", err)
		return err
	}
	client = github.NewClient(&http.Client{Transport: itr})
	discordClient = dg

	teamWaitlist = make(map[string]AddMemberData, 0)

	championWaitlist = make(map[string]AddChampionData, 0)

	return nil
}

func HandleRepositoryEvent(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Unmarshal
	var event RepositoryEvent
	err = json.Unmarshal(b, &event)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if event.Action == "created" {
		handleRepositoryCreate(event.EventRepository)
	} else if event.Action == "deleted" {
		handleRepositoryDelete(event.EventRepository)
	}
}

// Chore tasks for creating a repository
func handleRepositoryCreate(repo Repository) {
	projectExists := false
	var championRole *discordgo.Role
	var projectRole *discordgo.Role
	var textChannel *discordgo.Channel
	/* brigade := select brigades where github_organization matches repo.owner.name */
	var brigade models.Brigade
	err := db.Where("github_organization = ?", repo.Owner.Name).First(&brigade).Error
	if err != nil {
		fmt.Println("error fetching brigade,", err)
		return
	}
	if channels, err := discordClient.GuildChannels(brigade.GuildID); err != nil {
		fmt.Println("error fetching guild channels,", err)
	} else {
		for _, channel := range channels {
			if (channel.ParentID == brigade.ActiveProjectCategoryID || channel.ParentID == brigade.InactiveProjectCategoryID) && strings.Contains(strings.ToLower(repo.Name), channel.Name) {
				projectExists = true
				textChannel = channel
				if roles, err := discordClient.GuildRoles(brigade.GuildID); err != nil {
					fmt.Println("error fetching guild roles,", err)
				} else {
					for _, role := range roles {
						if strings.ToLower(role.Name) == textChannel.Name {
							projectRole = role
						}
						if strings.ToLower(role.Name) == textChannel.Name+"-champion" {
							championRole = role
						}
					}
				}
			}
		}
	}
	// Create Discord roles
	if !projectExists {
		c := noire.NewRGB(float64(colorGenerator.Intn(256)), float64(colorGenerator.Intn(256)), float64(colorGenerator.Intn(256)))
		colorInt := int(c.Red)
		colorInt = (colorInt << 8) + int(c.Green)
		colorInt = (colorInt << 8) + int(c.Blue)
		var err error
		championRole, err = discordClient.GuildRoleCreate(brigade.GuildID)
		if err != nil {
			fmt.Println("error creating guild role,", err)
		} else {
			rolePermission := discordgo.PermissionCreateInstantInvite | discordgo.PermissionChangeNickname | discordgo.PermissionReadMessages | discordgo.PermissionSendMessages | discordgo.PermissionSendTTSMessages | discordgo.PermissionEmbedLinks | discordgo.PermissionAttachFiles | discordgo.PermissionReadMessageHistory | discordgo.PermissionMentionEveryone | discordgo.PermissionUseExternalEmojis | discordgo.PermissionAddReactions | discordgo.PermissionVoiceConnect | discordgo.PermissionVoiceSpeak
			if _, err = discordClient.GuildRoleEdit(brigade.GuildID, championRole.ID, repo.Name+"-champion", colorInt, false, rolePermission, true); err != nil {
				fmt.Println("error editing guild role,", err)
			}
		}
		projectRole, err = discordClient.GuildRoleCreate(brigade.GuildID)
		if err != nil {
			fmt.Println("error creating guild role,", err)
		} else {
			c = c.Lighten(.25)
			colorInt := int(c.Red)
			colorInt = (colorInt << 8) + int(c.Green)
			colorInt = (colorInt << 8) + int(c.Blue)
			rolePermission := discordgo.PermissionCreateInstantInvite | discordgo.PermissionChangeNickname | discordgo.PermissionReadMessages | discordgo.PermissionSendMessages | discordgo.PermissionSendTTSMessages | discordgo.PermissionEmbedLinks | discordgo.PermissionAttachFiles | discordgo.PermissionReadMessageHistory | discordgo.PermissionMentionEveryone | discordgo.PermissionUseExternalEmojis | discordgo.PermissionAddReactions | discordgo.PermissionVoiceConnect | discordgo.PermissionVoiceSpeak
			if _, err = discordClient.GuildRoleEdit(brigade.GuildID, projectRole.ID, repo.Name, colorInt, false, rolePermission, true); err != nil {
				fmt.Println("error editing guild role,", err)
			}
		}
	}

	// Create Discord channel
	projectChampionOverwrite := discordgo.PermissionOverwrite{
		ID:    championRole.ID,
		Type:  "role",
		Allow: discordgo.PermissionReadMessages | discordgo.PermissionManageWebhooks | discordgo.PermissionManageChannels,
	}
	projectOverwrite := discordgo.PermissionOverwrite{
		ID:    projectRole.ID,
		Type:  "role",
		Allow: discordgo.PermissionReadMessages,
	}
	memberOverwrite := discordgo.PermissionOverwrite {
		ID: brigade.MemberRole,
		Type: "role",
		Allow: discordgo.PermissionReadMessages,
	}
	everyoneOverwrite := discordgo.PermissionOverwrite{
		ID:   brigade.GuildID,
		Type: "role",
		Deny: discordgo.PermissionReadMessages,
	}
	channelCreateData := discordgo.GuildChannelCreateData{
		Name:     repo.Name,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: brigade.ActiveProjectCategoryID,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			&projectChampionOverwrite,
			&memberOverwrite,
			&everyoneOverwrite,
		},
	}
	if !projectExists {
		var err error
		textChannel, err = discordClient.GuildChannelCreateComplex(brigade.GuildID, channelCreateData)
		if err != nil {
			fmt.Println("error creating guild channel,", err)
		}
	}
	// Add webhook to Discord text channel
	if discordWebhook, err := discordClient.WebhookCreate(textChannel.ID, "github-webhook", ""); err != nil {
		fmt.Println("error creating guild channel webhook,", err)
	} else {
		discordWebhookURL := fmt.Sprintf("https://discordapp.com/api/webhooks/%v/%v/github", discordWebhook.ID, discordWebhook.Token)
		githubHookName := "web"
		githubHookConfig := make(map[string]interface{})
		githubHookConfig["content_type"] = "json"
		githubHookConfig["url"] = discordWebhookURL
		githubHook := github.Hook{
			Name:   &githubHookName,
			Config: githubHookConfig,
			Events: []string{"issues"},
		}

		_, _, err = client.Repositories.CreateHook(context.Background(), repo.Owner.Name, repo.Name, &githubHook)
		if err != nil {
			fmt.Println("error creating GitHub webhook,", err)
		}
	}

	//Create Discord github channel
	githubChannelCreateData := discordgo.GuildChannelCreateData{
		Name:     repo.Name + "-github",
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: brigade.ActiveProjectCategoryID,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			&projectChampionOverwrite,
			&projectOverwrite,
			&everyoneOverwrite,
		},
	}
	if textChannel, err := discordClient.GuildChannelCreateComplex(brigade.GuildID, githubChannelCreateData); err != nil {
		fmt.Println("error creating guild channel,", err)
	} else {
		if discordWebhook, err := discordClient.WebhookCreate(textChannel.ID, "github-webhook", ""); err != nil {
			fmt.Println("error creating channel webhook,", err)
		} else {
			discordWebhookURL := fmt.Sprintf("https://discordapp.com/api/webhooks/%v/%v/github", discordWebhook.ID, discordWebhook.Token)
			githubHookName := "web"
			githubHookConfig := make(map[string]interface{})
			githubHookConfig["content_type"] = "json"
			githubHookConfig["url"] = discordWebhookURL
			githubHook := github.Hook{
				Name:   &githubHookName,
				Config: githubHookConfig,
				Events: []string{"push"},
			}

			_, _, err = client.Repositories.CreateHook(context.Background(), repo.Owner.Name, repo.Name, &githubHook)
			if err != nil {
				fmt.Println("error creating GitHub webhook,", err)
			}
		}
	}

	if channels, err := discordClient.GuildChannels(brigade.GuildID); err != nil {
		fmt.Println("error fetching guild channels,", err)
	} else {
		sort.Slice(channels, func(i, j int) bool {
			return channels[i].Name < channels[j].Name
		})
		i := 0
		for _, channel := range channels {
			if channel.Type == discordgo.ChannelTypeGuildText && channel.ParentID == brigade.ActiveProjectCategoryID {
				channel.Position = i
				i++
			}
		}
		if err := discordClient.GuildChannelsReorder(brigade.GuildID, channels); err != nil {
			fmt.Println("error reordering guild channels,", err)
		}
	}

	// Create Github team
	if !projectExists {
		privacy := "closed"
		newTeam := github.NewTeam{
			Name:    repo.Name,
			Privacy: &privacy,
		}
		if team, _, err := client.Teams.CreateTeam(context.Background(), repo.Owner.Name, newTeam); err != nil {
			fmt.Println("error creating GitHub team,", err)
		} else {
			options := github.TeamAddTeamRepoOptions{Permission: "push"}
			if _, err = client.Teams.AddTeamRepo(context.Background(), *team.ID, repo.Owner.Name, repo.Name, &options); err != nil {
				fmt.Println("error adding repository to team,", err)
			}
		}
	}
}

// Chore tasks for deleting a repository
func handleRepositoryDelete(repo Repository) {
	var brigade models.Brigade
	err := db.Where("github_organization = ?", repo.Owner.Name).First(&brigade).Error
	if err != nil {
		fmt.Println("error fetching brigade,", err)
		return
	}
	// Delete Discord role
	if roles, err := discordClient.GuildRoles(brigade.GuildID); err != nil {
		fmt.Println("error fetching guild roles", err)
	} else {
		for _, role := range roles {
			if role.Name == repo.Name || role.Name == repo.Name+"-champion" {
				if err = discordClient.GuildRoleDelete(brigade.GuildID, role.ID); err != nil {
					fmt.Println("error deleting guild role,", err)
				}
			}
		}
	}

	// Delete Discord channel
	if channels, err := discordClient.GuildChannels(brigade.GuildID); err != nil {
		fmt.Println("error fetching guild channels,", err)
	} else {
		for _, channel := range channels {
			if strings.HasPrefix(channel.Name, repo.Name) {
				if _, err = discordClient.ChannelDelete(channel.ID); err != nil {
					fmt.Println("error deleting guild channel,", err)
				}
			}
		}
	}

	// Delete GitHub team
	nextPage := 0
	for moreTeams := true; moreTeams; moreTeams = nextPage != 0 {
		opt := github.ListOptions{
			Page:    nextPage,
			PerPage: 100,
		}
		teams, res, err := client.Teams.ListTeams(context.Background(), repo.Owner.Name, &opt)
		if err != nil {
			fmt.Println("error fetching GitHub teams,", err)
			nextPage = 0
		} else {
			nextPage = res.NextPage
			for _, team := range teams {
				if *team.Name == repo.Name {
					if _, err = client.Teams.DeleteTeam(context.Background(), *team.ID); err != nil {
						fmt.Println("error deleting GitHub team,", err)
					}
					break
				}
			}
		}
	}
}

// Dispatch a username to the appropriate waitlist
func DispatchUsername(data discord.MessageData, githubName string) shared.FunctionResponse {
	var validChampion, validTeamMember bool
	var errorMessage string
	var successMessage string
	if _, validChampion = championWaitlist[data.Author.ID]; validChampion {
		res := setProjectChampion(data.Author.ID, githubName)
		if res.Success != "" {
			successMessage += res.Success + "\n"
		}
		if res.Error != "" {
			errorMessage += res.Error + "\n"
		}
	}
	if _, validTeamMember = teamWaitlist[data.Author.ID]; validTeamMember {
		res := addUserToTeam(data.Author.ID, githubName)
		if res.Success != "" {
			successMessage += res.Success + "\n"
		}
		if res.Error != "" {
			errorMessage += res.Error + "\n"
		}
	}
	if !validChampion && !validTeamMember {
		return shared.FunctionResponse {
			ChannelID: data.Author.ID,
			Error: "Was not expecting a GitHub username from you. Have you either `!join`ed a project or been requested to be a project champion?",
			Success: nil,
		}
	}
	return shared.FunctionResponse {
		ChannelID: data.Author.ID,
		Success:   successMessage,
		Error:     errorMessage,
	}
}

// Sets champions for a project
func AddUserToChampionWaitlist(discordUser string, owner, project string) {
	championWaitlist[discordUser] = AddChampionData{
		Project: project,
		Owner:   owner,
	}
}

// Actually makes a user project champion
func setProjectChampion(discordUser, githubName string) shared.FunctionResponse {
	opt := github.RepositoryAddCollaboratorOptions{
		Permission: "admin",
	}
	championSetData := championWaitlist[discordUser]
	if _, err := client.Repositories.AddCollaborator(context.Background(), championSetData.Owner, championSetData.Project, githubName, &opt); err != nil {
		return shared.FunctionResponse {
			ChannelID: nil,
			Success: nil,
			Error: "Failed to give you administrator access to " + championSetData.Project + ". Please contact a brigade captain to manually add you.",
		}
	} else {
		delete(championWaitlist, discordUser)
		return shared.FunctionResponse {
			ChannelID: nil,
			Success: "You've been added as a champion of " + championSetData.Project,
			Error: nil,
		}
	}
}

// Adds a user to the waitlist(waiting to receive their GitHub username for team)
func AddUserToTeamWaitlist(discordUser, owner, team string) {
	teamWaitlist[discordUser] = AddMemberData{
		Team:  team,
		Owner: owner,
	}
}

// Actually adds user to team(and therefore GitHub org)
func addUserToTeam(discordUser, githubName string) shared.FunctionResponse {
	teamAddData := teamWaitlist[discordUser]
	nextPage := 0
	for moreTeams := true; moreTeams; moreTeams = nextPage != 0 {
		opt := github.ListOptions{
			Page:    nextPage,
			PerPage: 100,
		}
		teams, res, err := client.Teams.ListTeams(context.Background(), teamAddData.Owner, &opt)
		if err != nil {
			fmt.Println("error fetching GitHub team,", err)
			nextPage = 0
		} else {
			nextPage = res.NextPage
			for _, team := range teams {
				if strings.ToLower(*team.Name) == strings.ToLower(teamAddData.Team) {
					opts := github.TeamAddTeamMembershipOptions{Role: "member"}
					if _, _, err = client.Teams.AddTeamMembership(context.Background(), *team.ID, githubName, &opts); err != nil {
						fmt.Println("error adding user to GitHub team,", err)
						return shared.FunctionResponse {
							ChannelID: nil,
							Success: nil,
							Error: "Failed to add you to the GitHub team **" + teamAddData.Team + "**. Try again later.",
						}
					}
					delete(teamWaitlist, discordUser)
					return shared.FunctionResponse {
						ChannelID: nil,
						Success: "You've been added to **" + teamAddData.Team + "**",
						Error: nil,
					}
				}
			}
		}
	}
	return shared.FunctionResponse {
		ChannelID: nil,
		Success: nil,
		Error: "Failed to find the GitHub team for **" + teamAddData.Team + "**. Try again later.",
	}
}

// Creates an issue
func CreateIssue(text string, brigade models.Brigade, channel discordgo.Channel) []shared.FunctionResponse {
	issue := github.IssueRequest{
		Title: &text,
	}
	if issue, _, err := client.Issues.Create(context.Background(), brigade.GithubOrganization, channel.Name, &issue); err != nil {
		fmt.Println("error creating GitHub issue,", err)
		return []shared.FunctionResponse {
			{
				ChannelID: channel.ID,
				Success:   nil,
				Error:     "Failed to create GitHub issue on " + channel.Name,
			},
		}
	} else {
		return []shared.FunctionResponse {
			{
				ChannelID: channel.ID,
				Success:   "Issue created: " + *issue.URL,
				Error:     nil,
			},
		}
	}
}
