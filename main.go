package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/codefordenver/scout/global"
	"github.com/codefordenver/scout/pkg/discord"
	"github.com/codefordenver/scout/pkg/gdrive"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	global.Token = os.Getenv("SCOUT_TOKEN")
	global.NewRole = os.Getenv("NEW_ROLE")
	global.OnboardingRole = os.Getenv("ONBOARDING_ROLE")
	global.MemberRole = os.Getenv("MEMBER_ROLE")
	global.OnboardingInviteCode = os.Getenv("ONBOARDING_INVITE_CODE")
	global.CodeOfConductMessageID = os.Getenv("CODE_OF_CONDUCT_MESSAGE_ID")
	global.AgendaFolderID = os.Getenv("AGENDA_FOLDER_ID")
	global.LocationString = os.Getenv("SCOUTLOCATION_STRING")
}

func main() {
	var err error
	global.DriveClient, err = gdrive.Create()
	if err != nil {
		fmt.Println("error creating Google Drive session, ", err)
		return
	}

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
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	if err = dg.Close(); err != nil {
		fmt.Println("error closing Discord session, ", err)
	}
}
