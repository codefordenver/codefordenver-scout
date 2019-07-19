package main

import (
	"context"
	"fmt"
	"github.com/codefordenver/scout/global"
	"github.com/codefordenver/scout/pkg/discord"
	"github.com/codefordenver/scout/pkg/gdrive"
	"github.com/codefordenver/scout/pkg/github"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func init() {
	global.Token = os.Getenv("SCOUT_TOKEN")
	global.NewRole = os.Getenv("NEW_ROLE")
	global.OnboardingRole = os.Getenv("ONBOARDING_ROLE")
	global.MemberRole = os.Getenv("MEMBER_ROLE")
	global.OnboardingInviteCode = os.Getenv("ONBOARDING_INVITE_CODE")
	global.CodeOfConductMessageID = os.Getenv("CODE_OF_CONDUCT_MESSAGE_ID")
	global.AgendaFolderID = os.Getenv("AGENDA_FOLDER_ID")
	global.LocationString = os.Getenv("SCOUT_LOCATION_STRING")
	global.DiscordGuildId = os.Getenv("DISCORD_GUILD_ID")
	global.ProjectCategoryId = os.Getenv("PROJECT_CATEGORY_ID")
	global.IssueEmoji = os.Getenv("SCOUT_ISSUE_EMOJI")
	global.GithubOrgName = os.Getenv("SCOUT_ORG_NAME")
}

func main() {
	var err error
	global.DriveClient, err = gdrive.Create()
	if err != nil {
		fmt.Println("error creating Google Drive session, ", err)
		return
	}

	global.GithubClient, err = github.Create()

	global.DiscordClient, err = discord.Create()
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	global.DiscordClient.AddHandler(discord.MessageCreate)
	global.DiscordClient.AddHandler(discord.UserJoin)
	global.DiscordClient.AddHandler(discord.ConnectToGuild)
	global.DiscordClient.AddHandler(discord.UserReact)

	err = global.DiscordClient.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return

	}

	port := os.Getenv("PORT")
	var server *http.Server
	if port == "" {
		server = &http.Server{Addr: ":3000", Handler: http.HandlerFunc(github.HandleRepositoryEvent)}
	} else {
		server = &http.Server{Addr: ":" + port, Handler: http.HandlerFunc(github.HandleRepositoryEvent)}
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Println("error starting github webhook,", err)
		}
	}()

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	if err = global.DiscordClient.Close(); err != nil {
		fmt.Println("error closing Discord session, ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println("error shutting down github webhook,", err)
	} else {
		cancel()
	}
}
