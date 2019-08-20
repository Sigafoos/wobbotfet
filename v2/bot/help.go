package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func runHelp(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	message := "here is what you can ask me:\n"

	for command, text := range help {
		message = fmt.Sprintf("%s\n**%s**: %s", message, command, text)
	}
	return message
}

func init() {
	registerCommand("help", runHelp, "print this message")
}
