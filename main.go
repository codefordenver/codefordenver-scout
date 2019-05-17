package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// Global variables
var (
	token                string
	newRole              string
	onboardingRole       string
	memberRole           string
	onboardingInviteCode string
	inviteCount          = make(map[string]int, 0)
)

func init() {
	token = os.Getenv("SCOUT_TOKEN")
	newRole = os.Getenv("NEW_ROLE")
	onboardingRole = os.Getenv("ONBOARDING_ROLE")
	memberRole = os.Getenv("MEMBER_ROLE")
	onboardingInviteCode = os.Getenv("ONBOARDING_INVITE_CODE")
}

func main() {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session, ", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)
	dg.AddHandler(userJoin)
	dg.AddHandler(joinGuild)

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

func joinGuild(s *discordgo.Session, r *discordgo.Ready) {
	for _, guild := range r.Guilds {
		invites, err := s.GuildInvites(guild.ID)
		if err != nil {
			fmt.Println("error fetching guild invites, ", err)
			return
		}
		for _, invite := range invites {
			if invite.Code == onboardingInviteCode {
				inviteCount[guild.ID] = invite.Uses
			}
		}
	}
}

func userJoin(s *discordgo.Session, g *discordgo.GuildMemberAdd) {
	user := g.User
	guildID := g.GuildID
	invites, err := s.GuildInvites(guildID)
	if err != nil {
		fmt.Println("error fetching guild invites, ", err)
		return
	}
	for _, invite := range invites {
		if invite.Code == onboardingInviteCode {
			if inviteCount[guildID] == invite.Uses {
				if err := s.GuildMemberRoleAdd(guildID, user.ID, newRole); err != nil {
					fmt.Println("error adding role, ", err)
					return
				}
			} else {
				inviteCount[guildID] = invite.Uses
				if err := s.GuildMemberRoleAdd(guildID, user.ID, onboardingRole); err != nil {
					fmt.Println("error adding role, ", err)
					return
				}
			}
		}
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
		if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
			fmt.Println("error deleting command message, ", err)
		}
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
		onboard(s, m, newRole)
	case "onboard":
		onboard(s, m, onboardingRole)
	default:
		fmt.Println("unrecognized command, ", commandName)
	}
}

func onboard(s *discordgo.Session, m *discordgo.MessageCreate, r string) {
	guildID := m.GuildID
	guild, err := s.Guild(guildID)
	if err != nil {
		fmt.Println("error fetching guild, ", err)
		return
	}
	member, err := s.GuildMember(guildID, m.Author.ID)
	if err != nil {
		fmt.Println("error fetching message author, ", err)
		return
	}
	if member != nil {
		if contains(member.Roles, memberRole) {
			onboardedUsers := make([]*discordgo.User, 0)
			for _, member := range guild.Members {
				if contains(member.Roles, r) {
					if err = s.GuildMemberRoleRemove(guildID, member.User.ID, r); err != nil {
						fmt.Println("error removing role, ", err)
						return
					}
					if err = s.GuildMemberRoleAdd(guildID, member.User.ID, memberRole); err != nil {
						fmt.Println("error adding member role, ", err)
						return
					}
					onboardedUsers = append(onboardedUsers, member.User)
				}
			}
			var confirmMessage *discordgo.Message
			numberOnboarded := len(onboardedUsers)
			if numberOnboarded > 0 {
				confirmMessageContent := "Successfully onboarded "
				for i, user := range onboardedUsers {
					if numberOnboarded > 2 {
						if i == numberOnboarded - 1 {
							confirmMessageContent += "and <@!" + user.ID + ">"
						} else {
							confirmMessageContent += "<@!" + user.ID + ">, "
						}
					} else if numberOnboarded > 1 {
						if i == numberOnboarded - 1 {
							confirmMessageContent += " and <@!" + user.ID + ">"
						} else {
							confirmMessageContent += "<@!" + user.ID + ">"
						}
					} else {
						confirmMessageContent += "<@!" + user.ID + ">"
					}
				}
				confirmMessage = &discordgo.Message {
					Content:      confirmMessageContent,
					ChannelID:    m.ChannelID,
				}
			} else {
				confirmMessage = &discordgo.Message {
					Content:      "No users to onboard",
					ChannelID:    m.ChannelID,
				}
			}
			if _, err = s.ChannelMessageSend(m.ChannelID, confirmMessage.Content); err != nil {
				fmt.Println("error sending onboarding confirmation message, ", err)
			}
		} else {
			if _, err = s.ChannelMessageSend(m.ChannelID, "You do not have permission to execute this command"); err != nil {
				fmt.Println("error sending permissions message, ", err)
				return
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
