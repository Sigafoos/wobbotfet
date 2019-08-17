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

func want(pieces []string, m *discordgo.MessageCreate) string {
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
			failed = append(failed, w)
			continue
		} else if response.StatusCode != http.StatusCreated {
			log.Printf("error creating want: got %v\n", response.StatusCode)
			return "uh oh, something went wrong"
		}
		succeeded = append(succeeded, w)
	}
	var message string
	if len(succeeded) > 0 {
		message = "Added to your want list: " + strings.Join(succeeded, ", ")
	}
	if len(failed) > 0 {
		if len(message) > 0 {
			message += "\n\n"
		}
		message += "Failed adding: " + strings.Join(failed, ", ")
	}
	return message
}

func listWants(pieces []string, m *discordgo.MessageCreate) string {
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

func init() {
	registerCommand("want", want)
	registerCommand("wants", listWants)
}
