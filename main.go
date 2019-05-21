package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// Global variables
var (
	token                  string
	newRole                string
	onboardingRole         string
	memberRole             string
	onboardingInviteCode   string
	codeOfConductMessageID string
	inviteCount            = make(map[string]int, 0)
)

func init() {
	token = os.Getenv("SCOUT_TOKEN")
	newRole = os.Getenv("NEW_ROLE")
	onboardingRole = os.Getenv("ONBOARDING_ROLE")
	memberRole = os.Getenv("MEMBER_ROLE")
	onboardingInviteCode = os.Getenv("ONBOARDING_INVITE_CODE")
	codeOfConductMessageID = os.Getenv("CODE_OF_CONDUCT_MESSAGE_ID")
}

func main() {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session, ", err)
		return
	}

	dg.AddHandler(messageCreate)
	dg.AddHandler(userJoin)
	dg.AddHandler(joinGuild)
	dg.AddHandler(userReact)

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
				break
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
			if inviteCount[guildID] != invite.Uses {
				inviteCount[guildID] = invite.Uses
				if err := s.GuildMemberRoleAdd(guildID, user.ID, onboardingRole); err != nil {
					fmt.Println("error adding role, ", err)
					return
				}
				return
			}
		}
	}
}

func userReact(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	if m.MessageID == codeOfConductMessageID {
		member, err := s.GuildMember(m.GuildID, m.UserID)
		if err != nil {
			fmt.Println("error fetching member who reactred, ", err)
		}
		if contains(member.Roles, newRole) || contains(member.Roles, onboardingRole) || contains(member.Roles, memberRole) {
			return
		} else if err = s.GuildMemberRoleAdd(m.GuildID, m.UserID, newRole); err != nil {
			fmt.Println("error adding role, ", err)
		}
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
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
	commandName := strings.ToLower(strings.Split(commandText, " ")[0])
	switch commandName {
	case "onboardall":
		onboard(s, m, onboardingRole, newRole)
	case "onboard":
		onboard(s, m, onboardingRole)
	default:
		fmt.Println("unrecognized command, ", commandName)
	}
}

func onboard(s *discordgo.Session, m *discordgo.MessageCreate, r ...string) {
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
				for _, role := range r {
					if contains(member.Roles, role) {
						if err = s.GuildMemberRoleRemove(guildID, member.User.ID, role); err != nil {
							fmt.Println("error removing role, ", err)
							return
						}
						if err = s.GuildMemberRoleAdd(guildID, member.User.ID, memberRole); err != nil {
							fmt.Println("error adding member role, ", err)
							return
						}
						onboardedUsers = append(onboardedUsers, member.User)
						break
					}
				}
			}
			var confirmMessage *discordgo.Message
			numberOnboarded := len(onboardedUsers)
			if numberOnboarded > 0 {
				confirmMessageContent := "Successfully onboarded "
				for i, user := range onboardedUsers {
					if numberOnboarded > 2 {
						if i == numberOnboarded-1 {
							confirmMessageContent += "and <@!" + user.ID + ">"
						} else {
							confirmMessageContent += "<@!" + user.ID + ">, "
						}
					} else if numberOnboarded > 1 {
						if i == numberOnboarded-1 {
							confirmMessageContent += " and <@!" + user.ID + ">"
						} else {
							confirmMessageContent += "<@!" + user.ID + ">"
						}
					} else {
						confirmMessageContent += "<@!" + user.ID + ">"
					}
				}
				confirmMessage = &discordgo.Message{
					Content:   confirmMessageContent,
					ChannelID: m.ChannelID,
				}
			} else {
				confirmMessage = &discordgo.Message{
					Content:   "No users to onboard",
					ChannelID: m.ChannelID,
				}
			}
			if _, err = s.ChannelMessageSend(m.ChannelID, confirmMessage.Content); err != nil {
				fmt.Println("error sending onboarding confirmation message, ", err)
			}
		} else {
			if _, err = s.ChannelMessageSend(m.ChannelID, "You do not have permission to execute this command"); err != nil {
				fmt.Println("error sending permissions message, ", err)
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
