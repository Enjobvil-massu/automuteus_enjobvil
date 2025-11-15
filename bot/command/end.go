package command

import (
	"github.com/bwmarrin/discordgo"
)

var End = discordgo.ApplicationCommand{
	Name:        "stop",
	Description: "End a game",
}
