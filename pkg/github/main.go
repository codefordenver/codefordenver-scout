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
	"io/ioutil"
	"net/http"
	"os"
)

type Repository struct {
	Name  string `json:"name"`
	Owner struct {
		Name string `json:"name"`
	} `json:"owner"`
}

type RepositoryEvent struct {
	Action          string     `json:"action"`
	EventRepository Repository `json:"repository"`
}

func Create() (*github.Client, error) {
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
	_, err := global.DiscordClient.GuildChannelCreateComplex(global.DiscordGuildId, channelCreateData)
	if err != nil {
		fmt.Println("error creating text channel for new project,", err)
		return
	}

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
