package discord

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/codefordenver/scout/global"
	"github.com/codefordenver/scout/pkg/gdrive"
	"strings"
)

type Permission int

const (
	PermissionAll Permission = iota
	PermissionMembers
	PermissionDM
	PermissionChannel
)

type Command struct {
	Keyword    string
	Handler    func(s *discordgo.Session, m *discordgo.MessageCreate)
	Permission Permission
}

type CommandHandler struct {
	Commands map[string]Command
}

var cmdHandler CommandHandler

// Dispatch a command, checking permissions first
func (c CommandHandler) DispatchCommand(key string, s *discordgo.Session, m *discordgo.MessageCreate) error {
	if command, exists := c.Commands[key]; exists {
		switch command.Permission {
		case PermissionMembers:
			member, err := s.GuildMember(m.GuildID, m.Author.ID)
			if err != nil {
				return err
			}
			if contains(member.Roles, global.MemberRole) {
				command.Handler(s, m)
			} else {
				if _, err = s.ChannelMessageSend(m.ChannelID, "You do not have permission to execute this command"); err != nil {
					fmt.Println("error sending permissions message,", err)
				}
			}
		case PermissionDM:
			channel, err := s.Channel(m.ChannelID)
			if err != nil {
				return err
			}
			if channel.Type == discordgo.ChannelTypeDM || channel.Type == discordgo.ChannelTypeGroupDM {
				command.Handler(s, m)
			} else {
				if _, err = s.ChannelMessageSend(m.ChannelID, "This command is only accessible from a DM"); err != nil {
					fmt.Println("error sending permissions message,", err)
				}
			}
		case PermissionChannel:
			channel, err := s.Channel(m.ChannelID)
			if err != nil {
				return err
			}
			if channel.Type == discordgo.ChannelTypeGuildText {
				command.Handler(s, m)
			} else {
				if _, err = s.ChannelMessageSend(m.ChannelID, "This command is only accessible from a server text channel"); err != nil {
					fmt.Println("error sending permissions message,", err)
				}
			}
		case PermissionAll:
			command.Handler(s, m)
		}
		return nil
	} else {
		if _, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Unrecognized command, %v", key)); err != nil {
			fmt.Println("error sending unrecognized command message,", err)
			return err
		}
		return nil
	}
}

// Register a command to the command handler.
func (c *CommandHandler) RegisterCommand(cmd Command) {
	c.Commands[cmd.Keyword] = cmd
}

// Create discord session and command handler
func Create() (*discordgo.Session, error) {
	dg, err := discordgo.New("Bot " + global.Token)
	if err != nil {
		return nil, err
	}

	cmdHandler.Commands = make(map[string]Command)

	onboardCommand := Command{Keyword: "onboard",
		Handler:    onboard,
		Permission: PermissionMembers,
	}
	cmdHandler.RegisterCommand(onboardCommand)
	onboardAllCommand := Command{Keyword: "onboardall",
		Handler:    onboardAll,
		Permission: PermissionMembers,
	}
	cmdHandler.RegisterCommand(onboardAllCommand)
	getAgendaCommand := Command{Keyword: "agenda",
		Handler:    getAgenda,
		Permission: PermissionAll,
	}
	cmdHandler.RegisterCommand(getAgendaCommand)
	fetchFileCommand := Command{Keyword: "fetch",
		Handler:    fetchFile,
		Permission: PermissionMembers,
	}
	cmdHandler.RegisterCommand(fetchFileCommand)

	fmt.Println(cmdHandler.Commands)

	return dg, nil
}

// When the bot connects to a server, record the number of uses on the onboarding invite, set role IDs.
func ConnectToGuild(s *discordgo.Session, r *discordgo.Ready) {
	for _, guild := range r.Guilds {
		invites, err := s.GuildInvites(guild.ID)
		if err != nil {
			fmt.Println("error fetching guild invites,", err)
			return
		}
		for _, invite := range invites {
			if invite.Code == global.OnboardingInviteCode {
				global.InviteCount[guild.ID] = invite.Uses
				break
			}
		}
	}
}

// When a user joins the server, give them the onboarding role if they joined using the onboarding invite.
func UserJoin(s *discordgo.Session, g *discordgo.GuildMemberAdd) {
	user := g.User
	guildID := g.GuildID
	invites, err := s.GuildInvites(guildID)
	if err != nil {
		fmt.Println("error fetching guild invites,", err)
		return
	}
	for _, invite := range invites {
		if invite.Code == global.OnboardingInviteCode {
			if global.InviteCount[guildID] != invite.Uses {
				global.InviteCount[guildID] = invite.Uses
				if err := s.GuildMemberRoleAdd(guildID, user.ID, global.OnboardingRole); err != nil {
					fmt.Println("error adding role,", err)
					return
				}
				return
			}
		}
	}
}

// When a user reacts to the welcome message to indicate that they have read and understand the rules, promote them to the new member role.
func UserReact(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	if m.MessageID == global.CodeOfConductMessageID {
		member, err := s.GuildMember(m.GuildID, m.UserID)
		if err != nil {
			fmt.Println("error fetching member who reacted,", err)
		}
		if contains(member.Roles, global.NewRole) || contains(member.Roles, global.OnboardingRole) || contains(member.Roles, global.MemberRole) {
			return
		} else if err = s.GuildMemberRoleAdd(m.GuildID, m.UserID, global.NewRole); err != nil {
			fmt.Println("error adding role,", err)
		}
	}
}

// When a message is sent, check if it is a command and handle it accordingly.
func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if strings.HasPrefix(m.Content, "!") {
		if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
			fmt.Println("error deleting command message,", err)
		}
		commandText := strings.TrimPrefix(m.Content, "!")
		commandName := strings.ToLower(strings.Split(commandText, " ")[0])
		err := cmdHandler.DispatchCommand(commandName, s, m)
		if err != nil {
			fmt.Println("error dispatching command", err)
		}
	}
}

// Onboard members with the onboarding role.
func onboard(s *discordgo.Session, m *discordgo.MessageCreate) {
	onboardGroup(s, m, global.OnboardingRole)
}

// Onboard members with the onboarding or new member role.
func onboardAll(s *discordgo.Session, m *discordgo.MessageCreate) {
	onboardGroup(s, m, global.OnboardingRole, global.NewRole)
}

// Give users with the onboarding and/or new member role the full member role
func onboardGroup(s *discordgo.Session, m *discordgo.MessageCreate, r ...string) {
	guildID := m.GuildID
	guild, err := s.Guild(guildID)
	if err != nil {
		fmt.Println("error fetching guild,", err)
		return
	}
	onboardedUsers := make([]*discordgo.User, 0)
	for _, member := range guild.Members {
		for _, role := range r {
			if contains(member.Roles, role) {
				if err = s.GuildMemberRoleRemove(guildID, member.User.ID, role); err != nil {
					fmt.Println("error removing role,", err)
					return
				}
				if err = s.GuildMemberRoleAdd(guildID, member.User.ID, global.MemberRole); err != nil {
					fmt.Println("error adding member role,", err)
					return
				}
				onboardedUsers = append(onboardedUsers, member.User)
				break
			}
		}
	}
	var confirmMessageContent string
	numberOnboarded := len(onboardedUsers)
	if numberOnboarded > 0 {
		confirmMessageContent = "Successfully onboarded "
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
	} else {
		confirmMessageContent = "No users to onboard"
	}
	if _, err = s.ChannelMessageSend(m.ChannelID, confirmMessageContent); err != nil {
		fmt.Println("error sending onboarding confirmation message,", err)
	}

}

// Return a link to the agenda for the next meeting
func getAgenda(s *discordgo.Session, m *discordgo.MessageCreate) {
	message := gdrive.FetchAgenda()
	if _, err := s.ChannelMessageSend(m.ChannelID, message); err != nil {
		fmt.Println("error sending agenda message,", err)
	}
}

// Return a link to requested file
func fetchFile(s *discordgo.Session, m *discordgo.MessageCreate) {
	fileName := strings.Fields(m.Content)[1]
	message := gdrive.FetchFile(fileName)
	if _, err := s.ChannelMessageSend(m.ChannelID, message); err != nil {
		fmt.Println("error sending file message,", err)
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
