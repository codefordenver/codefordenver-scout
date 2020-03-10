package discord

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/codefordenver/codefordenver-scout/models"
	"github.com/codefordenver/codefordenver-scout/pkg/gdrive"
	"github.com/codefordenver/codefordenver-scout/pkg/github"
	"github.com/codefordenver/codefordenver-scout/pkg/shared"
	"github.com/jinzhu/gorm"
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

type Command struct {
	Keyword    string
	Handler    func(shared.CommandData) []shared.FunctionResponse
	PermissionMap map[int]Permission
	MinArgs int
	MaxArgs int
}

type CommandHandler struct {
	Commands map[string]Command
}

var cmdHandler CommandHandler

func handleResponse(s *discordgo.Session, r []shared.FunctionResponse) {
	for _, response := range r {
		if _, err := s.Channel(response.ChannelID); err != nil {
			if _, err = s.UserChannelCreate(response.ChannelID); err != nil {
				fmt.Println("Failed to create DM channel to send response from command")
			}
		}
		if response.Success != "" {
			if _, err := s.ChannelMessageSend(response.ChannelID, response.Success); err != nil {
				fmt.Println("Failed to send response from command to channel")
			}
		}
		if response.Error != "" {
			if _, err := s.ChannelMessageSend(response.ChannelID, response.Error); err != nil {
				fmt.Println("Failed to send response from command to channel")
			}
		}
	}
}

// Dispatch a command, checking permissions first
func (c CommandHandler) DispatchCommand(args []string, s *discordgo.Session, m *discordgo.MessageCreate) error {
	var brigade models.Brigade
	// Check if guildID exists before fetching brigade for commands executed in DMs
	if len(m.GuildID) > 0 {
		err := db.Where("guild_id = ?", m.GuildID).First(&brigade).Error
		if err != nil {
			fmt.Println("error fetching brigade,", err)
			return err
		}
	}
	key := args[0]
	if len(args) > 1 {
		args = args[1:]
	} else {
		args = []string{}
	}
	msgData := shared.MessageData{
		ChannelID: m.ChannelID,
		Author:    m.Author,
	}
	cmdData := shared.CommandData{
		Session:     s,
		Brigade:     brigade,
		MessageData: msgData,
		Args:        args,
	}
	if command, exists := c.Commands[key]; exists {
		if len(args) < command.MinArgs || (command.MaxArgs != -1 && len(args) > command.MaxArgs) {
			if command.MinArgs == command.MaxArgs {
				if _, err := s.ChannelMessageSend(m.ChannelID, "Incorrect number of arguments provided to execute command. Required: "+argCountFmt(command.MinArgs)); err != nil {
					return err
				}
			} else {
				if _, err := s.ChannelMessageSend(m.ChannelID, "Incorrect number of arguments provided to execute command. Required: "+argCountFmt(command.MinArgs)+"-"+argCountFmt(command.MaxArgs)); err != nil {
					return err
				}
			}
			return nil
		}
		var response []shared.FunctionResponse
		// Check if Permission Map includes provided number of arguments
		permission, permissionExists := command.PermissionMap[len(args)]
		if !permissionExists {
			// Check if Permission Map includes variable argument option
			permission, permissionExists = command.PermissionMap[-1]
		}
		switch permission {
		case PermissionAdmin:
			if channel, err := s.Channel(m.ChannelID); err != nil {
				return err
			} else {
				if channel.Type == discordgo.ChannelTypeGuildText {
					if perm, err := s.UserChannelPermissions(m.Author.ID, m.ChannelID); err != nil {
						return err
					} else if (perm & discordgo.PermissionAdministrator) == discordgo.PermissionAdministrator {
						response = command.Handler(cmdData)
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
					if contains(member.Roles, brigade.MemberRole) {
						response = command.Handler(cmdData)
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
				response = command.Handler(cmdData)
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
			response = command.Handler(cmdData)
		}
		handleResponse(s, response)
		return nil
	} else {
		if _, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Unrecognized command, %v", key)); err != nil {
			fmt.Println("error sending channel message,", err)
			return err
		}
		return nil
	}
}

// Register a command to the command handler.
func (c *CommandHandler) RegisterCommand(cmd Command) {
	c.Commands[cmd.Keyword] = cmd
}

var db *gorm.DB

// Create discord session and command handler
func New(dbConnection *gorm.DB) (*discordgo.Session, error) {
	db = dbConnection

	dg, err := discordgo.New("Bot " + os.Getenv("SCOUT_TOKEN"))
	if err != nil {
		return nil, err
	}

	cmdHandler.Commands = make(map[string]Command)

	permissions := make(map[int]Permission)
	permissions[0] = PermissionMember
	onboardCommand := Command{
		Keyword:    "onboard",
		Handler:    onboard,
		PermissionMap: permissions,
		MinArgs: 0,
		MaxArgs: 0,
	}
	cmdHandler.RegisterCommand(onboardCommand)

	permissions = make(map[int]Permission)
	permissions[0] = PermissionMember
	onboardAllCommand := Command{
		Keyword:    "onboardall",
		Handler:    onboardAll,
		PermissionMap: permissions,
		MinArgs: 0,
		MaxArgs: 0,
	}
	cmdHandler.RegisterCommand(onboardAllCommand)

	permissions = make(map[int]Permission)
	permissions[0] = PermissionChannel
	getAgendaCommand := Command{
		Keyword:    "agenda",
		Handler:    getAgenda,
		PermissionMap: permissions,
		MinArgs: 0,
		MaxArgs: 0,
	}
	cmdHandler.RegisterCommand(getAgendaCommand)

	permissions = make(map[int]Permission)
	permissions[1] = PermissionMember
	joinCommand := Command{
		Keyword:    "join",
		Handler:    joinProject,
		PermissionMap: permissions,
		MinArgs: 1,
		MaxArgs: 1,
	}
	cmdHandler.RegisterCommand(joinCommand)

	permissions = make(map[int]Permission)
	permissions[1] = PermissionMember
	leaveCommand := Command{
		Keyword:    "leave",
		Handler:    leaveProject,
		PermissionMap: permissions,
		MinArgs: 1,
		MaxArgs: 1,
	}
	cmdHandler.RegisterCommand(leaveCommand)

	permissions = make(map[int]Permission)
	permissions[2] = PermissionAdmin
	trackCommand := Command{
		Keyword:    "track",
		Handler:    trackFile,
		PermissionMap: permissions,
		MinArgs: 2,
		MaxArgs: 2,
	}
	cmdHandler.RegisterCommand(trackCommand)

	permissions = make(map[int]Permission)
	permissions[1] = PermissionAdmin
	untrackCommand := Command{
		Keyword:    "untrack",
		Handler:    untrackFile,
		PermissionMap: permissions,
		MinArgs: 1,
		MaxArgs: 1,
	}
	cmdHandler.RegisterCommand(untrackCommand)

	permissions = make(map[int]Permission)
	permissions[1] = PermissionMember
	fetchFileCommand := Command{
		Keyword:    "fetch",
		Handler:    fetchFileDispatch,
		PermissionMap: permissions,
		MinArgs: 1,
		MaxArgs: 1,
	}
	cmdHandler.RegisterCommand(fetchFileCommand)

	permissions = make(map[int]Permission)
	permissions[1] = PermissionAdmin
	maintainProjectCommand := Command{
		Keyword:    "maintain",
		Handler:    maintainProject,
		PermissionMap: permissions,
		MinArgs: 1,
		MaxArgs: 1,
	}
	cmdHandler.RegisterCommand(maintainProjectCommand)

	permissions = make(map[int]Permission)
	permissions[2] = PermissionAdmin
	permissions[-1] = PermissionAdmin
	championsCommand := Command{
		Keyword:    "champion",
		Handler:    setChampions,
		PermissionMap: permissions,
		MinArgs: 2,
		MaxArgs: -1,
	}
	cmdHandler.RegisterCommand(championsCommand)

	permissions = make(map[int]Permission)
	permissions[1] = PermissionDM
	githubCommand := Command{
		Keyword:    "github",
		Handler:    sendGithubUsername,
		PermissionMap: permissions,
		MinArgs: 1,
		MaxArgs: 1,
	}
	cmdHandler.RegisterCommand(githubCommand)

	return dg, nil
}

// When the bot connects to a server, record the number of uses on the onboarding invite, set role IDs.
func ConnectToGuild(s *discordgo.Session, r *discordgo.Ready) {
	for _, guild := range r.Guilds {
		/* brigade := select brigades where guildID matches guild.ID */
		var brigade models.Brigade
		err := db.Where("guild_id = ?", guild.ID).First(&brigade).Error
		if err != nil {
			fmt.Println("error fetching brigade,", err)
			return
		}
		if invites, err := s.GuildInvites(guild.ID); err != nil {
			fmt.Println("error fetching guild invites,", err)
		} else {
			for _, invite := range invites {
				if invite.Code == brigade.OnboardingInviteCode {
					brigade.OnboardingInviteCount = invite.Uses
					/* Modify from brigades where guildID matches brigade.GuildID to increment onboarding_invite_count*/
					if err = db.Save(&brigade).Error; err != nil {
						fmt.Println("error updating brigade,", err)
					}
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
	/* brigade := select brigades matching guildID */
	var brigade models.Brigade
	err := db.Where("guild_id = ?", g.GuildID).First(&brigade).Error
	if err != nil {
		fmt.Println("error fetching brigade,", err)
		return
	}
	invites, err := s.GuildInvites(guildID)
	if err != nil {
		fmt.Println("error fetching guild invites,", err)
		return
	}
	for _, invite := range invites {
		if invite.Code == brigade.OnboardingInviteCode {
			if brigade.OnboardingInviteCount != invite.Uses {
				brigade.OnboardingInviteCount = invite.Uses
				/* Update invite count in db*/
				if err = db.Save(&brigade).Error; err != nil {
					fmt.Println("error updating brigade,", err)
				}
				if err := s.GuildMemberRoleAdd(guildID, user.ID, brigade.OnboardingRole); err != nil {
					fmt.Println("error adding guild role,", err)
					return
				}
				return
			}
		}
	}
}

// When a user reacts to the welcome message to indicate that they have read and understand the rules, promote them to the new member role.
func UserReact(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	/* brigade := select brigades where GuildID matches m.GuildID */
	var brigade models.Brigade
	err := db.Where("guild_id = ?", m.GuildID).First(&brigade).Error
	if err != nil {
		fmt.Println("error fetching brigade,", err)
		return
	}
	if m.MessageID == brigade.CodeOfConductMessageID {
		member, err := s.GuildMember(m.GuildID, m.UserID)
		if err != nil {
			fmt.Println("error fetching guild member,", err)
		}
		if contains(member.Roles, brigade.NewUserRole) || contains(member.Roles, brigade.OnboardingRole) || contains(member.Roles, brigade.MemberRole) {
			return
		} else if err = s.GuildMemberRoleAdd(m.GuildID, m.UserID, brigade.NewUserRole); err != nil {
			fmt.Println("error adding guild role,", err)
		}
		return
	}
	if channel, err := s.Channel(m.ChannelID); err == nil && channel.Type == discordgo.ChannelTypeGuildText && channel.ParentID == brigade.ActiveProjectCategoryID && m.Emoji.Name == brigade.IssueEmoji {
		if msg, err := s.ChannelMessage(m.ChannelID, m.MessageID); err != nil {
			fmt.Println("error fetching channel message,", err)
		} else {
			handleResponse(s, github.CreateIssue(msg.Content, brigade, *channel))
		}
	} else if err != nil {
		fmt.Println("error fetching guild channel,", err)
	}
}

// When a message is sent, check if it is a command and handle it accordingly.
func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if strings.HasPrefix(m.Content, "!") || containsUser(m.Mentions, s.State.User) {
		if channel, err := s.Channel(m.ChannelID); err != nil {
			fmt.Println("error fetching guild channel,", err)
		} else if channel.Type == discordgo.ChannelTypeGuildText {
			if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
				fmt.Println("error deleting channel message,", err)
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
func onboard(data shared.CommandData) []shared.FunctionResponse {
	return onboardGroup(data, data.Brigade.OnboardingRole)
}

// Onboard members with the onboarding or new member role.
func onboardAll(data shared.CommandData) []shared.FunctionResponse {
	return onboardGroup(data, data.Brigade.OnboardingRole, data.Brigade.NewUserRole)
}

// Give users with the onboarding and/or new member role the full member role
func onboardGroup(data shared.CommandData, r ...string) []shared.FunctionResponse {
	guildID := data.GuildID
	guild, err := data.Session.Guild(guildID)
	if err != nil {
		fmt.Println("error fetching guild,", err)
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Error:     "Failed to get Discord server for onboarding. Try again later.",
			},
		}
	}
	var errors string
	onboardedUsers := make([]*discordgo.User, 0)
	for _, member := range guild.Members {
		for _, role := range r {
			if contains(member.Roles, role) {
				if err = data.Session.GuildMemberRoleRemove(guildID, member.User.ID, role); err != nil {
					fmt.Println("error removing guild role,", err)
					errors += "\nFailed to remove **" + role + "** role from " + orEmpty(member.Nick, member.User.Username) + ". Have an administrator to remove it manually."
				}
				if err = data.Session.GuildMemberRoleAdd(guildID, member.User.ID, data.Brigade.MemberRole); err != nil {
					fmt.Println("error adding guild role,", err)
					errors += "\nFailed to add **" + role + "** role to " + orEmpty(member.Nick, member.User.Username) + ". Have an administrator to add it manually."
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
	return []shared.FunctionResponse{
		{
			ChannelID: data.ChannelID,
			Success:   confirmMessageContent,
			Error:     errors,
		},
	}
}

// Return a link to the agenda for the next meeting
func getAgenda(data shared.CommandData) []shared.FunctionResponse {
	return []shared.FunctionResponse{
		gdrive.FetchAgenda(data),
	}
}

// Add user to project
func joinProject(data shared.CommandData) []shared.FunctionResponse {
	projectName := data.Args[0]
	if roles, err := data.Session.GuildRoles(data.GuildID); err != nil {
		fmt.Println("error fetching guild roles,", err)
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Error:     "Failed to get Discord roles to add you to project. Try again later.",
			},
		}
	} else {
		for _, role := range roles {
			if strings.ToLower(role.Name) == strings.ToLower(projectName) {
				if err := data.Session.GuildMemberRoleAdd(data.GuildID, data.Author.ID, role.ID); err != nil {
					fmt.Println("error adding guild role,", err)
					return []shared.FunctionResponse{
						{
							ChannelID: data.ChannelID,
							Error:     "Failed to add **" + role.Name + "** role to " + data.Author.Username + ". Have an administrator add it manually.",
						},
					}
				}
			}
		}
		github.AddUserToTeamWaitlist(data.Author.ID, data.Brigade.GithubOrganization, projectName)
		return []shared.FunctionResponse{
			{
				ChannelID: data.Author.ID,
				Success:   "Trying to add you to the github team for " + projectName + ". Please respond with `!github your-github-username` to be added.",
			},
		}
	}
}

// Remove user from project
func leaveProject(data shared.CommandData) []shared.FunctionResponse {
	projectName := data.Args[0]
	if roles, err := data.Session.GuildRoles(data.GuildID); err != nil {
		fmt.Println("error fetching guild roles,", err)
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Error:     "Failed to get Discord roles to remove project role. Try again later.",
			},
		}
	} else {
		for _, role := range roles {
			if strings.HasPrefix(strings.ToLower(role.Name), strings.ToLower(projectName)) {
				if err := data.Session.GuildMemberRoleRemove(data.GuildID, data.MessageData.Author.ID, role.ID); err != nil {
					fmt.Println("error removing guild role,", err)
					return []shared.FunctionResponse{
						{
							ChannelID: data.ChannelID,
							Error:     "Failed to remove **" + role.Name + "** role from " + data.Author.Username + ". Have an administrator to remove it manually.",
						},
					}
				}
			}
		}
	}
	return []shared.FunctionResponse{
		{
			ChannelID: data.ChannelID,
			Success:   "You were successfully removed from " + projectName + ".",
		},
	}
}

// Set project champion(s)
func setChampions(data shared.CommandData) []shared.FunctionResponse {
	/* brigade := select brigades where guildID matches data.GuildID */
	projectName := data.Args[0]
	users := data.Args[1:]
	responses := make([]shared.FunctionResponse, 0)
	for _, user := range users {
		userID := strings.TrimSuffix(strings.TrimPrefix(user, "<@"), ">")
		discordUser, err := data.Session.User(userID)
		if err != nil {
			responses = append(responses, shared.FunctionResponse{
				ChannelID: data.ChannelID,
				Error:     "Failed to find user " + user + ". Try again later.",
			})
		} else {
			if roles, err := data.Session.GuildRoles(data.GuildID); err != nil {
				fmt.Println("error fetching guild roles,", err)
				responses = append(responses, shared.FunctionResponse{
					ChannelID: data.ChannelID,
					Error:     "Failed to get Discord roles to add champion role. Try again later.",
				})
			} else {
				for _, role := range roles {
					if strings.ToLower(role.Name) == strings.ToLower(projectName)+"-champion" {
						if err := data.Session.GuildMemberRoleAdd(data.GuildID, discordUser.ID, role.ID); err != nil {
							fmt.Println("error adding guild role,", err)
							responses = append(responses, shared.FunctionResponse{
								ChannelID: data.ChannelID,
								Error:     "Failed to get Discord roles to add champion role. Have an administrator to add it manually.",
							})
						}
					}
				}
			}
			github.AddUserToChampionWaitlist(discordUser.ID, data.Brigade.GithubOrganization, projectName)
		}
	}
	return responses
}

// Send github username to add user to team or set as admin
func sendGithubUsername(data shared.CommandData) []shared.FunctionResponse {
	githubName := data.Args[0]
	return []shared.FunctionResponse{
		github.DispatchUsername(data.MessageData, githubName),
	}
}

// Add a file
func trackFile(data shared.CommandData) []shared.FunctionResponse {
	fileName := strings.ToLower(data.Args[0])
	link := data.Args[1]
	file, err := fetchFile(data)
	if file != nil {
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Success:     "A file with the name **" + fileName + "** is already tracked: " + file.URL,
			},
		}
	} else if err != nil {
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Error:     "Failed to check if a file with the name **" + fileName + "** is already tracked. Try again later.",
			},
		}
	}
	/* New file = &File{...}*/
	file = &models.File{
		BrigadeID: data.Brigade.ID,
		Name:      fileName,
		URL:       link,
	}
	if err := db.Create(file).Error; err != nil {
		fmt.Println("error storing file record,", err)
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Error:     "Failed to track new file. Try again later.",
			},
		}
	} else {
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Success:   "File successfully tracked. Use `!fetch " + fileName + "` to retrieve it, or `!untrack " + fileName + "` to untrack it.",
			},
		}
	}
}

func untrackFile(data shared.CommandData) []shared.FunctionResponse {
	fileName := strings.ToLower(data.Args[0])
	file, err := fetchFile(data)
	if file == nil {
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Success:     "No file with the name **" + fileName + "** is tracked.",
			},
		}
	} else if err != nil {
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Error:     "Failed to check if a file with the name **" + fileName + "** is already tracked. Try again later.",
			},
		}
	}
	/* Delete from Files where FileName matches file.Name and brigade ID matches brigade with data.GuildID */
	if err = db.Delete(&file).Error; err != nil {
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Error:     "Failed to untrack **" + fileName + "**. Try again later.",
			},
		}
	} else {
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Success:   "Successfully untracked **" + fileName + "**.",
			},
		}
	}
}

// Handle fetch command
func fetchFileDispatch(data shared.CommandData) []shared.FunctionResponse {
	file, err := fetchFile(data)
	var msg string
	if file != nil {
		msg = file.URL
	} else if err != nil {
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Error:     "Failed to fetch file **" + data.Args[0] + "**. Try again later.",
			},
		}
	} else {
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Success:     "File **" + data.Args[0] + "** not found. Use `!track " + data.Args[0] + " [link]` to track it",
			},
		}
	}
	return []shared.FunctionResponse{
		{
			ChannelID: data.ChannelID,
			Success:   msg,
		},
	}
}

// Return a link to requested file
func fetchFile(data shared.CommandData) (*models.File, error) {
	fileName := strings.ToLower(data.Args[0])
	/* file := select all from Files where name matches fileName and brigade ID matches a brigade with data.GuildID*/
	var files []models.File
	err := db.Where("name = ? and brigade_id = ?", fileName, data.ID).Find(&files).Error
	if err != nil {
		fmt.Println("error fetching file,", err)
		return nil, err
	}
	if len(files) > 0 {
		file := files[0]
		return &file, nil
	} else {
		return nil, nil
	}
}

// Move project to maintenance
func maintainProject(data shared.CommandData) []shared.FunctionResponse {
	projectName := data.Args[0]
	guild, err := data.Session.Guild(data.GuildID)
	if err != nil {
		fmt.Println("error fetching guild,", err)
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Error:     "Failed to fetch Discord server for project. Try again later.",
			},
		}
	}
	var channel, githubChannel *discordgo.Channel
	for _, ch := range guild.Channels {
		if strings.ToLower(projectName) == ch.Name {
			channel = ch
		} else if strings.ToLower(projectName) == strings.TrimSuffix(ch.Name, "-github") {
			githubChannel = ch
		}
		if channel != nil && githubChannel != nil {
			break
		}
	}
	if githubChannel == nil || channel == nil {
		fmt.Println("error fetching guild channels,", err)
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Error:     "Failed to fetch Discord channels for project. Try again later.",
			},
		}
	}
	if _, err = data.Session.ChannelDelete(githubChannel.ID); err != nil {
		fmt.Println("error deleting guild channel,", err)
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Error:     "Failed to delete GitHub channel for **" + projectName + "**. Have an administrator do this manually.",
			},
		}
	}
	editData := discordgo.ChannelEdit{
		ParentID: data.Brigade.InactiveProjectCategoryID,
	}
	if _, err = data.Session.ChannelEditComplex(channel.ID, &editData); err != nil {
		fmt.Println("error editing guild channel,", err)
		return []shared.FunctionResponse{
			{
				ChannelID: data.ChannelID,
				Error:     "Failed to move discussion channel for **" + projectName + "**. Have an administrator do this manually.",
			},
		}
	}
	return []shared.FunctionResponse{
		{
			ChannelID: data.ChannelID,
			Success:   "Successfully moved  **" + projectName + "** to maintenance.",
		},
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

func orEmpty(str, defaultStr string) string {
	if str == "" {
		return defaultStr
	}
	return str
}

func argCountFmt(argCount int) string {
	if argCount == 1 {
		return "∞"
	} else {
		return strconv.Itoa(argCount)
	}
}

func containsUser(slice []*discordgo.User, value *discordgo.User) bool {
	for _, item := range slice {
		if item.ID == value.ID {
			return true
		}
	}
	return false
}
