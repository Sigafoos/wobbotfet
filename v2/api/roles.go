package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// GetRoles returns the roles for a server.
func (a *API) GetRoles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	server, ok := vars["server"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	roles, err := a.bot.Roles(server)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(roles)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(b)
}
