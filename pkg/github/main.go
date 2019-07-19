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

var colorGenerator *rand.Rand

var brigades map[string]global.Brigade

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

	brigades = make(map[string]global.Brigade, 0)

	for _, brigade := range global.Brigades {
		brigades[brigade.GithubOrg] = brigade
	}

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
	championRole, err := global.DiscordClient.GuildRoleCreate(brigades[repo.Owner.Name].GuildID)
	if err != nil {
		fmt.Println("error creating role for new project,", err)
	} else {
		rolePermission := discordgo.PermissionCreateInstantInvite | discordgo.PermissionChangeNickname | discordgo.PermissionReadMessages | discordgo.PermissionSendMessages | discordgo.PermissionSendTTSMessages | discordgo.PermissionEmbedLinks | discordgo.PermissionAttachFiles | discordgo.PermissionReadMessageHistory | discordgo.PermissionMentionEveryone | discordgo.PermissionUseExternalEmojis | discordgo.PermissionAddReactions | discordgo.PermissionVoiceConnect | discordgo.PermissionVoiceSpeak
		if _, err = global.DiscordClient.GuildRoleEdit(brigades[repo.Owner.Name].GuildID, championRole.ID, repo.Name+"-champion", colorInt, false, rolePermission, true); err != nil {
			fmt.Println("error editing role for new project,", err)
		}
	}
	projectRole, err := global.DiscordClient.GuildRoleCreate(brigades[repo.Owner.Name].GuildID)
	if err != nil {
		fmt.Println("error creating role for new project,", err)
	} else {
		color = color.Lighten(.25)
		colorInt := int(color.Red)
		colorInt = (colorInt << 8) + int(color.Green)
		colorInt = (colorInt << 8) + int(color.Blue)
		rolePermission := discordgo.PermissionCreateInstantInvite | discordgo.PermissionChangeNickname | discordgo.PermissionReadMessages | discordgo.PermissionSendMessages | discordgo.PermissionSendTTSMessages | discordgo.PermissionEmbedLinks | discordgo.PermissionAttachFiles | discordgo.PermissionReadMessageHistory | discordgo.PermissionMentionEveryone | discordgo.PermissionUseExternalEmojis | discordgo.PermissionAddReactions | discordgo.PermissionVoiceConnect | discordgo.PermissionVoiceSpeak
		if _, err = global.DiscordClient.GuildRoleEdit(brigades[repo.Owner.Name].GuildID, projectRole.ID, repo.Name, colorInt, false, rolePermission, true); err != nil {
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
		ID:   brigades[repo.Owner.Name].EveryoneRole,
		Type: "role",
		Deny: discordgo.PermissionReadMessages,
	}
	channelCreateData := discordgo.GuildChannelCreateData{
		Name:     repo.Name,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: brigades[repo.Owner.Name].ProjectCategoryID,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			&projectChampionOverwrite,
			&projectOverwrite,
			&everyoneOverwrite,
		},
	}
	if textChannel, err := global.DiscordClient.GuildChannelCreateComplex(brigades[repo.Owner.Name].GuildID, channelCreateData); err != nil {
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
		ParentID: brigades[repo.Owner.Name].ProjectCategoryID,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			&projectChampionOverwrite,
			&projectOverwrite,
			&everyoneOverwrite,
		},
	}
	if textChannel, err := global.DiscordClient.GuildChannelCreateComplex(brigades[repo.Owner.Name].GuildID, githubChannelCreateData); err != nil {
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

	if channels, err := global.DiscordClient.GuildChannels(brigades[repo.Owner.Name].GuildID); err != nil {
		fmt.Println("error fetching guild text channels,", err)
	} else {
		sort.Slice(channels, func(i, j int) bool {
			return channels[i].Name < channels[j].Name
		})
		i := 0
		for _, channel := range channels {
			if channel.Type == discordgo.ChannelTypeGuildText && channel.ParentID == brigades[repo.Owner.Name].ProjectCategoryID {
				channel.Position = i
				i++
			}
		}
		if err := global.DiscordClient.GuildChannelsReorder(brigades[repo.Owner.Name].GuildID, channels); err != nil {
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
	if roles, err := global.DiscordClient.GuildRoles(brigades[repo.Owner.Name].GuildID); err != nil {
		fmt.Println("error fetching Discord roles")
	} else {
		for _, role := range roles {
			if role.Name == repo.Name || role.Name == repo.Name+"-champion" {
				if err = global.DiscordClient.GuildRoleDelete(brigades[repo.Owner.Name].GuildID, role.ID); err != nil {
					fmt.Println("error deleting role for deleted project,", err)
				}
			}
		}
	}

	// Delete Discord channel
	if channels, err := global.DiscordClient.GuildChannels(brigades[repo.Owner.Name].GuildID); err != nil {
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
