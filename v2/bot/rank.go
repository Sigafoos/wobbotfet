package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Sigafoos/iv/model"
	"github.com/bwmarrin/discordgo"
)

var rankURL = "https://ivservice.herokuapp.com/iv?pokemon=%s&ivs=%v/%v/%v"

func init() {
	url := os.Getenv("rankurl")
	if url != "" {
		rankURL = url
	}
	registerCommand("rank", rank, "`rank azumarill 4 1 3` to see the rank (out of 4096 possible combinations) of your IV spread's stat product")
	registerCommand("vrank", verboseRank, "`vrank azumarill 4 1 3` to get the same rank as `rank` with the values used in its calculation")
	registerCommand("betterthan", betterthanRank, "`betterthan azumarill 4 1 3` to see the chances of getting a better Pokemon from a variety of situations")
}

func rank(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	return getRank(pieces, m, false, false)
}

func verboseRank(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	return getRank(pieces, m, true, false)
}

func betterthanRank(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	return getRank(pieces, m, false, true)
}

func getRank(pieces []string, m *discordgo.MessageCreate, verbose bool, betterthan bool) string {
	query, err := parseQuery(pieces)
	if err != nil {
		return fmt.Sprintf("`%s` isn't a valid rank command (did you pass `4/1/3` instead of `4 1 3`?)", strings.Join(pieces, " "))
	}

	cmd := "rank"
	if verbose {
		cmd = "vrank"
	} else if betterthan {
		cmd = "betterthan"
	}

	access.Printf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", m.GuildID, m.ChannelID, m.Author.String(), cmd, query.League, query.Pokemon, query.Atk, query.Def, query.HP)

	atk, def, hp, err := parseIVs(query.Atk, query.Def, query.HP)
	if err != nil {
		return err.Error()
	}

	// temp?
	if query.League != "great" {
		return "sorry, but I only know how to handle great league right now"
	}

	parsedURL := fmt.Sprintf(rankURL, url.QueryEscape(query.Pokemon), atk, def, hp)
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

	rank := *spread.Ranks.All
	if query.Floor != "" {
		switch query.Floor {
		// all others can still be obtained in the wild so who cares
		case FloorHatched:
			rank = *spread.Ranks.Hatched
		}
	}

	message := fmt.Sprintf("your %s is rank %v (%v%%)", query.Pokemon, rank, (math.Trunc(spread.Percentage*100) / 100))

	if verbose {
		message = fmt.Sprintf("%s\n\nCP: `%v`\nLevel: `%v`\nAttack: `%v`\nDefense: `%v`\nHP: `%v`\nProduct: `%v`", message, spread.CP, spread.Level, spread.Stats.Attack, spread.Stats.Defense, spread.Stats.HP, spread.Product)
	}

	if betterthan == true {
		wild := float64(math.Round(float64(*spread.Ranks.All-1)/4096*100*100)) / 100
		good := float64(math.Round(float64(*spread.Ranks.Good-1)/3375*100*100)) / 100
		great := float64(math.Round(float64(*spread.Ranks.Great-1)/2744*100*100)) / 100
		ultra := float64(math.Round(float64(*spread.Ranks.Ultra-1)/2197*100*100)) / 100
		weather := float64(math.Round(float64(*spread.Ranks.Weather-1)/1728*100*100)) / 100
		best := float64(math.Round(float64(*spread.Ranks.Best-1)/1331*100*100)) / 100
		hatched := float64(math.Round(float64(*spread.Ranks.Hatched-1)/216*100*100)) / 100
		lucky := float64(math.Round(float64(*spread.Ranks.Lucky-1)/64*100*100)) / 100

		message = fmt.Sprintf("%s\n\nYour chances of getting a better %s:\n\n`%v%%`: Wild catch", message, query.Pokemon, wild)
		message = fmt.Sprintf("%s\n`%v%%`: Trade with Good Friend", message, good)
		message = fmt.Sprintf("%s\n`%v%%`: Trade with Great Friend", message, great)
		message = fmt.Sprintf("%s\n`%v%%`: Trade with Ultra Friend", message, ultra)
		message = fmt.Sprintf("%s\n`%v%%`: Weather boosted catch", message, weather)
		message = fmt.Sprintf("%s\n`%v%%`: Trade with Best Friend", message, best)
		message = fmt.Sprintf("%s\n`%v%%`: Hatched/Raid/Research", message, hatched)
		message = fmt.Sprintf("%s\n`%v%%`: Lucky Trade", message, lucky)
	}

	return message
}

func parseQuery(p []string) (Query, error) {
	floor, ok := FloorMap[p[len(p)-1]]
	if ok {
		p = p[:len(p)-1]
	}

	if len(p) < 4 {
		return Query{}, fmt.Errorf("not enough IVs")
	}
	q := Query{
		Atk:   p[len(p)-3],
		Def:   p[len(p)-2],
		HP:    p[len(p)-1],
		Floor: floor,
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
