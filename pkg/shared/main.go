package shared

import (
	"github.com/bwmarrin/discordgo"
	"github.com/codefordenver/codefordenver-scout/models"
)

type FunctionResponse struct {
	ChannelID string
	Success string
	Error string
}

type MessageData struct {
	ChannelID string
	Author    *discordgo.User
}

type CommandData struct {
	Session *discordgo.Session
	MessageData
	models.Brigade
	Args   []string
}
