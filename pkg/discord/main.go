package discord

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/codefordenver/codefordenver-scout/models"
	"github.com/codefordenver/codefordenver-scout/pkg/gdrive"
	"github.com/codefordenver/codefordenver-scout/pkg/github"
	"github.com/codefordenver/codefordenver-scout/pkg/shared"
	"github.com/jinzhu/gorm"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Command struct {
	Keyword                 string
	shared.ExecutionContext // Execution context specifies what information is needed by the command. Should be the MINIMUM information. For example, if at least a brigade is required but a project can be specified, use ContextBrigade.
	ContextHandler          func([]string) shared.ExecutionContext
	shared.Permission       // Permission should only used in conjunction with brigade or project execution contexts. Otherwise, use PermissionEveryone.
	PermissionHandler       func([]string) shared.Permission
	Handler                 func(shared.CommandData) shared.CommandResponse
	MinArgs                 int
	MaxArgs                 int
}

type CommandHandler struct {
	Commands map[string]Command
}

var cmdHandler CommandHandler

func handleResponse(s *discordgo.Session, r shared.CommandResponse) {
	if _, err := s.Channel(r.ChannelID); err != nil {
		if _, err = s.UserChannelCreate(r.ChannelID); err != nil {
			fmt.Println("Failed to create DM channel to send response from command")
		}
	}
	if r.Success != "" {
		if _, err := s.ChannelMessageSend(r.ChannelID, r.Success); err != nil {
			fmt.Println("Failed to send response from command to channel")
		}
	}
	if r.Error.ErrorString != "" {
		if r.Error.ErrorType == shared.ArgumentError {
			r.Error.ErrorString += " If you attempted to specify a brigade or project, ensure it was sent without spaces in the name."
		}
		if _, err := s.ChannelMessageSend(r.ChannelID, r.Error.ErrorString); err != nil {
			fmt.Println("Failed to send response from command to channel")
		}
	}
}

func getBrigade(name string) *models.Brigade {
	var brigade models.Brigade
	if err := db.Find(&brigade, "name = ?", name).Error; err != nil {
		fmt.Println(err)
		return nil
	}
	return &brigade
}

func getChannelBrigade(channel *discordgo.Channel) *models.Brigade {
	var brigade models.Brigade
	if err := db.Find(&brigade, "guild_id = ?", channel.GuildID).Error; err != nil {
		fmt.Println(err)
		return nil
	}
	return &brigade
}

func getProject(name string, brigadeID int) *models.Project {
	name = strings.ToLower(name)
	var project models.Project
	if err := db.Find(&project, "brigade_id = ? and name = ?", brigadeID, name).Error; err != nil {
		fmt.Println(err)
		return nil
	}
	return &project
}

func getChannelProject(channel *discordgo.Channel) *models.Project {
	var project models.Project
	if err := db.Find(&project, "discord_channel_id = ? or github_discord_channel_id = ?", channel.ID, channel.ID).Error; err != nil {
		fmt.Println(err)
		return nil
	}
	return &project
}

// Dispatch a command, checking permissions first
func (c CommandHandler) DispatchCommand(args []string, s *discordgo.Session, m *discordgo.MessageCreate) error {
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
		MessageData: msgData,
		Args:        args,
	}
	if command, exists := c.Commands[key]; exists {
		if channel, err := s.Channel(cmdData.ChannelID); err != nil {
			return err
		} else {
			// Populate command brigade & project based on either arguments provided or where the command was run from
			if len(cmdData.Args) >= command.MinArgs+2 {
				if brigade := getBrigade(cmdData.Args[len(cmdData.Args)-2]); brigade != nil { // If second to last argument is a brigade name
					cmdData.Brigade = brigade
					cmdData.BrigadeArg = cmdData.Args[len(cmdData.Args)-2]
					cmdData.Args = append(cmdData.Args[:len(cmdData.Args)-2], cmdData.Args[len(cmdData.Args)-1]) // Remove brigade from argument list for command execution
					var project *models.Project
					fmt.Println(brigade.ID)
					if project = getProject(cmdData.Args[len(cmdData.Args)-1], brigade.ID); project != nil { // If last argument is a project name
						cmdData.Project = project
						cmdData.ProjectArg = cmdData.Args[len(cmdData.Args)-1]
						cmdData.Args = cmdData.Args[:len(cmdData.Args)-1] // Remove project from argument list for command execution
					} else if project := getChannelProject(channel); project != nil { // If command was run from a project channel
						cmdData.Project = project
					}
				} else if brigade := getBrigade(cmdData.Args[len(cmdData.Args)-1]); brigade != nil { // If last argument is a brigade name
					cmdData.Brigade = brigade
					cmdData.BrigadeArg = cmdData.Args[len(cmdData.Args)-1]
					cmdData.Args = cmdData.Args[:len(cmdData.Args)-1] // Remove brigade from argument list for command execution
				} else if brigade := getChannelBrigade(channel); brigade != nil { // If command was run from a brigade channel, and no brigade argument was provided
					cmdData.Brigade = brigade
					var project *models.Project
					if project = getProject(cmdData.Args[len(cmdData.Args)-1], brigade.ID); project != nil { // If last argument is a project name
						cmdData.Project = project
						cmdData.ProjectArg = cmdData.Args[len(cmdData.Args)-1]
						cmdData.Args = cmdData.Args[:len(cmdData.Args)-1] // Remove project from argument list for command execution
					} else if project := getChannelProject(channel); project != nil { // If command was run from a project channel
						cmdData.Project = project
					}
				}
			} else if len(cmdData.Args) >= command.MinArgs+1 {
				if brigade := getBrigade(cmdData.Args[len(cmdData.Args)-1]); brigade != nil { // If last argument is a brigade name
					cmdData.Brigade = brigade
					cmdData.BrigadeArg = cmdData.Args[len(cmdData.Args)-1]
					cmdData.Args = cmdData.Args[:len(cmdData.Args)-1] // Remove brigade from argument list for command execution
				} else if brigade := getChannelBrigade(channel); brigade != nil { // If command was run from a brigade channel, and no brigade argument was provided
					cmdData.Brigade = brigade
					var project *models.Project
					if project = getProject(cmdData.Args[len(cmdData.Args)-1], brigade.ID); project != nil { // If last argument is a project name
						cmdData.Project = project
						cmdData.ProjectArg = cmdData.Args[len(cmdData.Args)-1]
						cmdData.Args = cmdData.Args[:len(cmdData.Args)-1] // Remove project from argument list for command execution
					} else if project := getChannelProject(channel); project != nil { // If command was run from a project channel
						cmdData.Project = project
					}
				}
			} else {                                                       // If no arguments were provided, find brigade & project from channel
				if brigade := getChannelBrigade(channel); brigade != nil { // If command was run from a brigade channel, and no brigade argument was provided
					cmdData.Brigade = brigade
				}
				if project := getChannelProject(channel); project != nil { // If command was run from a project channel, and no project argument was provided
					cmdData.Project = project
				}
			}

			var context shared.ExecutionContext
			if command.ContextHandler == nil {
				context = command.ExecutionContext
			} else {
				context = command.ContextHandler(cmdData.Args)
			}

			switch context { // Check command execution environment, send error if the execution environment is not valid
			case shared.ContextDM:
				if channel.Type != discordgo.ChannelTypeDM && channel.Type != discordgo.ChannelTypeGroupDM {
					if _, err := s.ChannelMessageSend(m.ChannelID, "`!"+key+"` must be executed in a DM with Scout. Please try again."); err != nil {
						return err
					}
					return nil
				}
			case shared.ContextBrigade:
				if cmdData.Brigade == nil {
					if _, err := s.ChannelMessageSend(m.ChannelID, "`!"+key+"` must be executed from a brigade server. Ensure you either ran the command from a brigade server or specified the brigade as an argument."); err != nil {
						return err
					}
					return nil
				}
			case shared.ContextProject:
				if cmdData.Project == nil {
					if _, err := s.ChannelMessageSend(m.ChannelID, "`!"+key+"` must be executed from a project channel. Ensure you either ran the command from a project server or specified the brigade and project as arguments."); err != nil {
						return err
					}
					return nil
				}
			}

			if len(cmdData.Args) < command.MinArgs || (command.MaxArgs != -1 && len(cmdData.Args) > command.MaxArgs) { // Check if # of arguments is adequate
				if command.MinArgs == command.MaxArgs {
					if _, err := s.ChannelMessageSend(m.ChannelID, "Incorrect number of arguments provided to execute command. Required: "+argCountFmt(command.MinArgs)+". If you attempted to specify a brigade or project, ensure it was sent without spaces in the name."); err != nil {
						return err
					}
				} else {
					if _, err := s.ChannelMessageSend(m.ChannelID, "Incorrect number of arguments provided to execute command. Required: "+argCountFmt(command.MinArgs)+"-"+argCountFmt(command.MaxArgs)+". If you attempted to specify a brigade or project, ensure it was sent without spaces in the name."); err != nil {
						return err
					}
				}
				return nil
			}

			var permission shared.Permission
			if command.PermissionHandler == nil {
				permission = command.Permission
			} else {
				permission = command.PermissionHandler(cmdData.Args)
			}

			var response shared.CommandResponse

			switch permission {
			case shared.PermissionAdmin:
				var channelID string
				if cmdData.Project != nil { // Check if project was specified
					channelID = cmdData.Project.DiscordChannelID
				} else if cmdData.Brigade != nil { // Otherwise check if brigade was specified
					channelID = cmdData.Brigade.ActiveProjectCategoryID
				} else {
					channelID = cmdData.ChannelID
				}
				if perm, err := s.UserChannelPermissions(cmdData.Author.ID, channelID); err != nil {
					return err
				} else if (perm & discordgo.PermissionAdministrator) == discordgo.PermissionAdministrator {
					response = command.Handler(cmdData)
				} else {
					if _, err = s.ChannelMessageSend(m.ChannelID, "You do not have permission to execute this command"); err != nil {
						return err
					}
				}
			case shared.PermissionMember:
				member, err := s.GuildMember(cmdData.Brigade.GuildID, cmdData.Author.ID)
				if err != nil {
					return err
				}
				if contains(member.Roles, cmdData.Brigade.MemberRole) {
					response = command.Handler(cmdData)
				} else {
					if _, err = s.ChannelMessageSend(m.ChannelID, "You do not have permission to execute this command"); err != nil {
						return err
					}
				}
			case shared.PermissionEveryone:
				response = command.Handler(cmdData)
			}
			handleResponse(s, response)
			return nil
		}
	} else {
		if _, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Unrecognized command, **%v**", key)); err != nil {
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

	onboardCommand := Command{
		Keyword:          "onboard",
		Handler:          onboard,
		ExecutionContext: shared.ContextBrigade,
		Permission:       shared.PermissionMember,
		MinArgs:          0,
		MaxArgs:          0,
	}
	cmdHandler.RegisterCommand(onboardCommand)

	onboardAllCommand := Command{
		Keyword:          "onboardall",
		Handler:          onboardAll,
		ExecutionContext: shared.ContextBrigade,
		Permission:       shared.PermissionMember,
		MinArgs:          0,
		MaxArgs:          0,
	}
	cmdHandler.RegisterCommand(onboardAllCommand)

	getAgendaCommand := Command{
		Keyword:          "agenda",
		Handler:          getAgenda,
		ExecutionContext: shared.ContextBrigade,
		Permission:       shared.PermissionMember,
		MinArgs:          0,
		MaxArgs:          0,
	}
	cmdHandler.RegisterCommand(getAgendaCommand)

	joinCommand := Command{
		Keyword:          "join",
		Handler:          joinProject,
		ExecutionContext: shared.ContextProject,
		Permission:       shared.PermissionMember,
		MinArgs:          1,
		MaxArgs:          1,
	}
	cmdHandler.RegisterCommand(joinCommand)

	leaveCommand := Command{
		Keyword:          "leave",
		Handler:          leaveProject,
		ExecutionContext: shared.ContextProject,
		Permission:       shared.PermissionMember,
		MinArgs:          1,
		MaxArgs:          1,
	}
	cmdHandler.RegisterCommand(leaveCommand)

	trackCommand := Command{
		Keyword:          "track",
		Handler:          trackFile,
		ExecutionContext: shared.ContextBrigade,
		Permission:       shared.PermissionAdmin,
		MinArgs:          2,
		MaxArgs:          2,
	}
	cmdHandler.RegisterCommand(trackCommand)

	untrackCommand := Command{
		Keyword:          "untrack",
		Handler:          untrackFile,
		ExecutionContext: shared.ContextBrigade,
		Permission:       shared.PermissionAdmin,
		MinArgs:          1,
		MaxArgs:          1,
	}
	cmdHandler.RegisterCommand(untrackCommand)

	fetchFileCommand := Command{
		Keyword:          "fetch",
		Handler:          fetchFileDispatch,
		ExecutionContext: shared.ContextBrigade,
		Permission:       shared.PermissionMember,
		MinArgs:          1,
		MaxArgs:          1,
	}
	cmdHandler.RegisterCommand(fetchFileCommand)

	checkInCommand := Command{
		Keyword:          "in",
		Handler:          checkIn,
		ExecutionContext: shared.ContextBrigade,
		Permission:       shared.PermissionMember,
		MinArgs:          0,
		MaxArgs:          -1,
	}
	cmdHandler.RegisterCommand(checkInCommand)

	checkOutCommand := Command{
		Keyword:           "out",
		Handler:           checkOut,
		ContextHandler:    checkOutContexts,
		PermissionHandler: checkOutPermissions,
		MinArgs:           0,
		MaxArgs:           -1,
	}
	cmdHandler.RegisterCommand(checkOutCommand)

	getTimeCommand := Command{
		Keyword:           "time",
		Handler:           getTime,
		ContextHandler:    getTimeContexts,
		PermissionHandler: getTimePermissions,
		MinArgs:           0,
		MaxArgs:           1,
	}
	cmdHandler.RegisterCommand(getTimeCommand)

	maintainProjectCommand := Command{
		Keyword:          "maintain",
		Handler:          maintainProject,
		ExecutionContext: shared.ContextProject,
		Permission:       shared.PermissionAdmin,
		MinArgs:          1,
		MaxArgs:          1,
	}
	cmdHandler.RegisterCommand(maintainProjectCommand)

	championsCommand := Command{
		Keyword:          "champion",
		Handler:          setChampions,
		ExecutionContext: shared.ContextProject,
		Permission:       shared.PermissionAdmin,
		MinArgs:          1,
		MaxArgs:          -1,
	}
	cmdHandler.RegisterCommand(championsCommand)

	githubCommand := Command{
		Keyword:          "github",
		Handler:          sendGithubUsername,
		ExecutionContext: shared.ContextAny,
		Permission:       shared.PermissionEveryone,
		MinArgs:          1,
		MaxArgs:          1,
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
func onboard(data shared.CommandData) shared.CommandResponse {
	return onboardGroup(data, data.Brigade.OnboardingRole)
}

// Onboard members with the onboarding or new member role.
func onboardAll(data shared.CommandData) shared.CommandResponse {
	return onboardGroup(data, data.Brigade.OnboardingRole, data.Brigade.NewUserRole)
}

// Give users with the onboarding and/or new member role the full member role
func onboardGroup(data shared.CommandData, r ...string) shared.CommandResponse {
	guild, err := data.Session.Guild(data.Brigade.GuildID)
	if err != nil {
		fmt.Println("error fetching guild,", err)
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "Failed to get Discord server for onboarding. Try again later.",
			},
		}
	}
	var onboardingErrors string
	onboardedUsers := make([]*discordgo.User, 0)
	for _, member := range guild.Members {
		for _, role := range r {
			if contains(member.Roles, role) {
				if err = data.Session.GuildMemberRoleRemove(data.Brigade.GuildID, member.User.ID, role); err != nil {
					fmt.Println("error removing guild role,", err)
					onboardingErrors += "\nFailed to remove **" + role + "** role from " + orEmpty(member.Nick, member.User.Username) + ". Have an administrator to remove it manually."
				}
				if err = data.Session.GuildMemberRoleAdd(data.Brigade.GuildID, member.User.ID, data.Brigade.MemberRole); err != nil {
					fmt.Println("error adding guild role,", err)
					onboardingErrors += "\nFailed to add **" + role + "** role to " + orEmpty(member.Nick, member.User.Username) + ". Have an administrator to add it manually."
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
	return shared.CommandResponse{
		ChannelID: data.ChannelID,
		Success:   confirmMessageContent,
		Error: shared.CommandError{
			ErrorType:   shared.ExecutionError,
			ErrorString: onboardingErrors,
		},
	}
}

// Return a link to the agenda for the next meeting
func getAgenda(data shared.CommandData) shared.CommandResponse {
	return gdrive.FetchAgenda(data)
}

// Add user to project
func joinProject(data shared.CommandData) shared.CommandResponse {
	if roles, err := data.Session.GuildRoles(data.Brigade.GuildID); err != nil {
		fmt.Println("error fetching guild roles,", err)
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "Failed to get Discord roles to add you to project. Try again later.",
			},
		}
	} else {
		for _, role := range roles {
			if strings.ToLower(role.Name) == strings.ToLower(data.Project.Name) {
				if err := data.Session.GuildMemberRoleAdd(data.Brigade.GuildID, data.Author.ID, role.ID); err != nil {
					fmt.Println("error adding guild role,", err)
					return shared.CommandResponse{
						ChannelID: data.ChannelID,
						Error: shared.CommandError{
							ErrorType:   shared.ExecutionError,
							ErrorString: "Failed to add **" + role.Name + "** role to " + data.Author.Username + ". Have an administrator add it manually.",
						},
					}
				}
			}
		}
		github.AddUserToTeamWaitlist(data.Author.ID, data.Brigade.GithubOrganization, data.Project.Name)
		return shared.CommandResponse{
			ChannelID: data.Author.ID,
			Success:   "Trying to add you to the github team for " + data.Project.Name + ". Please respond with `!github your-github-username` to be added.",
		}
	}
}

// Remove user from project
func leaveProject(data shared.CommandData) shared.CommandResponse {
	if roles, err := data.Session.GuildRoles(data.GuildID); err != nil {
		fmt.Println("error fetching guild roles,", err)
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "Failed to get Discord roles to remove project role. Try again later.",
			},
		}
	} else {
		for _, role := range roles {
			if strings.HasPrefix(strings.ToLower(role.Name), strings.ToLower(data.Project.Name)) {
				if err := data.Session.GuildMemberRoleRemove(data.GuildID, data.MessageData.Author.ID, role.ID); err != nil {
					fmt.Println("error removing guild role,", err)
					return shared.CommandResponse{
						ChannelID: data.ChannelID,
						Error: shared.CommandError{
							ErrorType:   shared.ExecutionError,
							ErrorString: "Failed to remove **" + role.Name + "** role from " + data.Author.Username + ". Have an administrator to remove it manually.",
						},
					}
				}
			}
		}
	}
	return shared.CommandResponse{
		ChannelID: data.ChannelID,
		Success:   "You were successfully removed from " + data.Project.Name + ".",
	}
}

// Set project champion(s)
func setChampions(data shared.CommandData) shared.CommandResponse {
	users := data.Args[0:]
	var addedChampions []string
	success := ""
	championErrors := make([]string, 0)
	for _, user := range users {
		userID := strings.TrimSuffix(strings.TrimPrefix(user, "<@"), ">")
		if discordUser, err := data.Session.User(userID); err != nil {
			championErrors = append(championErrors, "Failed to find user "+user+". Try again later.")
		} else {
			if roles, err := data.Session.GuildRoles(data.GuildID); err != nil {
				fmt.Println("error fetching guild roles,", err)
				championErrors = append(championErrors, "Failed to get Discord roles to add champion role. Try again later.")
			} else {
				addedChampions = make([]string, 0)
				for _, role := range roles {
					if strings.ToLower(role.Name) == strings.ToLower(data.Project.Name)+"-champion" {
						if err := data.Session.GuildMemberRoleAdd(data.GuildID, discordUser.ID, role.ID); err != nil {
							fmt.Println("error adding guild role,", err)
							championErrors = append(championErrors, "Failed to add champion role to <@!"+userID+">. Have an administrator to add it manually.")
						} else {
							addedChampions = append(addedChampions, userID)
						}
					}
				}
			}
			numberAdded := len(addedChampions)
			if numberAdded > 0 {
				success = "Successfully onboarded "
				for i, userID := range addedChampions {
					if numberAdded > 2 {
						if i == numberAdded-1 {
							success += "and <@!" + userID + ">"
						} else {
							success += "<@!" + userID + ">, "
						}
					} else if numberAdded > 1 {
						if i == numberAdded-1 {
							success += " and <@!" + userID + ">"
						} else {
							success += "<@!" + userID + ">"
						}
					} else {
						success += "<@!" + userID + ">"
					}
				}
			}
			github.AddUserToChampionWaitlist(discordUser.ID, data.Brigade.GithubOrganization, data.Project.Name)
		}
	}
	return shared.CommandResponse{
		ChannelID: data.ChannelID,
		Success:   success,
		Error: shared.CommandError{
			ErrorType:   shared.ExecutionError,
			ErrorString: strings.Join(championErrors, "\n"),
		},
	}
}

// Send github username to add user to team or set as admin
func sendGithubUsername(data shared.CommandData) shared.CommandResponse {
	githubName := data.Args[0]
	return github.DispatchUsername(data.MessageData, githubName)
}

// Add a file
func trackFile(data shared.CommandData) shared.CommandResponse {
	fileName := strings.ToLower(data.Args[0])
	link := data.Args[1]
	file, err := fetchFile(data)
	if file != nil {
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Success:   "A file with the name **" + fileName + "** is already tracked: " + file.URL,
		}
	} else if err != nil {
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "Failed to check if a file with the name **" + fileName + "** is already tracked. Try again later.",
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
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "Failed to track new file. Try again later.",
			},
		}
	} else {
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Success:   "File successfully tracked. Use `!fetch " + fileName + "` to retrieve it, or `!untrack " + fileName + "` to untrack it.",
		}
	}
}

func untrackFile(data shared.CommandData) shared.CommandResponse {
	fileName := strings.ToLower(data.Args[0])
	file, err := fetchFile(data)
	if file == nil {
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Success:   "No file with the name **" + fileName + "** is tracked.",
		}
	} else if err != nil {
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "Failed to check if a file with the name **" + fileName + "** is already tracked. Try again later.",
			},
		}
	}
	/* Delete from Files where FileName matches file.Name and brigade ID matches brigade with data.GuildID */
	if err = db.Delete(&file).Error; err != nil {
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "Failed to untrack **" + fileName + "**. Try again later.",
			},
		}
	} else {
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Success:   "Successfully untracked **" + fileName + "**.",
		}
	}
}

// Handle fetch command
func fetchFileDispatch(data shared.CommandData) shared.CommandResponse {
	file, err := fetchFile(data)
	var msg string
	if file != nil {
		msg = file.URL
	} else if err != nil {
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "Failed to fetch file **" + data.Args[0] + "**. Try again later.",
			},
		}
	} else {
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Success:   "File **" + data.Args[0] + "** not found. Use `!track " + data.Args[0] + " [link]` to track it",
		}
	}
	return shared.CommandResponse{
		ChannelID: data.ChannelID,
		Success:   msg,
	}
}

// Return a link to requested file
func fetchFile(data shared.CommandData) (*models.File, error) {
	fileName := strings.ToLower(data.Args[0])
	/* file := select all from Files where name matches fileName and brigade ID matches a brigade with data.GuildID*/
	var files []models.File
	if err := db.Where("name = ? and brigade_id = ?", fileName, data.Brigade.ID).Find(&files).Error; err != nil {
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

func isTime(timeStr string) bool {
	if matches, err := regexp.Match(`((\d{1,2}|\w{3})([/ ])\d{1,2})? ?\d{1,2}(:\d{1,2}){1,2}(AM|PM)?`, []byte(timeStr)); err != nil {
		return false
	} else {
		return matches
	}
}

// Check common time formats
func parseTime(timeStr string, location *time.Location) (time.Time, error) {
	if isTime(timeStr) {
		now := time.Now()
		formats := []string{
			// Time only
			"3:04PM",
			"3:04:05PM",
			"15:04",
			"15:04:05",
			// Date & time
			"Jan 2 3:04PM",
			"Jan 2 3:04:05PM",
			"Jan 2 15:04",
			"Jan 2 15:04:05",
			"1/2 3:04PM",
			"1/2 3:04:05PM",
			"1/2 15:04",
			"1/2 15:04:05",
		}
		var outTime time.Time
		var err error
		var t time.Time
		for i, format := range formats {
			if t, err = time.Parse(format, timeStr); err == nil {
				if i <= 3 {
					outTime = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), location)
				} else {
					outTime = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), location)
				}
				return outTime, nil
			}
		}
		return time.Time{}, err
	} else {
		return time.Time{}, errors.New("failed to parse provided time")
	}
}

// Format duration with spaces
func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%1dh %1dm %1ds", h, m, s)
}

func checkIn(data shared.CommandData) shared.CommandResponse {
	var tz *time.Location
	var err error
	if len(data.Args) == 0 {
		return startSession(data, time.Now())
	} else if tz, err = time.LoadLocation(data.Brigade.TimezoneString); err != nil {
		tz = time.Local
	}
	if inTime, err := parseTime(strings.Join(data.Args[0:], " "), tz); err != nil {
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ArgumentError,
				ErrorString: "Failed to parse provided time. Try formatting your time as `3:04PM` or `Jan 2 3:04PM` if you're starting a session from another day.",
			},
		}
	} else {
		return startSession(data, inTime)
	}
}

func startSession(data shared.CommandData, inTime time.Time) shared.CommandResponse {
	session := models.VolunteerSession{
		BrigadeID:     data.Brigade.ID,
		DiscordUserID: data.Author.ID,
		StartTime:     inTime,
	}
	if data.Project != nil {
		session.ProjectID = sql.NullInt64{
			Int64: int64(data.Project.ID),
			Valid: true,
		}
	} else {
		session.ProjectID = sql.NullInt64{
			Int64: 0,
			Valid: false,
		}
	}
	var count int
	if db.Model(&models.VolunteerSession{}).Where("discord_user_id = ? and duration is null", data.Author.ID).Count(&count); count > 0 {
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "You already have an active volunteering session. Please end it before starting a new one.",
			},
		}
	}
	if inTime.After(time.Now()) {
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "You can't start a volunteering session in the future. Try an earlier time.",
			},
		}
	}
	if err := db.Create(&session).Error; err != nil {
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "Failed to start volunteering session. Try again later.",
			},
		}
	}
	return shared.CommandResponse{
		ChannelID: data.ChannelID,
		Success:   "Started a volunteering session for <@!" + data.Author.ID + ">. Use `!out` to end your session.",
	}
}

func checkOutContexts(args []string) shared.ExecutionContext {
	if len(args) > 0 && !isTime(strings.Join(args[0:], " ")) {
		return shared.ContextBrigade
	} else {
		return shared.ContextAny
	}
}

func checkOutPermissions(args []string) shared.Permission {
	if len(args) > 0 && !isTime(strings.Join(args[0:], " ")) {
		return shared.PermissionAdmin
	} else {
		return shared.PermissionEveryone
	}
}

func checkOut(data shared.CommandData) shared.CommandResponse {
	var tz *time.Location
	var err error
	if len(data.Args) == 0 {
		return endSession(data, time.Now())
	} else if tz, err = time.LoadLocation(data.Brigade.TimezoneString); err != nil {
		tz = time.Local
	}
	if outTime, err := parseTime(strings.Join(data.Args[0:], " "), tz); err == nil {
		return endSession(data, outTime)
	} else if data.Args[0] == "all" {
		var sessions []models.VolunteerSession
		if err := db.Find(&sessions, "brigade_id = ? and duration is null", data.Brigade.ID).Error; err != nil {
			return shared.CommandResponse{
				ChannelID: data.ChannelID,
				Error: shared.CommandError{
					ErrorType:   shared.ExecutionError,
					ErrorString: "Failed to get active volunteering sessions for this brigade. Try again later.",
				},
			}
		}
		var outTime time.Time
		if len(data.Args) > 1 {
			var timeErr error
			if outTime, timeErr = parseTime(strings.Join(data.Args[1:], " "), tz); timeErr != nil {
				return shared.CommandResponse{
					ChannelID: data.ChannelID,
					Error: shared.CommandError{
						ErrorType:   shared.ArgumentError,
						ErrorString: "Failed to parse provided time. Try formatting your time as `3:04PM` or `Jan 2 3:04PM` if you're starting a session from another day.",
					},
				}
			}
		} else {
			outTime = time.Now()
		}
		successes := make([]string, 0)
		outErrors := make([]string, 0)
		for _, session := range sessions {
			if discordUser, err := data.Session.User(session.DiscordUserID); err != nil {
				outErrors = append(outErrors, "Failed to find user <@!"+session.DiscordUserID+">. Try again later.")
			} else {
				response := endSession(shared.CommandData{
					MessageData: shared.MessageData{
						Author: discordUser,
					},
				}, outTime)
				if response.Success != "" {
					successes = append(successes, response.Success)
				} else {
					outErrors = append(outErrors, response.Error.ErrorString)
				}
			}
		}
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Success:   strings.Join(successes, "\n"),
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: strings.Join(outErrors, "\n"),
			},
		}
	} else {
		var users []*discordgo.User
		successes := make([]string, 0)
		outErrors := make([]string, 0)
		for i := 0; i < len(data.Args); i++ { // Loop through arguments until we hit one that isn't a user
			userID := strings.TrimSuffix(strings.TrimPrefix(data.Args[i], "<@"), ">")
			if userID != data.Args[i] { // Was a mention, therefore argument is a userID
				if discordUser, err := data.Session.User(userID); err != nil {
					outErrors = append(outErrors, "Failed to find user "+data.Args[i]+". Try again later.")
				} else {
					users = append(users, discordUser)
				}
			} else if outTime, timeErr := parseTime(strings.Join(data.Args[i:], " "), tz); timeErr != nil && i != len(data.Args)-1 {
				return shared.CommandResponse{
					ChannelID: data.ChannelID,
					Error: shared.CommandError{
						ErrorType:   shared.ArgumentError,
						ErrorString: "Failed to parse provided time. Try formatting your time as `3:04PM` or `Jan 2 3:04PM` if you're starting a session from another day.",
					},
				}
			} else {
				if timeErr != nil { // If no out time was provided, set out time to now
					outTime = time.Now()
				}
				for _, user := range users {
					response := endSession(shared.CommandData{
						MessageData: shared.MessageData{
							Author: user,
						},
					}, outTime)
					if response.Success != "" {
						successes = append(successes, response.Success)
					} else {
						outErrors = append(outErrors, response.Error.ErrorString)
					}
				}
			}
		}
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Success:   strings.Join(successes, "\n"),
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: strings.Join(outErrors, "\n"),
			},
		}
	}
}

func endSession(data shared.CommandData, outTime time.Time) shared.CommandResponse {
	var session models.VolunteerSession
	if err := db.First(&session, "discord_user_id = ? and duration is null", data.Author.ID).Error; err != nil {
		return shared.CommandResponse{
			Error: shared.CommandError{
				ErrorString: "<@!" + data.Author.ID + "> doesn't seem to have an active volunteering session to end.",
			},
		}
	}
	if outTime.Before(session.StartTime) {
		return shared.CommandResponse{
			Error: shared.CommandError{
				ErrorString: "<@! + " + data.Author.ID + ">'s volunteering session can't be ended before it started. Try a later time.",
			},
		}
	}
	if outTime.After(time.Now()) {
		return shared.CommandResponse{
			Error: shared.CommandError{
				ErrorString: "<@! + " + data.Author.ID + ">'s volunteering session can't be ended in the future. Try an earlier time.",
			},
		}
	}
	if err := db.Model(&session).Update("duration", outTime.Round(time.Second).Sub(session.StartTime.Round(time.Second))).Error; err != nil {
		return shared.CommandResponse{
			Error: shared.CommandError{
				ErrorString: "Failed to end <@! + " + data.Author.ID + ">'s active volunteering session. Try again later.",
			},
		}
	}
	return shared.CommandResponse{
		ChannelID: data.ChannelID,
		Success:   "Ended <@!" + data.Author.ID + ">'s volunteering session, which lasted **" + fmtDuration(time.Duration(session.Duration.Int64)) + "**",
	}
}

func getTimePermissions(args []string) shared.Permission {
	if len(args) > 0 {
		return shared.PermissionAdmin
	}
	return shared.PermissionEveryone
}

func getTimeContexts(args []string) shared.ExecutionContext {
	if len(args) > 0 {
		return shared.ContextBrigade
	}
	return shared.ContextAny
}

func getTime(data shared.CommandData) shared.CommandResponse {
	var initialSessionSet *gorm.DB
	var dataFor string
	if data.ProjectArg != "" {
		if initialSessionSet = db.Model(models.VolunteerSession{}).Where("brigade_id = ? and project_id = ?", data.Brigade.ID, data.Project.ID); initialSessionSet.Error != nil {
			return shared.CommandResponse{
				ChannelID: data.ChannelID,
				Error: shared.CommandError{
					ErrorType:   shared.ExecutionError,
					ErrorString: "Failed to fetch volunteering sessions",
				},
			}
		}
		dataFor = "**" + data.Brigade.DisplayName + "**'s **" + data.ProjectArg + "** project"
	} else if data.BrigadeArg != "" {
		if initialSessionSet = db.Model(models.VolunteerSession{}).Where("brigade_id = ?", data.Brigade.ID); initialSessionSet.Error != nil {
			return shared.CommandResponse{
				ChannelID: data.ChannelID,
				Error: shared.CommandError{
					ErrorType:   shared.ExecutionError,
					ErrorString: "Failed to fetch volunteering sessions",
				},
			}
		}
		dataFor = "**" + data.Brigade.Name + "**"
	} else {
		dataFor = "<@!" + data.Author.ID + ">"
		if initialSessionSet = db.Model(models.VolunteerSession{}).Where("discord_user_id = ?", data.Author.ID); initialSessionSet.Error != nil {
			return shared.CommandResponse{
				ChannelID: data.ChannelID,
				Error: shared.CommandError{
					ErrorType:   shared.ExecutionError,
					ErrorString: "Failed to fetch volunteering sessions",
				},
			}
		}
	}
	if len(data.Args) == 1 {
		if data.Args[0] == "projects" {
			if rows, err := initialSessionSet.Select("project_id, sum(duration)").Group("project_id").Rows(); err != nil {
				return shared.CommandResponse{
					ChannelID: data.ChannelID,
					Error: shared.CommandError{
						ErrorType:   shared.ExecutionError,
						ErrorString: "Failed to fetch brigade's volunteer sessions by project. Try again later.",
					},
				}
			} else {
				successes := make([]string, 0)
				successes = append(successes, "Total time for " + dataFor + " by project:")
				groupingErrors := make([]string, 0)
				var projectID sql.NullInt64
				var totalTimeNano int
				for rows.Next() {
					if err := rows.Scan(&projectID, &totalTimeNano); err != nil {
						return shared.CommandResponse{
							ChannelID: data.ChannelID,
							Error: shared.CommandError{
								ErrorType:   shared.ExecutionError,
								ErrorString: "Failed to group volunteer sessions by project. Try again later.",
							},
						}
					}
					if projectID.Valid {
						var project models.Project
						if err := db.Find(&project, "id = ?", projectID.Int64).Error; err != nil {
							groupingErrors = append(groupingErrors, fmt.Sprintf("Failed to find project with ID **%v**.", projectID.Int64))
						} else {
							successes = append(successes, fmt.Sprintf("%v: **%v**", project.Name, fmtDuration(time.Duration(totalTimeNano))))
						}
					} else {
						successes = append(successes, fmt.Sprintf("No project: **%v**", fmtDuration(time.Duration(totalTimeNano))))
					}
				}
				return shared.CommandResponse{
					ChannelID: data.ChannelID,
					Success:   strings.Join(successes, "\n"),
					Error: shared.CommandError{
						ErrorType:   shared.ExecutionError,
						ErrorString: strings.Join(groupingErrors, "\n"),
					},
				}
			}
		} else if data.Args[0] == "users" {
			if rows, err := initialSessionSet.Select("discord_user_id, sum(duration)").Group("discord_user_id").Rows(); err != nil {
				return shared.CommandResponse{
					ChannelID: data.ChannelID,
					Error: shared.CommandError{
						ErrorType:   shared.ExecutionError,
						ErrorString: "Failed to fetch brigade's volunteer sessions by user. Try again later.",
					},
				}
			} else {
				successes := make([]string, 0)
				successes = append(successes, "Total time for " + dataFor + " by user:")
				groupingErrors := make([]string, 0)
				var discordUserID string
				var totalTimeNano int
				for rows.Next() {
					if err := rows.Scan(&discordUserID, &totalTimeNano); err != nil {
						return shared.CommandResponse{
							ChannelID: data.ChannelID,
							Error: shared.CommandError{
								ErrorType:   shared.ExecutionError,
								ErrorString: "Failed to group volunteer sessions by user. Try again later.",
							},
						}
					}
					if _, err := data.Session.User(discordUserID); err != nil {
						groupingErrors = append(groupingErrors, "Failed to find user <@!"+discordUserID+">. Try again later.")
					} else {
						successes = append(successes, fmt.Sprintf("<@!%v>: **%v**", discordUserID, fmtDuration(time.Duration(totalTimeNano))))
					}
				}
				return shared.CommandResponse{
					ChannelID: data.ChannelID,
					Success:   strings.Join(successes, "\n"),
					Error: shared.CommandError{
						ErrorType:   shared.ExecutionError,
						ErrorString: strings.Join(groupingErrors, "\n"),
					},
				}
			}
		} else {
			return shared.CommandResponse{
				ChannelID: data.ChannelID,
				Error: shared.CommandError{
					ErrorType:   shared.ArgumentError,
					ErrorString: "Invalid category to sort by provided. Try `projects` or `users`.",
				},
			}
		}
	} else {
		var totalTimeNano int
		if err := initialSessionSet.Select("sum(duration)").Row().Scan(&totalTimeNano); err != nil {
			return shared.CommandResponse{
				ChannelID: data.ChannelID,
				Error:     shared.CommandError{
					ErrorType:   shared.ExecutionError,
					ErrorString: "Failed to fetch volunteer sessions. Try again later.",
				},
			}
		}
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Success:   fmt.Sprintf("Total time for %v: **%v**", dataFor, fmtDuration(time.Duration(totalTimeNano))),
		}
	}
}

// Move project to maintenance
func maintainProject(data shared.CommandData) shared.CommandResponse {
	channel, err := data.Session.Channel(data.Project.DiscordChannelID)
	githubChannel, err := data.Session.Channel(data.Project.GithubDiscordChannelID)
	if githubChannel == nil || channel == nil {
		fmt.Println("error fetching guild channels,", err)
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "Failed to fetch Discord channels for project. Try again later.",
			},
		}
	}
	if _, err = data.Session.ChannelDelete(githubChannel.ID); err != nil {
		fmt.Println("error deleting guild channel,", err)
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "Failed to delete GitHub channel for **" + data.Project.Name + "**. Have an administrator do this manually.",
			},
		}
	}
	editData := discordgo.ChannelEdit{
		ParentID: data.Brigade.InactiveProjectCategoryID,
	}
	if _, err = data.Session.ChannelEditComplex(channel.ID, &editData); err != nil {
		fmt.Println("error editing guild channel,", err)
		return shared.CommandResponse{
			ChannelID: data.ChannelID,
			Error: shared.CommandError{
				ErrorType:   shared.ExecutionError,
				ErrorString: "Failed to move discussion channel for **" + data.Project.Name + "**. Have an administrator do this manually.",
			},
		}
	}
	return shared.CommandResponse{
		ChannelID: data.ChannelID,
		Success:   "Successfully moved  **" + data.Project.Name + "** to maintenance.",
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
	if argCount == -1 {
		return "∞"
	}
	return strconv.Itoa(argCount)
}

func containsUser(slice []*discordgo.User, value *discordgo.User) bool {
	for _, item := range slice {
		if item.ID == value.ID {
			return true
		}
	}
	return false
}
