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

var colorGenerator *rand.Rand

var expectedUsernames map[string]AddMemberData

func Create() (*github.Client, error) {
	colorGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

	tr := http.DefaultTransport
	credsEnv := os.Getenv("GITHUB_CREDS")
	creds, err := base64.StdEncoding.DecodeString(credsEnv)
	if err != nil {
		fmt.Println("error reading Drive client secret file,", err)
		return nil, err
	}
	itr, err := ghinstallation.New(tr, 31388, 1101679, creds)
	if err != nil {
		fmt.Println("error creating github key", err)
		return nil, err
	}
	client := github.NewClient(&http.Client{Transport: itr})

	expectedUsernames = make(map[string]AddMemberData, 0)

	return client, nil
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
	// Create Discord roles
	color := noire.NewRGB(float64(colorGenerator.Intn(256)), float64(colorGenerator.Intn(256)), float64(colorGenerator.Intn(256)))
	colorInt := int(color.Red)
	colorInt = (colorInt << 8) + int(color.Green)
	colorInt = (colorInt << 8) + int(color.Blue)
	championRole, err := global.DiscordClient.GuildRoleCreate(global.DiscordGuildId)
	if err != nil {
		fmt.Println("error creating role for new project,", err)
	} else {
		rolePermission := discordgo.PermissionCreateInstantInvite | discordgo.PermissionChangeNickname | discordgo.PermissionReadMessages | discordgo.PermissionSendMessages | discordgo.PermissionSendTTSMessages | discordgo.PermissionEmbedLinks | discordgo.PermissionAttachFiles | discordgo.PermissionReadMessageHistory | discordgo.PermissionMentionEveryone | discordgo.PermissionUseExternalEmojis | discordgo.PermissionAddReactions | discordgo.PermissionVoiceConnect | discordgo.PermissionVoiceSpeak
		if _, err = global.DiscordClient.GuildRoleEdit(global.DiscordGuildId, championRole.ID, repo.Name+"-champion", colorInt, false, rolePermission, true); err != nil {
			fmt.Println("error editing role for new project,", err)
		}
	}
	projectRole, err := global.DiscordClient.GuildRoleCreate(global.DiscordGuildId)
	if err != nil {
		fmt.Println("error creating role for new project,", err)
	} else {
		color = color.Lighten(.25)
		colorInt := int(color.Red)
		colorInt = (colorInt << 8) + int(color.Green)
		colorInt = (colorInt << 8) + int(color.Blue)
		rolePermission := discordgo.PermissionCreateInstantInvite | discordgo.PermissionChangeNickname | discordgo.PermissionReadMessages | discordgo.PermissionSendMessages | discordgo.PermissionSendTTSMessages | discordgo.PermissionEmbedLinks | discordgo.PermissionAttachFiles | discordgo.PermissionReadMessageHistory | discordgo.PermissionMentionEveryone | discordgo.PermissionUseExternalEmojis | discordgo.PermissionAddReactions | discordgo.PermissionVoiceConnect | discordgo.PermissionVoiceSpeak
		if _, err = global.DiscordClient.GuildRoleEdit(global.DiscordGuildId, projectRole.ID, repo.Name, colorInt, false, rolePermission, true); err != nil {
			fmt.Println("error editing role for new project,", err)
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
		ID:   global.EveryoneRole[global.DiscordGuildId],
		Type: "role",
		Deny: discordgo.PermissionReadMessages,
	}
	channelCreateData := discordgo.GuildChannelCreateData{
		Name:     repo.Name,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: global.ProjectCategoryId,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			&projectChampionOverwrite,
			&projectOverwrite,
			&everyoneOverwrite,
		},
	}
	if textChannel, err := global.DiscordClient.GuildChannelCreateComplex(global.DiscordGuildId, channelCreateData); err != nil {
		fmt.Println("error creating text channel for new project,", err)
	} else {
		if discordWebhook, err := global.DiscordClient.WebhookCreate(textChannel.ID, "github-webhook", ""); err != nil {
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

			_, _, err = global.GithubClient.Repositories.CreateHook(context.Background(), repo.Owner.Name, repo.Name, &githubHook)
			if err != nil {
				fmt.Println("error creating Github webhook,", err)
			}
		}
	}

	//Create Discord github channel
	githubChannelCreateData := discordgo.GuildChannelCreateData{
		Name:     repo.Name + "-github",
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: global.ProjectCategoryId,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			&projectChampionOverwrite,
			&projectOverwrite,
			&everyoneOverwrite,
		},
	}
	if textChannel, err := global.DiscordClient.GuildChannelCreateComplex(global.DiscordGuildId, githubChannelCreateData); err != nil {
		fmt.Println("error creating github channel for new project,", err)
	} else {
		if discordWebhook, err := global.DiscordClient.WebhookCreate(textChannel.ID, "github-webhook", ""); err != nil {
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

			_, _, err = global.GithubClient.Repositories.CreateHook(context.Background(), repo.Owner.Name, repo.Name, &githubHook)
			if err != nil {
				fmt.Println("error creating Github webhook,", err)
			}
		}
	}

	if channels, err := global.DiscordClient.GuildChannels(global.DiscordGuildId); err != nil {
		fmt.Println("error fetching guild text channels,", err)
	} else {
		sort.Slice(channels, func(i, j int) bool {
			return channels[i].Name < channels[j].Name
		})
		i := 0
		for _, channel := range channels {
			if channel.Type == discordgo.ChannelTypeGuildText && channel.ParentID == global.ProjectCategoryId {
				channel.Position = i
				i++
			}
		}
		if err := global.DiscordClient.GuildChannelsReorder(global.DiscordGuildId, channels); err != nil {
			fmt.Println("error reordering guild text channels,", err)
		}
	}

	// Create Github team
	privacy := "closed"
	newTeam := github.NewTeam{
		Name:    repo.Name,
		Privacy: &privacy,
	}
	if team, _, err := global.GithubClient.Teams.CreateTeam(context.Background(), repo.Owner.Name, newTeam); err != nil {
		fmt.Println("error creating github team for new project,", err)
	} else {
		options := github.TeamAddTeamRepoOptions{Permission: "push"}
		if _, err = global.GithubClient.Teams.AddTeamRepo(context.Background(), *team.ID, repo.Owner.Name, repo.Name, &options); err != nil {
			fmt.Println("error adding repository to team,", err)
		}
	}
}

// Chore tasks for deleting a repository
func handleRepositoryDelete(repo Repository) {
	// Delete Discord role
	if roles, err := global.DiscordClient.GuildRoles(global.DiscordGuildId); err != nil {
		fmt.Println("error fetching Discord roles")
	} else {
		for _, role := range roles {
			if role.Name == repo.Name || role.Name == repo.Name+"-champion" {
				if err = global.DiscordClient.GuildRoleDelete(global.DiscordGuildId, role.ID); err != nil {
					fmt.Println("error deleting role for deleted project,", err)
				}
			}
		}
	}

	// Delete Discord channel
	if channels, err := global.DiscordClient.GuildChannels(global.DiscordGuildId); err != nil {
		fmt.Println("error fetching Discord channels,", err)
	} else {
		for _, channel := range channels {
			if strings.HasPrefix(channel.Name, repo.Name) {
				if _, err = global.DiscordClient.ChannelDelete(channel.ID); err != nil {
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
		teams, res, err := global.GithubClient.Teams.ListTeams(context.Background(), repo.Owner.Name, &opt)
		if err != nil {
			fmt.Println("error fetching Github team,", err)
			nextPage = 0
		} else {
			nextPage = res.NextPage
			for _, team := range teams {
				if *team.Name == repo.Name {
					if _, err = global.GithubClient.Teams.DeleteTeam(context.Background(), *team.ID); err != nil {
						fmt.Println("error deleting github team for deleted project,", err)
					}
					break
				}
			}
		}
	}
}

// Adds a user to the waitlist(waiting to receive their Github username for team)
func AddUserToTeamWaitlist(discordUser, owner, team string) {
	expectedUsernames[discordUser] = AddMemberData{
		Team:  team,
		Owner: owner,
	}
}

func AddUserToTeam(discordUser, githubName string) string {
	if teamAddData, ok := expectedUsernames[discordUser]; !ok {
		return "Wasn't expecting a github username from you! Did you use `!join` yet?"
	} else {
		nextPage := 0
		for moreTeams := true; moreTeams; moreTeams = nextPage != 0 {
			opt := github.ListOptions{
				Page:    nextPage,
				PerPage: 100,
			}
			teams, res, err := global.GithubClient.Teams.ListTeams(context.Background(), teamAddData.Owner, &opt)
			if err != nil {
				fmt.Println("error fetching Github team,", err)
				nextPage = 0
			} else {
				nextPage = res.NextPage
				for _, team := range teams {
					if *team.Name == teamAddData.Team {
						opts := github.TeamAddTeamMembershipOptions{Role: "member"}
						if _, _, err = global.GithubClient.Teams.AddTeamMembership(context.Background(), *team.ID, githubName, &opts); err != nil {
							fmt.Println("error adding user to github team,", err)
							return "Failed to add you to the github team `" + teamAddData.Team + "`. Try again later."
						}
						delete(expectedUsernames, discordUser)
						return "You've been added to " + teamAddData.Team
					}
				}
			}
		}
		return "Failed to find the github team for `" + teamAddData.Team + "`. Try again later."
	}
}

// Creates an issue
func CreateIssue(text, repository string) *string {
	issue := github.IssueRequest{
		Title: &text,
	}
	if _, _, err := global.GithubClient.Issues.Create(context.Background(), global.GithubOrgName, repository, &issue); err != nil {
		fmt.Println("error creating github issue,", err)
		msg := "Failed to create issue"
		return &msg
	}
	return nil
}
