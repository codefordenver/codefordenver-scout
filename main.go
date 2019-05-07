package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	Token string
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session, ", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)
	dg.AddHandler(userJoin)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection, ", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	if err = dg.Close(); err != nil {
		fmt.Println("error closing Discord session, ", err)
	}
}

func userJoin(s *discordgo.Session, g *discordgo.GuildMemberAdd) {
	user := *g.User
	err := s.GuildMemberRoleAdd(g.GuildID, user.ID, "575139365123129354")
	fmt.Println("New user: ", user)
	if err != nil {
		fmt.Println(err)
		return
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!") {
		handleCommand(s, m)
	}
}

func handleCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	commandText := strings.TrimPrefix(m.Content, "!")
	//commandArgs := strings.Split(commandText, " ")
	commandName := strings.ToLower(strings.Split(commandText, " ")[0])
	//if len(commandArgs) > 1 {
	//	args := strings.Split(commandText, " ")[1:]
	//}
	switch commandName {
	case "onboardall":
		onboardAll(s, m)
	default:
		fmt.Println("Unrecognized command: ", commandName)
	}
}

func onboardAll(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID
	guild, err := s.Guild(guildID)
	if err != nil {
		fmt.Println("error fetching guild, ", err)
	}
	member, err := s.GuildMember(guildID, m.Author.ID)
	if err != nil {
		fmt.Println("error fetching message author, ", err)
	}
	if member != nil {
		if contains(member.Roles, "575139388061777931") {
			if err != nil {
				fmt.Println("error fetching guild members, ", err)
			}
			for _, member := range guild.Members {
				if contains(member.Roles, "575139365123129354") {
					fmt.Println(member.Nick)
					if err = s.GuildMemberRoleRemove(guildID, member.User.ID, "575139365123129354"); err != nil {
						fmt.Println("Error removing New Member role, ", err)
					}
					if err = s.GuildMemberRoleAdd(guildID, member.User.ID, "575139388061777931"); err != nil {
						fmt.Println("Error adding CFD Member role, ", err)
					}
				}
			}
		}
	}
}

func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func indexOf(slice []string, value string) int {
	for i, item := range slice {
		if item == value {
			return i
		}
	}
	return -1
}
