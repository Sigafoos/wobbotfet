package bot

import (
	"github.com/bwmarrin/discordgo"
)

func help(pieces []string, m *discordgo.MessageCreate) string {
	return "usage: `!<command> <league?> <pokemon> <atk> <def> <sta>` (league is optional and defaults to `great`)\n\nCapitalization is irrelevant and `(`, `)` and `.` are stripped, so `Deoxys (Defense)` and `deoxys defense` are the same\n\n**Commands**\n`!rank` will tell you the rank of your IV spread\n`!vrank` (for `verbose rank`) will give you the values of each stat as well as the product, in case you want to double check the values against other, less Wobby, IV services\n`!betterthan` will tell you the odds of obtaining a higher rank"
}

func init() {
	registerCommand("help", help)
}
