package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// GetActivePMs returns a list of private messages the bot is waiting for a response on.
func (a *API) GetActivePMs(w http.ResponseWriter, r *http.Request) {
	pms := a.bot.ActivePMs()

	b, err := json.Marshal(pms)
	if err != nil {
		log.Printf("error marshalling json: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(b)
}
