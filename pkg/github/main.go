package github

import (
	"context"
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

func Create() (*github.Client, error) {
	colorGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

	tr := http.DefaultTransport
	itr, err := ghinstallation.NewKeyFromFile(tr, 31388, 1101679, global.PrivateKeyDir+"cfd-scout.2019-05-23.private-key.pem")
	if err != nil {
		fmt.Println("error creating github key", err)
		return nil, err
	}
	client := github.NewClient(&http.Client{Transport: itr})

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

// Setup tasks for a repository being created
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
	if _, err := global.DiscordClient.GuildChannelCreateComplex(global.DiscordGuildId, channelCreateData); err != nil {
		fmt.Println("error creating text channel for new project,", err)
	} else {
		// Create Discord webhook
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
			}

			_, _, err = global.GithubClient.Repositories.CreateHook(context.Background(), repo.Owner.Name, repo.Name, &githubHook)
			if err != nil {
				fmt.Println("error creating Github webhook,", err)
			}
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
		// Add repository to Github team
		options := github.TeamAddTeamRepoOptions{Permission: "push"}
		if _, err = global.GithubClient.Teams.AddTeamRepo(context.Background(), *team.ID, repo.Owner.Name, repo.Name, &options); err != nil {
			fmt.Println("error adding repository to team,", err)
			return
		}
	}
}

func handleRepositoryDelete(repo Repository) {
	if channels, err := global.DiscordClient.GuildChannels(global.DiscordGuildId); err != nil {
		fmt.Println("error fetching Discord guild,", err)
	} else {
		for _, channel := range channels {
			if channel.Name == repo.Name {
				if _, err = global.DiscordClient.ChannelDelete(channel.ID); err != nil {
					fmt.Println("error deleting text channel for deleted project,", err)
				}
				break
			}
		}
	}
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
