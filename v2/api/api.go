package api

import (
	"github.com/Sigafoos/wobbotfet/bot"
)

// An API is the handler for the wob dashboard api.
type API struct {
	bot *bot.Bot
}

// New returns a new API.
func New(b *bot.Bot) *API {
	return &API{
		bot: b,
	}
}
