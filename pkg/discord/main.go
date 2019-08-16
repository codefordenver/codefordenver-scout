package discord

import (
	"fmt"
	"github.com/brianloveswords/airtable"
	"github.com/bwmarrin/discordgo"
	"github.com/codefordenver/scout/global"
	"github.com/codefordenver/scout/pkg/gdrive"
	"github.com/codefordenver/scout/pkg/github"
	"os"
	"strconv"
	"strings"
)

type Permission int

const (
	PermissionAll Permission = iota
	PermissionAdmin
	PermissionMember
	PermissionDM
	PermissionChannel
)

type MessageData struct {
	ChannelID string
	GuildID   string
	Author    *discordgo.User
}

type CommandData struct {
	Session *discordgo.Session
	MessageData
	Args []string
}

type Command struct {
	Keyword    string
	Handler    func(CommandData)
	Permission Permission
	MinArgs    int
	MaxArgs    int
}

type CommandHandler struct {
	Commands map[string]Command
}

var cmdHandler CommandHandler

var brigades map[string]*global.Brigade

// Dispatch a command, checking permissions first
func (c CommandHandler) DispatchCommand(args []string, s *discordgo.Session, m *discordgo.MessageCreate) error {
	key := args[0]
	if len(args) > 1 {
		args = args[1:]
	}
	msgData := MessageData{
		ChannelID: m.ChannelID,
		GuildID:   m.GuildID,
		Author:    m.Author,
	}
	cmdData := CommandData{
		Session:     s,
		MessageData: msgData,
		Args:        args,
	}
	if command, exists := c.Commands[key]; exists {
		if (command.MinArgs != -1 && len(args) < command.MinArgs) || (command.MaxArgs != -1 && len(args) > command.MaxArgs) {
			if command.MinArgs == command.MaxArgs {
				if _, err := s.ChannelMessageSend(m.ChannelID, "Incorrect number of arguments provided to execute command. Required: "+strconv.Itoa(command.MinArgs)); err != nil {
					return err
				}
			} else {
				if _, err := s.ChannelMessageSend(m.ChannelID, "Incorrect number of arguments provided to execute command. Required: "+strconv.Itoa(command.MinArgs)+"-"+strconv.Itoa(command.MaxArgs)); err != nil {
					return err
				}
			}
			return nil
		}
		switch command.Permission {
		case PermissionAdmin:
			if channel, err := s.Channel(m.ChannelID); err != nil {
				return err
			} else {
				if channel.Type == discordgo.ChannelTypeGuildText {
					if perm, err := s.UserChannelPermissions(m.Author.ID, m.ChannelID); err != nil {
						return err
					} else if (perm & discordgo.PermissionAdministrator) == discordgo.PermissionAdministrator {
						command.Handler(cmdData)
					} else {
						if _, err = s.ChannelMessageSend(m.ChannelID, "You do not have permission to execute this command"); err != nil {
							return err
						}
					}
				} else {
					if _, err = s.ChannelMessageSend(m.ChannelID, "This command is only accessible from a server text channel"); err != nil {
						return err
					}
				}
			}
		case PermissionMember:
			if channel, err := s.Channel(m.ChannelID); err != nil {
				return err
			} else {
				if channel.Type == discordgo.ChannelTypeGuildText {
					member, err := s.GuildMember(m.GuildID, m.Author.ID)
					if err != nil {
						return err
					}
					if contains(member.Roles, brigades[m.GuildID].MemberRole) {
						command.Handler(cmdData)
					} else {
						if _, err = s.ChannelMessageSend(m.ChannelID, "You do not have permission to execute this command"); err != nil {
							return err
						}
					}
				} else {
					if _, err = s.ChannelMessageSend(m.ChannelID, "This command is only accessible from a server text channel"); err != nil {
						return err
					}
				}
			}
		case PermissionDM:
			channel, err := s.Channel(m.ChannelID)
			if err != nil {
				return err
			}
			if channel.Type == discordgo.ChannelTypeDM || channel.Type == discordgo.ChannelTypeGroupDM {
				command.Handler(cmdData)
			} else {
				if _, err = s.ChannelMessageSend(m.ChannelID, "This command is only accessible from a DM"); err != nil {
					return err
				}
			}
		case PermissionChannel:
			channel, err := s.Channel(m.ChannelID)
			if err != nil {
				return err
			}
			if channel.Type == discordgo.ChannelTypeGuildText {
				command.Handler(cmdData)
			} else {
				if _, err = s.ChannelMessageSend(m.ChannelID, "This command is only accessible from a server text channel"); err != nil {
					return err
				}
			}
		case PermissionAll:
			command.Handler(cmdData)
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
	dg, err := discordgo.New("Bot " + os.Getenv("SCOUT_TOKEN"))
	if err != nil {
		return nil, err
	}

	cmdHandler.Commands = make(map[string]Command)

	onboardCommand := Command{
		Keyword:    "onboard",
		Handler:    onboard,
		Permission: PermissionMember,
		MinArgs:    0,
		MaxArgs:    0,
	}
	cmdHandler.RegisterCommand(onboardCommand)
	onboardAllCommand := Command{
		Keyword:    "onboardall",
		Handler:    onboardAll,
		Permission: PermissionMember,
		MinArgs:    0,
		MaxArgs:    0,
	}
	cmdHandler.RegisterCommand(onboardAllCommand)
	getAgendaCommand := Command{
		Keyword:    "agenda",
		Handler:    getAgenda,
		Permission: PermissionAll,
		MinArgs:    0,
		MaxArgs:    0,
	}
	cmdHandler.RegisterCommand(getAgendaCommand)
	listCommand := Command{
		Keyword:    "list-projects",
		Handler:    listProjects,
		Permission: PermissionChannel,
		MinArgs:    0,
		MaxArgs:    0,
	}
	cmdHandler.RegisterCommand(listCommand)
	joinCommand := Command{
		Keyword:    "join",
		Handler:    joinProject,
		Permission: PermissionMember,
		MinArgs:    1,
		MaxArgs:    1,
	}
	cmdHandler.RegisterCommand(joinCommand)
	leaveCommand := Command{
		Keyword:    "leave",
		Handler:    leaveProject,
		Permission: PermissionMember,
		MinArgs:    1,
		MaxArgs:    1,
	}
	cmdHandler.RegisterCommand(leaveCommand)
	championsCommand := Command{
		Keyword:    "champion",
		Handler:    setChampions,
		Permission: PermissionAdmin,
		MinArgs:    2,
		MaxArgs:    -1,
	}
	cmdHandler.RegisterCommand(championsCommand)
	githubCommand := Command{
		Keyword:    "github",
		Handler:    sendGithubUsername,
		Permission: PermissionDM,
		MinArgs:    1,
		MaxArgs:    1,
	}
	cmdHandler.RegisterCommand(githubCommand)
	trackCommand := Command{
		Keyword:    "track",
		Handler:    trackFile,
		Permission: PermissionAdmin,
		MinArgs:    2,
		MaxArgs:    2,
	}
	cmdHandler.RegisterCommand(trackCommand)
	untrackCommand := Command{
		Keyword:    "untrack",
		Handler:    untrackFile,
		Permission: PermissionAdmin,
		MinArgs:    1,
		MaxArgs:    1,
	}
	cmdHandler.RegisterCommand(untrackCommand)
	fetchFileCommand := Command{
		Keyword:    "fetch",
		Handler:    fetchFileDispatch,
		Permission: PermissionMember,
		MinArgs:    1,
		MaxArgs:    1,
	}
	cmdHandler.RegisterCommand(fetchFileCommand)

	brigades = make(map[string]*global.Brigade, 0)

	for _, brigade := range global.Brigades {
		brigades[brigade.GuildID] = &brigade
	}

	return dg, nil
}

// When the bot connects to a server, record the number of uses on the onboarding invite, set role IDs.
func ConnectToGuild(s *discordgo.Session, r *discordgo.Ready) {
	for _, guild := range r.Guilds {
		invites, err := s.GuildInvites(guild.ID)
		if err != nil {
			fmt.Println("error fetching guild invites,", err)
		} else {
			for _, invite := range invites {
				if invite.Code == brigades[guild.ID].OnboardingInviteCode {
					brigades[guild.ID].InviteCount = invite.Uses
					break
				}
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
		if invite.Code == brigades[guildID].OnboardingInviteCode {
			if brigades[guildID].InviteCount != invite.Uses {
				brigades[guildID].InviteCount = invite.Uses
				if err := s.GuildMemberRoleAdd(guildID, user.ID, brigades[g.GuildID].OnboardingRole); err != nil {
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
	if m.MessageID == brigades[m.GuildID].CodeOfConductMessageID {
		member, err := s.GuildMember(m.GuildID, m.UserID)
		if err != nil {
			fmt.Println("error fetching member who reacted,", err)
		}
		if contains(member.Roles, brigades[m.GuildID].NewRole) || contains(member.Roles, brigades[m.GuildID].OnboardingRole) || contains(member.Roles, brigades[m.GuildID].MemberRole) {
			return
		} else if err = s.GuildMemberRoleAdd(m.GuildID, m.UserID, brigades[m.GuildID].NewRole); err != nil {
			fmt.Println("error adding role,", err)
		}
		return
	}
	if channel, err := s.Channel(m.ChannelID); err == nil && channel.Type == discordgo.ChannelTypeGuildText && channel.ParentID == brigades[m.GuildID].ProjectCategoryID && m.Emoji.Name == brigades[m.GuildID].IssueEmoji {
		if msg, err := s.ChannelMessage(m.ChannelID, m.MessageID); err != nil {
			fmt.Println("error fetching message to create issue,", err)
		} else {
			errorMessage := github.CreateIssue(msg.Content, channel.Name, brigades[m.GuildID])
			if errorMessage != nil {
				if _, err := s.ChannelMessageSend(m.ChannelID, *errorMessage); err != nil {
					fmt.Println("error sending issue status,", err)
				}
			}
		}
	} else if err != nil {
		fmt.Println("error getting channel to create issue,", err)
	}
}

// When a message is sent, check if it is a command and handle it accordingly.
func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if strings.HasPrefix(m.Content, "!") || containsUser(m.Mentions, s.State.User) {
		if channel, err := s.Channel(m.ChannelID); err != nil {
			fmt.Println("error fetching command channel,", err)
		} else if channel.Type == discordgo.ChannelTypeGuildText {
			if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
				fmt.Println("error deleting command message,", err)
			}
		}
		commandText := strings.TrimPrefix(m.Content, "!")
		if commandText == m.Content {
			commandText = strings.TrimPrefix(m.Content, fmt.Sprintf("<@%v>", s.State.User.ID))
		}
		args := strings.Fields(commandText)
		err := cmdHandler.DispatchCommand(args, s, m)
		if err != nil {
			fmt.Println("error dispatching command", err)
		}
	}
}

// Onboard members with the onboarding role.
func onboard(data CommandData) {
	onboardGroup(data.Session, data.MessageData, brigades[data.GuildID].OnboardingRole)
}

// Onboard members with the onboarding or new member role.
func onboardAll(data CommandData) {
	onboardGroup(data.Session, data.MessageData, brigades[data.GuildID].OnboardingRole, brigades[data.GuildID].NewRole)
}

// Give users with the onboarding and/or new member role the full member role
func onboardGroup(s *discordgo.Session, msgData MessageData, r ...string) {
	guildID := msgData.GuildID
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
				if err = s.GuildMemberRoleAdd(guildID, member.User.ID, brigades[msgData.GuildID].MemberRole); err != nil {
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
	if _, err = s.ChannelMessageSend(msgData.ChannelID, confirmMessageContent); err != nil {
		fmt.Println("error sending onboarding confirmation message,", err)
	}

}

// Return a link to the agenda for the next meeting
func getAgenda(data CommandData) {
	message := gdrive.FetchAgenda(brigades[data.GuildID])
	if _, err := data.Session.ChannelMessageSend(data.ChannelID, message); err != nil {
		fmt.Println("error sending agenda message,", err)
	}
}

// List available projects
func listProjects(data CommandData) {
	if channels, err := data.Session.GuildChannels(data.GuildID); err != nil {
		fmt.Println("error fetching guild channels,", err)
	} else {
		projectsMessage := "Current projects at `" + "codefordenver" + "`:"
		for _, channel := range channels {
			if channel.ParentID == brigades[data.GuildID].ProjectCategoryID {
				projectsMessage += "\n" + channel.Name
			}
		}
		if channel, err := data.Session.UserChannelCreate(data.Author.ID); err != nil {
			fmt.Println("error creating DM channel,", err)
		} else if _, err := data.Session.ChannelMessageSend(channel.ID, projectsMessage); err != nil {
			fmt.Println("error sending projects list,", err)
		}
	}
}

// Add user to project
func joinProject(data CommandData) {
	projectName := data.Args[0]
	if roles, err := data.Session.GuildRoles(data.GuildID); err != nil {
		fmt.Println("error fetching guild roles,", err)
	} else {
		for _, role := range roles {
			if strings.ToLower(role.Name) == strings.ToLower(projectName) {
				if err := data.Session.GuildMemberRoleAdd(data.GuildID, data.Author.ID, role.ID); err != nil {
					fmt.Println("error adding member role,", err)
				}
			}
		}
		if channel, err := data.Session.UserChannelCreate(data.Author.ID); err != nil {
			fmt.Println("error creating DM channel,", err)
		} else if _, err := data.Session.ChannelMessageSend(channel.ID, "Trying to add you to the github team for "+projectName+". Please respond with `!github your-github-username` to be added."); err != nil {
			fmt.Println("error sending channel message,", err)
		} else {
			github.AddUserToTeamWaitlist(data.Author.ID, brigades[data.GuildID].GithubOrg, projectName)
		}
	}
}

// Remove user from project
func leaveProject(data CommandData) {
	projectName := data.Args[0]
	if roles, err := data.Session.GuildRoles(data.GuildID); err != nil {
		fmt.Println("error fetching guild roles,", err)
	} else {
		for _, role := range roles {
			if strings.HasPrefix(strings.ToLower(role.Name), strings.ToLower(projectName)) {
				if err := data.Session.GuildMemberRoleRemove(data.MessageData.GuildID, data.MessageData.Author.ID, role.ID); err != nil {
					fmt.Println("error adding member role,", err)
				}
			}
		}
	}
}

// Set project champion(s)
func setChampions(data CommandData) {
	projectName := data.Args[0]
	users := data.Args[1:]
	for _, user := range users {
		userID := strings.TrimSuffix(strings.TrimPrefix(user, "<@"), ">")
		discordUser, err := data.Session.User(userID);
		if err != nil {
			if _, err := data.Session.ChannelMessageSend(data.ChannelID, "Failed to find user "+user); err != nil {
				fmt.Println("error sending failed to find user message,", err)
			}
			return
		}
		if roles, err := data.Session.GuildRoles(data.GuildID); err != nil {
			fmt.Println("error fetching guild roles,", err)
		} else {
			for _, role := range roles {
				if strings.ToLower(role.Name) == strings.ToLower(projectName)+"-champion" {
					if err := data.Session.GuildMemberRoleAdd(data.GuildID, discordUser.ID, role.ID); err != nil {
						fmt.Println("error adding member role,", err)
					}
				}
			}
			if channel, err := data.Session.UserChannelCreate(data.Author.ID); err != nil {
				fmt.Println("error creating DM channel,", err)
			} else if _, err := data.Session.ChannelMessageSend(channel.ID, "Trying to add you as a champion for "+projectName+". Please respond with `!github your-github-username` to be added."); err != nil {
				fmt.Println("error sending channel message,", err)
			} else {
				github.AddUserToChampionWaitlist(discordUser.ID, brigades[data.GuildID].GithubOrg, projectName)
			}
		}
	}
}

// Send github username to add user to team or set as admin
func sendGithubUsername(data CommandData) {
	githubName := data.Args[0]
	messages := github.DispatchUsername(data.Author.ID, githubName)
	for _, message := range messages {
		if channel, err := data.Session.UserChannelCreate(data.Author.ID); err != nil {
			fmt.Println("error creating DM channel,", err)
		} else if _, err := data.Session.ChannelMessageSend(channel.ID, message); err != nil {
			fmt.Println("error sending channel message,", err)
		}
	}
}

type FileRecord struct {
	airtable.Record
	Fields struct {
		Name string
		Link string
	}
}

// Add a file to Airtable tracking
func trackFile(data CommandData) {
	fileName := data.Args[0]
	link := data.Args[1]
	client := airtable.Client{
		APIKey: global.AirtableKey,
		BaseID: brigades[data.GuildID].AirtableBaseID,
	}
	file, err := fetchFile(data)
	if file != nil {
		if _, err := data.Session.ChannelMessageSend(data.ChannelID, "A file with the name **"+fileName+"** is already tracked: "+file.Fields.Link); err != nil {
			fmt.Println("error sending message to channel,", err)
		}
		return
	} else if err != nil {
		if _, err := data.Session.ChannelMessageSend(data.ChannelID, "Failed to check if a file with the name **"+fileName+"** is already tracked."); err != nil {
			fmt.Println("error sending message to channel,", err)
		}
		return
	}
	files := client.Table("Files")
	newRecord := FileRecord{
		Fields: struct {
			Name string
			Link string
		}{
			Name: fileName,
			Link: link,
		},
	}
	if err := files.Create(&newRecord); err != nil {
		fmt.Println("error creating new airtable record,", err)
		if _, err := data.Session.ChannelMessageSend(data.ChannelID, "Failed to track new file. Try again later."); err != nil {
			fmt.Println("error sending message to channel,", err)
		}
	} else {
		if _, err := data.Session.ChannelMessageSend(data.ChannelID, "File successfully tracked. Use `!fetch "+fileName+"` to retrieve it, or `!untrack "+fileName+"` to untrack it."); err != nil {
			fmt.Println("error sending message to channel,", err)
		}
	}
}

func untrackFile(data CommandData) {
	fileName := data.Args[0]
	client := airtable.Client{
		APIKey: global.AirtableKey,
		BaseID: brigades[data.GuildID].AirtableBaseID,
	}
	files := client.Table("Files")
	file, err := fetchFile(data)
	if file == nil {
		if _, err := data.Session.ChannelMessageSend(data.ChannelID, "No file with the name **"+fileName+"** is tracked."); err != nil {
			fmt.Println("error sending message to channel,", err)
		}
		return
	} else if err != nil {
		if _, err := data.Session.ChannelMessageSend(data.ChannelID, "Failed to check if a file with the name **"+fileName+"** is already tracked."); err != nil {
			fmt.Println("error sending message to channel,", err)
		}
		return
	}
	if err = files.Delete(file); err != nil {
		if _, err := data.Session.ChannelMessageSend(data.ChannelID, "Failed to untrack **"+fileName+"**. Try again later."); err != nil {
			fmt.Println("error sending message to channel,", err)
		}
	} else {
		if _, err := data.Session.ChannelMessageSend(data.ChannelID, "Successfully untracked **"+fileName+"**."); err != nil {
			fmt.Println("error sending message to channel,", err)
		}
	}
}

// Handle fetch command
func fetchFileDispatch(data CommandData) {
	file, err := fetchFile(data)
	var msg string
	if file != nil {
		msg = file.Fields.Link
	} else if err != nil {
		msg = "Failed to fetch file **" + data.Args[0] + "**. Try again later."
	} else {
		msg = "File **" + data.Args[0] + "** not found. Use `!track " + data.Args[0] + " [link]` to track it"
	}
	if _, err := data.Session.ChannelMessageSend(data.ChannelID, msg); err != nil {
		fmt.Println("error sending message to channel,", err)
	}
}

// Return a link to requested file
func fetchFile(data CommandData) (*FileRecord, error) {
	fileName := data.Args[0]
	client := airtable.Client{
		APIKey: global.AirtableKey,
		BaseID: brigades[data.GuildID].AirtableBaseID,
	}
	var results []FileRecord
	opts := airtable.Options{Filter: `{Name} = "` + fileName + `"`}
	files := client.Table("Files")
	var err error
	err = files.List(&results, &opts)
	if len(results) == 1 {
		return &results[0], nil
	} else if len(results) == 0 {
		return nil, nil
	}
	return nil, err
}

func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func containsUser(slice []*discordgo.User, value *discordgo.User) bool {
	for _, item := range slice {
		if item.ID == value.ID {
			return true
		}
	}
	return false
}
