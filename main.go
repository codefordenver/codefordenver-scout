package main

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
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
	global.PrivateKeyDir = os.Getenv("SCOUT_PRIVATE_KEY_DIR")
}

func main() {
	var err error
	global.DriveClient, err = gdrive.Create()
	if err != nil {
		fmt.Println("error creating Google Drive session, ", err)
		return
	}

	global.GithubClient, err = github.Create()

	dg, err := discordgo.New("Bot " + global.Token)
	if err != nil {
		fmt.Println("error creating Discord session, ", err)
		return
	}

	dg.AddHandler(discord.MessageCreate)
	dg.AddHandler(discord.UserJoin)
	dg.AddHandler(discord.ConnectToGuild)
	dg.AddHandler(discord.UserReact)

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection, ", err)
		return

	}

	server := &http.Server{Addr: ":3000", Handler: http.HandlerFunc(github.HandleRepositoryEvent)}

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

	if err = dg.Close(); err != nil {
		fmt.Println("error closing Discord session, ", err)
	}

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println("error shutting down github webhook,", err)
	}
}
