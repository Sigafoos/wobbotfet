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
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/Sigafoos/iv/model"
	"github.com/bwmarrin/discordgo"
)

const serviceBase = "https://ivservice.herokuapp.com/iv?pokemon=%s&ivs=%v/%v/%v"

var client = &http.Client{}

var access *log.Logger

var aLog, eLog *os.File

var version string

type Bot struct {
	owner   *discordgo.User
	pm      *discordgo.Channel
	session *discordgo.Session
}

type Query struct {
	League  string
	Pokemon string
	Atk     string
	Def     string
	HP      string
}

func New(auth string, owner *discordgo.User) *Bot {
	var err error
	aLog, err = os.OpenFile("access.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("cannot open access log for writing: %s\n", err)
	}
	access = log.New(aLog, "", log.Ldate|log.Ltime)

	eLog, err = os.OpenFile("error.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("cannot open error log for writing: %s\n", err)
	}
	log.SetOutput(eLog)

	session, err := discordgo.New(auth)
	if err != nil {
		log.Fatal(err)
	}

	b := &Bot{
		owner:   owner,
		session: session,
	}
	session.AddHandler(b.readMessage)

	return b
}

func (b *Bot) Start() {
	err := b.session.Open()
	if err != nil {
		log.Fatal(err)
	}

	if b.owner != nil {
		b.pm, err = b.session.UserChannelCreate(b.owner.ID)
		if err != nil {
			log.Printf("error opening PM with owner: %s", err.Error())
		}
	}
	if version != "" {
		b.session.UpdateStatus(0, version)
	}
	b.PM("starting")
	fmt.Println("wooooooooobbotfett!")
}

func (b *Bot) Close() {
	b.PM("going down")
	b.session.Close()

	err := aLog.Close()
	if err != nil {
		log.Printf("error closing access log: %s\n", err)
	}

	err = eLog.Close()
	if err != nil {
		log.Printf("error closing error log: %s\n", err)
	}
}

func (b *Bot) PM(message string) {
	if b.pm == nil {
		return
	}
	b.session.ChannelMessageSend(b.pm.ID, message)
}

func (b *Bot) readMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	defer func() {
		if r := recover(); r != nil {
			b.PM(fmt.Sprintf("recovered from panic: %v\n\n`%v`", r, string(debug.Stack())))
		}
	}()

	var command string
	// ignore messages posted by wobbotfet
	if m.Author.ID == s.State.User.ID {
		return
	}

	pieces := strings.Split(strings.ToLower(m.Content), " ")

	command = "rank"
	if pieces[0] == "!vrank" {
		command = "verbose"
	} else if pieces[0] == "!betterthan" {
		command = "better"
	} else if pieces[0] != "!rank" {
		return
	}
	if len(pieces) < 2 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s `%s` isn't a command I can parse", m.Author.Mention(), m.Content))
		return
	}
	if pieces[1] == "help" {
		b.helpMessage(s, m)
		return
	}

	query, err := parseQuery(pieces)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s `%s` isn't a command I can parse (did you pass `4/1/3` instead of `4 1 3`?)", m.Author.Mention(), m.Content))
		return
	}

	access.Printf("\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", m.GuildID, m.ChannelID, m.Author.String(), query.League, query.Pokemon, query.Atk, query.Def, query.HP)

	atk, def, hp, err := parseIVs(query.Atk, query.Def, query.HP)

	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s %s", m.Author.Mention(), err))
		return
	}

	// temp?
	if query.League != "great" {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s sorry, but I only know how to handle great league right now", m.Author.Mention()))
		return
	}

	parsedURL := fmt.Sprintf(serviceBase, url.QueryEscape(query.Pokemon), atk, def, hp)
	req, err := http.NewRequest(http.MethodGet, parsedURL, nil)
	if err != nil {
		log.Println(err)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s sorry, something's gone wrong", m.Author.Mention()))
		return
	}
	req.Header.Add("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s sorry, something's gone wrong", m.Author.Mention()))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s `%s` isn't a valid Pokemon", m.Author.Mention(), query.Pokemon))
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("got non-200: %v\n", resp.StatusCode)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s sorry, something's gone wrong", m.Author.Mention()))
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s sorry, something's gone wrong", m.Author.Mention()))
		return
	}
	var spread model.Spread
	err = json.Unmarshal(body, &spread)
	if err != nil {
		log.Println(err)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s sorry, something's gone wrong", m.Author.Mention()))
		return
	}

	message := fmt.Sprintf("%s your %s is rank %v (%v%%)", m.Author.Mention(), query.Pokemon, *spread.Ranks.All, (math.Trunc(spread.Percentage*100) / 100))

	if command == "better" {
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
	} else if command == "verbose" {
		message = fmt.Sprintf("%s\n\nCP: `%v`\nLevel: `%v`\nAttack: `%v`\nDefense: `%v`\nHP: `%v`\nProduct: `%v`", message, spread.CP, spread.Level, spread.Stats.Attack, spread.Stats.Defense, spread.Stats.HP, spread.Product)
	}
	s.ChannelMessageSend(m.ChannelID, message)
}

func (b *Bot) helpMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	help := fmt.Sprintf("%s usage: `!<command> <league?> <pokemon> <atk> <def> <sta>` (league is optional and defaults to `great`)\n\nCapitalization is irrelevant and `(`, `)` and `.` are stripped, so `Deoxys (Defense)` and `deoxys defense` are the same\n\n**Commands**\n`!rank` will tell you the rank of your IV spread\n`!vrank` (for `verbose rank`) will give you the values of each stat as well as the product, in case you want to double check the values against other, less Wobby, IV services\n`!betterthan` will tell you the odds of obtaining a higher rank", m.Author.Mention())
	if b.owner != nil {
		help += "\n\nAny questions or concerns: ask " + b.owner.Mention()
	}
	s.ChannelMessageSend(m.ChannelID, help)
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

func parseQuery(p []string) (Query, error) {
	if len(p) < 5 {
		return Query{}, fmt.Errorf("not enough IVs")
	}
	q := Query{
		Atk: p[len(p)-3],
		Def: p[len(p)-2],
		HP:  p[len(p)-1],
	}

	if p[1] == "great" || p[1] == "ultra" || p[1] == "master" {
		if len(p) < 6 {
			return q, fmt.Errorf("not enough IVs")
		}
		q.League = p[1]
		q.Pokemon = strings.Join(p[2:len(p)-3], " ")
	} else {
		q.League = "great"
		q.Pokemon = strings.Join(p[1:len(p)-3], " ")
	}

	return q, nil
}
