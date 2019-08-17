package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Sigafoos/iv/model"
	"github.com/bwmarrin/discordgo"
)

const serviceBase = "https://ivservice.herokuapp.com/iv?pokemon=%s&ivs=%v/%v/%v"

func init() {
	registerCommand("rank", rank)
	registerCommand("vrank", verboseRank)
}

func rank(pieces []string, m *discordgo.MessageCreate) string {
	return getRank(pieces, false)
}

func verboseRank(pieces []string, m *discordgo.MessageCreate) string {
	return getRank(pieces, true)
}

func getRank(pieces []string, verbose bool) string {
	query, err := parseQuery(pieces)
	if err != nil {
		return fmt.Sprintf("`%s` isn't a valid rank command (did you pass `4/1/3` instead of `4 1 3`?)", strings.Join(pieces, " "))
	}

	//access.Printf("\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", m.GuildID, m.ChannelID, m.Author.String(), query.League, query.Pokemon, query.Atk, query.Def, query.HP)

	atk, def, hp, err := parseIVs(query.Atk, query.Def, query.HP)
	if err != nil {
		return err.Error()
	}

	// temp?
	if query.League != "great" {
		return "sorry, but I only know how to handle great league right now"
	}

	parsedURL := fmt.Sprintf(serviceBase, url.QueryEscape(query.Pokemon), atk, def, hp)
	req, err := http.NewRequest(http.MethodGet, parsedURL, nil)
	if err != nil {
		log.Println(err)
		return "sorry, something's gone wrong"
	}
	req.Header.Add("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return "sorry, something's gone wrong"
	}

	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Sprintf("`%s` isn't a valid Pokemon", query.Pokemon)
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("got non-200 (%v) on %s\n", resp.StatusCode, parsedURL)
		return "sorry, something's gone wrong"
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return "sorry, something's gone wrong"
	}

	var spread model.Spread
	err = json.Unmarshal(body, &spread)
	if err != nil {
		log.Println(err)
		return "sorry, something's gone wrong"
	}

	message := fmt.Sprintf("your %s is rank %v (%v%%)", query.Pokemon, *spread.Ranks.All, (math.Trunc(spread.Percentage*100) / 100))

	if verbose {
		message = fmt.Sprintf("%s\n\nCP: `%v`\nLevel: `%v`\nAttack: `%v`\nDefense: `%v`\nHP: `%v`\nProduct: `%v`", message, spread.CP, spread.Level, spread.Stats.Attack, spread.Stats.Defense, spread.Stats.HP, spread.Product)
	}

	return message
}

func parseQuery(p []string) (Query, error) {
	if len(p) < 4 {
		return Query{}, fmt.Errorf("not enough IVs")
	}
	q := Query{
		Atk: p[len(p)-3],
		Def: p[len(p)-2],
		HP:  p[len(p)-1],
	}

	if p[1] == "great" || p[1] == "ultra" || p[1] == "master" {
		if len(p) < 5 {
			return q, fmt.Errorf("not enough IVs")
		}
		q.League = p[0]
		q.Pokemon = strings.Join(p[1:len(p)-3], " ")
	} else {
		q.League = "great"
		q.Pokemon = strings.Join(p[:len(p)-3], " ")
	}

	return q, nil
}

func parseIVs(atk, def, hp string) (iAtk, iDef, iHP int, err error) {
	errMsg := "`%s` doesn't appear to be a number between 0-15"
	iAtk, err = strconv.Atoi(atk)
	if err != nil || iAtk < 0 || iAtk > 15 {
		err = fmt.Errorf(errMsg, atk)
		return
	}
	iDef, err = strconv.Atoi(def)
	if err != nil || iDef < 0 || iDef > 15 {
		err = fmt.Errorf(errMsg, def)
		return
	}
	iHP, err = strconv.Atoi(hp)
	if err != nil || iHP < 0 || iHP > 15 {
		err = fmt.Errorf(errMsg, hp)
	}
	return
}
