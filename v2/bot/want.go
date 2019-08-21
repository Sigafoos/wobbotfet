package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Sigafoos/pokemongo"
	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

type Request struct {
	User    string `json:"user,omitempty"`
	Pokemon string `json:"pokemon,omitempty"`
}

var (
	wantURL = "http://pokewants.herokuapp.com"
)

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
		access.Printf("%s\t%s\t%s\twant\t%s\n", m.GuildID, m.ChannelID, m.Author.String(), w)
		b, err := json.Marshal(&Request{User: m.Author.ID, Pokemon: w})
		if err != nil {
			log.Println(err)
			failed = append(failed, w)
			continue
		}

		req, err := http.NewRequest(http.MethodPost, wantURL+"/want", bytes.NewReader(b))
		if err != nil {
			log.Println(err)
			failed = append(failed, w)
			continue
		}
		req.Header.Add("Content-Type", applicationJSON)
		req.Header.Add("Accept", applicationJSON)
		basicuser := viper.GetString("want.basicauth.user")
		basicpass := viper.GetString("want.basicauth.pass")
		if basicuser != "" && basicpass != "" {
			req.SetBasicAuth(basicuser, basicpass)
		}

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
	access.Printf("%s\t%s\t%s\twants\n", m.GuildID, m.ChannelID, m.Author.String())
	req, err := http.NewRequest(http.MethodGet, wantURL+"/want?user="+m.Author.ID, nil)
	if err != nil {
		log.Println(err)
		return "uh oh, something's gone wrong"
	}
	req.Header.Add("Content-Type", applicationJSON)
	req.Header.Add("Accept", applicationJSON)
	basicuser := viper.GetString("want.basicauth.user")
	basicpass := viper.GetString("want.basicauth.pass")
	if basicuser != "" && basicpass != "" {
		req.SetBasicAuth(basicuser, basicpass)
	}

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
		access.Printf("%s\t%s\t%s\tunwant\t%s", m.GuildID, m.ChannelID, m.Author.String(), w)
		b, err := json.Marshal(&Request{User: m.Author.ID, Pokemon: w})
		if err != nil {
			log.Println(err)
			failed = append(failed, w)
			continue
		}

		req, err := http.NewRequest(http.MethodDelete, wantURL+"/want", bytes.NewReader(b))
		if err != nil {
			log.Println(err)
			failed = append(failed, w)
			continue
		}
		req.Header.Add("Content-Type", applicationJSON)
		req.Header.Add("Accept", applicationJSON)
		basicuser := viper.GetString("want.basicauth.user")
		basicpass := viper.GetString("want.basicauth.pass")
		if basicuser != "" && basicpass != "" {
			req.SetBasicAuth(basicuser, basicpass)
		}

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

func searchForPokemon(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	if len(pieces) < 1 {
		return "I need a Pokemon to search for!"
	}
	if len(pieces) > 1 {
		return "you can only search for one Pokemon at a time"
	}

	access.Printf("%s\t%s\t%s\tsearch\t%s\n", m.GuildID, m.ChannelID, m.Author.String(), pieces[0])
	req, err := http.NewRequest(http.MethodGet, wantURL+"/search?name="+pieces[0], nil)
	if err != nil {
		log.Println(err)
		return "uh oh, something's gone wrong"
	}

	req.Header.Add("Content-Type", applicationJSON)
	req.Header.Add("Accept", applicationJSON)
	basicuser := viper.GetString("want.basicauth.user")
	basicpass := viper.GetString("want.basicauth.pass")
	if basicuser != "" && basicpass != "" {
		req.SetBasicAuth(basicuser, basicpass)
	}

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
		names = append(names, p.ID)
	}

	return fmt.Sprintf("`%s` matches: %s", pieces[0], strings.Join(names, ", "))
}

// add a role to a user. creates it if it doesn't exist. on error, log it and silently return.
//
// currently a bit of a mess.
func addRole(roleName string, m *discordgo.MessageCreate, s *discordgo.Session) {
	// don't bother if it's in a PM
	if m.GuildID == "" {
		return
	}

	var err error
	role := getRole(roleName, m.GuildID, s)
	if role == nil {
		role, err = s.GuildRoleCreate(m.GuildID)
		if err != nil {
			// don't log if the admin just hasn't granted permissions
			if !strings.HasPrefix(err.Error(), errorForbidden) {
				log.Printf("error creating roleName in guild %s: %s\n", m.GuildID, err.Error())
			}
			return
		}
		_, err = s.GuildRoleEdit(m.GuildID, role.ID, roleName, 0, false, 0, true)
		if err != nil {
			log.Printf("error updating roleName %s in guild %s: %s\n", roleName, m.GuildID, err.Error())

			err = s.GuildRoleDelete(m.GuildID, role.ID)
			if err != nil {
				log.Printf("error deleting roleName %s om guild %s: %s\n", roleName, m.GuildID, err.Error())
			}
			return
		}
	}

	err = s.GuildMemberRoleAdd(m.GuildID, m.Author.ID, role.ID)
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
	url := os.Getenv("wanturl")
	if url != "" {
		wantURL = url
	}
	registerCommand("want", want, "`want wobbuffet` to add to your wants")
	registerCommand("unwant", unwant, "`unwant wobbuffet` to remove from your wants")
	registerCommand("wants", listWants, "list your wants")
	registerCommand("search", searchForPokemon, "search for Pokemon by name")
}
