package github

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/bwmarrin/discordgo"
	"github.com/codefordenver/scout/global"
	"github.com/google/go-github/github"
	"io/ioutil"
	"net/http"
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

func Create() (*github.Client, error) {
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

func handleRepositoryCreate(repo Repository) {
	//Create Discord channel
	channelCreateData := discordgo.GuildChannelCreateData{
		Name:     repo.Name,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: global.ProjectCategoryId,
	}
	textChannel, err := global.DiscordClient.GuildChannelCreateComplex(global.DiscordGuildId, channelCreateData)
	if err != nil {
		fmt.Println("error creating text channel for new project,", err)
		return
	}

	discordWebhook, err := global.DiscordClient.WebhookCreate(textChannel.ID, "github-webhook", "")
	if err != nil {
		fmt.Println("error creating github webhook for text channel,", err)
	}

	discordWebhookURL := "https://discordapp.com/api/webhooks/" + discordWebhook.Token + "/github"
	githubHookName := "web"
	githubHookConfig := make(map[string]interface{})
	githubHookConfig["content_type"] = "json"
	githubHook := github.Hook{
		Name: &githubHookName,
		URL: &discordWebhookURL,
		Config: githubHookConfig,
	}

	_, _, err = global.GithubClient.Repositories.CreateHook(context.Background(), repo.Owner.Name, repo.Name, &githubHook)

	//Create Github team
	privacy := "closed"
	newTeam := github.NewTeam{
		Name:    repo.Name,
		Privacy: &privacy,
	}
	team, _, err := global.GithubClient.Teams.CreateTeam(context.Background(), repo.Owner.Name, newTeam)
	if err != nil {
		fmt.Println("error creating github team for new project,", err)
		return
	}

	options := github.TeamAddTeamRepoOptions{Permission: "push"}
	_, err = global.GithubClient.Teams.AddTeamRepo(context.Background(), *team.ID, repo.Owner.Name, repo.Name, &options)
	if err != nil {
		fmt.Println("error adding repository to team,", err)
		return
	}
}

func handleRepositoryDelete(repo Repository) {
	channels, err := global.DiscordClient.GuildChannels(global.DiscordGuildId)
	if err != nil {
		fmt.Println("error fetching Discord guild,", err)
	}
	for _, channel := range channels {
		if channel.Name == repo.Name {
			_, err = global.DiscordClient.ChannelDelete(channel.ID)
			if err != nil {
				fmt.Println("error deleting text channel for deleted project,", err)
			}
			return
		}
	}
}
