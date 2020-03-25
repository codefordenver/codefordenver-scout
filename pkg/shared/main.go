package shared

import (
	"github.com/bwmarrin/discordgo"
	"github.com/codefordenver/codefordenver-scout/models"
)

type Permission int

type ExecutionContext int

type ErrorType int

const (
	ContextDM ExecutionContext = 1 << iota
	ContextBrigade
	ContextProject
	ContextAny = ContextBrigade | ContextProject | ContextDM
)

const (
	PermissionAdmin Permission = 1 << iota
	PermissionMember
	PermissionEveryone
)

const (
	ArgumentError ErrorType = iota
	ExecutionError
)

type CommandError struct {
	ErrorType
	ErrorString string
}

type CommandResponse struct {
	ChannelID string
	Success   string
	Error     CommandError
}

type MessageData struct {
	ChannelID string
	Author    *discordgo.User
}

type CommandData struct {
	Session *discordgo.Session
	MessageData
	*models.Brigade
	*models.Project
	Args []string
	BrigadeArg string
	ProjectArg string
}
