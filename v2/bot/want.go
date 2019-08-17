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

func unwant(pieces []string, m *discordgo.MessageCreate) string {
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

func init() {
	registerCommand("want", want)
	registerCommand("wants", listWants)
	registerCommand("unwant", unwant)
}
