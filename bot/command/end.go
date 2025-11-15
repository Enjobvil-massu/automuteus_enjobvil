package command

import (
	"github.com/bwmarrin/discordgo"
)

var End = discordgo.ApplicationCommand{
	Name:        "stop",
	Description: "オートミュートを停止します",
}
