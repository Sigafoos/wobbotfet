package bot

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/Sigafoos/pokemongo"
	"github.com/bwmarrin/discordgo"
)

type Request struct {
	User    string `json:"user,omitempty"`
	Pokemon string `json:"pokemon,omitempty"`
}

const wantURL = "http://localhost:8080/want"

const applicationJSON = "application/json"

const errorForbidden = "HTTP 403 Forbidden"

// this can probably be abstracted into modifyWants or something; just have to handle errors
func want(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	if len(pieces) == 0 {
		return "you need to specify one or more Pokemon (to see wants, use `wants`)"
	}

	var succeeded []string
	var failed []string
	for _, w := range pieces {
		b, err := json.Marshal(&Request{User: m.Author.ID, Pokemon: w})
		if err != nil {
			log.Println(err)
			failed = append(failed, w)
			continue
		}

		req, err := http.NewRequest(http.MethodPost, wantURL, bytes.NewReader(b))
		if err != nil {
			log.Println(err)
			failed = append(failed, w)
			continue
		}
		req.Header.Add("Content-Type", applicationJSON)
		req.Header.Add("Accept", applicationJSON)

		response, err := client.Do(req)
		if err != nil {
			log.Println(err)
			failed = append(failed, w)
			continue
		}
		if response.Body != nil {
			defer response.Body.Close()
		}

		if response.StatusCode == http.StatusNotFound {
			failed = append(failed, w+" (no such Pokemon)")
			continue
		}

		if response.StatusCode == http.StatusConflict {
			failed = append(failed, w+" (already wanted)")
			continue
		}

		if response.StatusCode != http.StatusCreated {
			log.Printf("error creating want: got %v\n", response.StatusCode)
			return "uh oh, something went wrong"
		}
		succeeded = append(succeeded, w)
		addRole(w, m, s)
	}
	var message string
	if len(succeeded) > 0 {
		message = "added to your want list: " + strings.Join(succeeded, ", ")
	}
	if len(failed) > 0 {
		if len(message) > 0 {
			message += "\n\n"
		}
		message += "failed adding: " + strings.Join(failed, ", ")
	}
	return message
}

func listWants(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	req, err := http.NewRequest(http.MethodGet, wantURL+"?user="+m.Author.ID, nil)
	if err != nil {
		log.Println(err)
		return "uh oh, something's gone wrong"
	}
	req.Header.Add("Content-Type", applicationJSON)
	req.Header.Add("Accept", applicationJSON)

	response, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return "uh oh, something's gone wrong"
	}
	defer response.Body.Close()

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return "sorry, something's gone wrong"
	}

	var pokemon []pokemongo.Pokemon
	err = json.Unmarshal(b, &pokemon)
	if err != nil {
		log.Println(err)
		return "sorry, something's gone wrong"
	}

	var names []string
	for _, p := range pokemon {
		names = append(names, p.Name)
	}

	return "your wants: " + strings.Join(names, ", ")
}

func unwant(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	var succeeded []string
	var failed []string
	for _, w := range pieces {
		b, err := json.Marshal(&Request{User: m.Author.ID, Pokemon: w})
		if err != nil {
			log.Println(err)
			failed = append(failed, w)
			continue
		}

		req, err := http.NewRequest(http.MethodDelete, wantURL, bytes.NewReader(b))
		if err != nil {
			log.Println(err)
			failed = append(failed, w)
			continue
		}
		req.Header.Add("Content-Type", applicationJSON)
		req.Header.Add("Accept", applicationJSON)

		response, err := client.Do(req)
		if err != nil {
			log.Println(err)
			failed = append(failed, w)
			continue
		}
		if response.Body != nil {
			defer response.Body.Close()
		}

		if response.StatusCode == http.StatusNotFound {
			failed = append(failed, w+" (no such Pokemon)")
			continue
		}

		if response.StatusCode != http.StatusOK {
			log.Printf("error creating want: got %v\n", response.StatusCode)
			return "uh oh, something went wrong"
		}
		succeeded = append(succeeded, w)
		removeRole(w, m, s)
	}
	var message string
	if len(succeeded) > 0 {
		message = "removed from your want list: " + strings.Join(succeeded, ", ")
	}
	if len(failed) > 0 {
		if len(message) > 0 {
			message += "\n\n"
		}
		message += "failed removing: " + strings.Join(failed, ", ")
	}
	return message
}

// add a role to a user. creates it if it doesn't exist. on error, log it and silently return.
//
// currently a bit of a mess.
func addRole(roleName string, m *discordgo.MessageCreate, s *discordgo.Session) {
	// don't bother if it's in a PM
	if m.GuildID == "" {
		return
	}

	role := getRole(roleName, m.GuildID, s)
	if role == nil {
		role, err := s.GuildRoleCreate(m.GuildID)
		if err != nil {
			// don't log if the admin just hasn't granted permissions
			if !strings.HasPrefix(err.Error(), errorForbidden) {
				log.Printf("error creating roleName in guild %s: %s\n", m.GuildID, err.Error())
			}
			return
		}
		role, err = s.GuildRoleEdit(m.GuildID, role.ID, roleName, 0, false, 0, true)
		if err != nil {
			log.Printf("error updating roleName %s in guild %s: %s\n", roleName, m.GuildID, err.Error())

			err = s.GuildRoleDelete(m.GuildID, role.ID)
			if err != nil {
				log.Printf("error deleting roleName %s om guild %s: %s\n", roleName, m.GuildID, err.Error())
			}
			return
		}
	}

	err := s.GuildMemberRoleAdd(m.GuildID, m.Author.ID, role.ID)
	if err != nil {
		log.Printf("error adding roleName %s to user %s in guild %s: %s\n", roleName, m.Author.Username, m.GuildID, err.Error())
	}
}

func removeRole(roleName string, m *discordgo.MessageCreate, s *discordgo.Session) {
	// don't bother if it's in a PM
	if m.GuildID == "" {
		return
	}

	role := getRole(roleName, m.GuildID, s)
	if role == nil {
		return
	}

	err := s.GuildMemberRoleRemove(m.GuildID, m.Author.ID, role.ID)
	if err != nil {
		// don't log if the admin just hasn't granted permissions
		if !strings.HasPrefix(err.Error(), errorForbidden) {
			log.Printf("error removing role %s from user %s in guild %s: %s\n", roleName, m.Author.Username, m.GuildID, err.Error())
		}
	}
}

func getRole(roleName, guildID string, s *discordgo.Session) *discordgo.Role {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		log.Printf("error geetting roles for guild %s: %s\n", guildID, err.Error())
		return nil
	}

	for _, r := range roles {
		if r.Name == roleName {
			return r
		}
	}
	return nil
}

func init() {
	registerCommand("want", want)
	registerCommand("wants", listWants)
	registerCommand("unwant", unwant)
}
