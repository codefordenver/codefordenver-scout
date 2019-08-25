package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/bwmarrin/discordgo"
	"github.com/codefordenver/scout/global"
	"github.com/google/go-github/github"
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

var brigades map[string]*global.Brigade
var client *github.Client
var discord *discordgo.Session

func Create(dg *discordgo.Session) error {
	colorGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

	tr := http.DefaultTransport
	credsEnv := os.Getenv("GITHUB_CREDS")
	creds, err := base64.StdEncoding.DecodeString(credsEnv)
	if err != nil {
		fmt.Println("error reading Drive client secret file,", err)
		return err
	}
	itr, err := ghinstallation.New(tr, 31388, 1101679, creds)
	if err != nil {
		fmt.Println("error creating github key", err)
		return err
	}
	client = github.NewClient(&http.Client{Transport: itr})
	discord = dg

	teamWaitlist = make(map[string]AddMemberData, 0)

	championWaitlist = make(map[string]AddChampionData, 0)

	brigades = make(map[string]*global.Brigade, 0)

	for _, brigade := range global.Brigades {
		brigades[brigade.GithubOrg] = &brigade
	}

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
	if channels, err := discord.GuildChannels(brigades[repo.Owner.Name].GuildID); err != nil {
		fmt.Println("error fetching guild channels,", err)
	} else {
		for _, channel := range channels {
			if channel.ParentID == brigades[repo.Owner.Name].ActiveProjectCategoryID && strings.Contains(strings.ToLower(repo.Name), channel.Name) {
				projectExists = true
				textChannel = channel
				if roles, err := discord.GuildRoles(brigades[repo.Owner.Name].GuildID); err != nil {
					fmt.Println("error fetching Discord roles,", err)
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
		championRole, err = discord.GuildRoleCreate(brigades[repo.Owner.Name].GuildID)
		if err != nil {
			fmt.Println("error creating role for new project,", err)
		} else {
			rolePermission := discordgo.PermissionCreateInstantInvite | discordgo.PermissionChangeNickname | discordgo.PermissionReadMessages | discordgo.PermissionSendMessages | discordgo.PermissionSendTTSMessages | discordgo.PermissionEmbedLinks | discordgo.PermissionAttachFiles | discordgo.PermissionReadMessageHistory | discordgo.PermissionMentionEveryone | discordgo.PermissionUseExternalEmojis | discordgo.PermissionAddReactions | discordgo.PermissionVoiceConnect | discordgo.PermissionVoiceSpeak
			if _, err = discord.GuildRoleEdit(brigades[repo.Owner.Name].GuildID, championRole.ID, repo.Name+"-champion", colorInt, false, rolePermission, true); err != nil {
				fmt.Println("error editing role for new project,", err)
			}
		}
		projectRole, err = discord.GuildRoleCreate(brigades[repo.Owner.Name].GuildID)
		if err != nil {
			fmt.Println("error creating role for new project,", err)
		} else {
			c = c.Lighten(.25)
			colorInt := int(c.Red)
			colorInt = (colorInt << 8) + int(c.Green)
			colorInt = (colorInt << 8) + int(c.Blue)
			rolePermission := discordgo.PermissionCreateInstantInvite | discordgo.PermissionChangeNickname | discordgo.PermissionReadMessages | discordgo.PermissionSendMessages | discordgo.PermissionSendTTSMessages | discordgo.PermissionEmbedLinks | discordgo.PermissionAttachFiles | discordgo.PermissionReadMessageHistory | discordgo.PermissionMentionEveryone | discordgo.PermissionUseExternalEmojis | discordgo.PermissionAddReactions | discordgo.PermissionVoiceConnect | discordgo.PermissionVoiceSpeak
			if _, err = discord.GuildRoleEdit(brigades[repo.Owner.Name].GuildID, projectRole.ID, repo.Name, colorInt, false, rolePermission, true); err != nil {
				fmt.Println("error editing role for new project,", err)
			}
		}
	}

	// Create Discord channel
	projectChampionOverwrite := discordgo.PermissionOverwrite{
		ID:    championRole.ID,
		Type:  "role",
		Allow: discordgo.PermissionReadMessages,
	}
	projectOverwrite := discordgo.PermissionOverwrite{
		ID:    projectRole.ID,
		Type:  "role",
		Allow: discordgo.PermissionReadMessages,
	}
	everyoneOverwrite := discordgo.PermissionOverwrite{
		ID:   brigades[repo.Owner.Name].EveryoneRole,
		Type: "role",
		Deny: discordgo.PermissionReadMessages,
	}
	channelCreateData := discordgo.GuildChannelCreateData{
		Name:     repo.Name,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: brigades[repo.Owner.Name].ActiveProjectCategoryID,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			&projectChampionOverwrite,
			&projectOverwrite,
			&everyoneOverwrite,
		},
	}
	if !projectExists {
		var err error
		textChannel, err = discord.GuildChannelCreateComplex(brigades[repo.Owner.Name].GuildID, channelCreateData)
		if err != nil {
			fmt.Println("error creating text channel for new project,", err)
		}
	}
	// Add webhook to Discord text channel
	if discordWebhook, err := discord.WebhookCreate(textChannel.ID, "github-webhook", ""); err != nil {
		fmt.Println("error creating github webhook for text channel,", err)
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
			fmt.Println("error creating Github webhook,", err)
		}
	}
	if !projectExists {
		// Send prompt to set champions in Discord text channel
		if _, err := discord.ChannelMessageSend(textChannel.ID, "@admin, use `!champions "+strings.ToLower(repo.Name)+" [list of project champion mentions]` to set champions for this project"); err != nil {
			fmt.Println("error sending project champions prompt", err)
		}
	}
	//Create Discord github channel
	githubChannelCreateData := discordgo.GuildChannelCreateData{
		Name:     repo.Name + "-github",
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: brigades[repo.Owner.Name].ActiveProjectCategoryID,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			&projectChampionOverwrite,
			&projectOverwrite,
			&everyoneOverwrite,
		},
	}
	if textChannel, err := discord.GuildChannelCreateComplex(brigades[repo.Owner.Name].GuildID, githubChannelCreateData); err != nil {
		fmt.Println("error creating github channel for new project,", err)
	} else {
		if discordWebhook, err := discord.WebhookCreate(textChannel.ID, "github-webhook", ""); err != nil {
			fmt.Println("error creating github webhook for github channel,", err)
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
				fmt.Println("error creating Github webhook,", err)
			}
		}
	}

	if channels, err := discord.GuildChannels(brigades[repo.Owner.Name].GuildID); err != nil {
		fmt.Println("error fetching guild text channels,", err)
	} else {
		sort.Slice(channels, func(i, j int) bool {
			return channels[i].Name < channels[j].Name
		})
		i := 0
		for _, channel := range channels {
			if channel.Type == discordgo.ChannelTypeGuildText && channel.ParentID == brigades[repo.Owner.Name].ActiveProjectCategoryID {
				channel.Position = i
				i++
			}
		}
		if err := discord.GuildChannelsReorder(brigades[repo.Owner.Name].GuildID, channels); err != nil {
			fmt.Println("error reordering guild text channels,", err)
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
			fmt.Println("error creating github team for new project,", err)
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
	// Delete Discord role
	if roles, err := discord.GuildRoles(brigades[repo.Owner.Name].GuildID); err != nil {
		fmt.Println("error fetching Discord roles", err)
	} else {
		for _, role := range roles {
			if role.Name == repo.Name || role.Name == repo.Name+"-champion" {
				if err = discord.GuildRoleDelete(brigades[repo.Owner.Name].GuildID, role.ID); err != nil {
					fmt.Println("error deleting role for deleted project,", err)
				}
			}
		}
	}

	// Delete Discord channel
	if channels, err := discord.GuildChannels(brigades[repo.Owner.Name].GuildID); err != nil {
		fmt.Println("error fetching Discord channels,", err)
	} else {
		for _, channel := range channels {
			if strings.HasPrefix(channel.Name, repo.Name) {
				if _, err = discord.ChannelDelete(channel.ID); err != nil {
					fmt.Println("error deleting text channel for deleted project,", err)
				}
			}
		}
	}

	// Delete Github team
	nextPage := 0
	for moreTeams := true; moreTeams; moreTeams = nextPage != 0 {
		opt := github.ListOptions{
			Page:    nextPage,
			PerPage: 100,
		}
		teams, res, err := client.Teams.ListTeams(context.Background(), repo.Owner.Name, &opt)
		if err != nil {
			fmt.Println("error fetching Github team,", err)
			nextPage = 0
		} else {
			nextPage = res.NextPage
			for _, team := range teams {
				if *team.Name == repo.Name {
					if _, err = client.Teams.DeleteTeam(context.Background(), *team.ID); err != nil {
						fmt.Println("error deleting github team for deleted project,", err)
					}
					break
				}
			}
		}
	}
}

// Dispatch a username to the appropriate waitlist
func DispatchUsername(discordUser, githubName string) []string {
	var message []string
	var validChampion, validTeamMember bool
	if _, validChampion = championWaitlist[discordUser]; validChampion {
		message = append(message, setProjectChampion(discordUser, githubName))
	}
	if _, validTeamMember = teamWaitlist[discordUser]; validTeamMember {
		message = append(message, addUserToTeam(discordUser, githubName))
	}
	if !validChampion && !validTeamMember {
		message = []string{"Was not expecting a github username from you. Have you either `!join`ed a project or been requested to be a project champion?"}
	}
	return message
}

// Sets champions for a project
func AddUserToChampionWaitlist(discordUser string, owner, project string) {
	championWaitlist[discordUser] = AddChampionData{
		Project: project,
		Owner:   owner,
	}
}

// Actually makes a user project champion
func setProjectChampion(discordUser, githubName string) string {
	opt := github.RepositoryAddCollaboratorOptions{
		Permission: "admin",
	}
	championSetData := championWaitlist[discordUser]
	if _, err := client.Repositories.AddCollaborator(context.Background(), championSetData.Owner, championSetData.Project, githubName, &opt); err != nil {
		return "Failed to give you administrator access to " + championSetData.Project + ". Please contact a brigade captain to manually add you."
	} else {
		delete(championWaitlist, discordUser)
		return "You've been added as a champion of " + championSetData.Project
	}
}

// Adds a user to the waitlist(waiting to receive their Github username for team)
func AddUserToTeamWaitlist(discordUser, owner, team string) {
	teamWaitlist[discordUser] = AddMemberData{
		Team:  team,
		Owner: owner,
	}
}

// Actually adds user to team(and therefore github org)
func addUserToTeam(discordUser, githubName string) string {
	teamAddData := teamWaitlist[discordUser]
	nextPage := 0
	for moreTeams := true; moreTeams; moreTeams = nextPage != 0 {
		opt := github.ListOptions{
			Page:    nextPage,
			PerPage: 100,
		}
		teams, res, err := client.Teams.ListTeams(context.Background(), teamAddData.Owner, &opt)
		if err != nil {
			fmt.Println("error fetching Github team,", err)
			nextPage = 0
		} else {
			nextPage = res.NextPage
			for _, team := range teams {
				if strings.ToLower(*team.Name) == strings.ToLower(teamAddData.Team) {
					opts := github.TeamAddTeamMembershipOptions{Role: "member"}
					if _, _, err = client.Teams.AddTeamMembership(context.Background(), *team.ID, githubName, &opts); err != nil {
						fmt.Println("error adding user to github team,", err)
						return "Failed to add you to the github team **" + teamAddData.Team + "**. Try again later."
					}
					delete(teamWaitlist, discordUser)
					return "You've been added to **" + teamAddData.Team + "**"
				}
			}
		}
	}
	return "Failed to find the github team for **" + teamAddData.Team + "**. Try again later."
}

// Creates an issue
func CreateIssue(text, repository string, brigade *global.Brigade) *string {
	issue := github.IssueRequest{
		Title: &text,
	}
	if _, _, err := client.Issues.Create(context.Background(), brigade.GithubOrg, repository, &issue); err != nil {
		fmt.Println("error creating github issue,", err)
		msg := "Failed to create issue"
		return &msg
	}
	return nil
}
