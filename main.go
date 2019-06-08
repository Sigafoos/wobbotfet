package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/Sigafoos/iv/model"
	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

const serviceBase = "https://ivservice.herokuapp.com/iv?pokemon=%s&ivs=%v/%v/%v"

var client = &http.Client{}

var sigafoos = discordgo.User{Username: "Sigafoos#6538", ID: "193777776543662081"}

type Query struct {
	League  string
	Pokemon string
	Atk     string
	Def     string
	HP      string
}

var access *log.Logger

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	aLog, err := os.OpenFile("access.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("cannot open access log for writing: %s\n", err)
	}
	access = log.New(aLog, "", log.Ldate|log.Ltime)

	eLog, err := os.OpenFile("error.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("cannot open error log for writing: %s\n", err)
	}
	log.SetOutput(eLog)

	var env, auth string
	if len(os.Args) > 2 {
		env = "prod"
	} else {
		env = os.Args[1]
	}
	token := viper.GetString(fmt.Sprintf("token.%s", env))
	if token == "" {
		log.Fatalf("no token found for environment '%s'", env)
	}
	auth = "Bot " + token

	discord, err := discordgo.New(auth)
	if err != nil {
		log.Fatal(err)
	}
	err = discord.Open()
	if err != nil {
		log.Fatal(err)
	}

	discord.AddHandler(readMessage)
	fmt.Println("wooooooooobbotfett!")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// quitting time. clean up after ourselves
	discord.Close()

	err = aLog.Close()
	if err != nil {
		log.Printf("error closing access log: %s\n", err)
	}

	err = eLog.Close()
	if err != nil {
		log.Printf("error closing error log: %s\n", err)
	}

	fmt.Println("woooobotfet :(")
}

func readMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	var verbose bool
	// ignore messages posted by wobbotfet
	if m.Author.ID == s.State.User.ID {
		return
	}

	pieces := strings.Split(m.Content, " ")
	if pieces[0] == "!vrank" {
		verbose = true
	} else if pieces[0] != "!rank" {
		return
	}
	if pieces[1] == "help" {
		helpMessage(s, m)
		return
	}
	query := parseQuery(pieces)

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
	if verbose {
		message = fmt.Sprintf("%s\n\nCP: `%v`\nLevel: `%v`\nAttack: `%v`\nDefense: `%v`\nHP: `%v`\nProduct: `%v`", message, spread.CP, spread.Level, spread.Stats.Attack, spread.Stats.Defense, spread.Stats.HP, spread.Product)
	}
	s.ChannelMessageSend(m.ChannelID, message)
}

func helpMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s usage: `!rank <league?> <pokemon> <atk> <def> <sta>` (league is optional and defaults to `great`)\n\nCapitalization is irrelevant and `(`, `)` and `.` are stripped, so `Deoxys (Defense)` and `deoxys defense` are the same\n\n`!vrank` (for `verbose rank`) will give you the values of each stat as well as the product, in case you want to double check the values against other, less Wobby, IV services\n\nAny questions or concerns: ask %s", m.Author.Mention(), sigafoos.Mention()))
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

func parseQuery(p []string) Query {
	q := Query{
		Atk: p[len(p)-3],
		Def: p[len(p)-2],
		HP:  p[len(p)-1],
	}

	if p[1] == "great" || p[1] == "ultra" || p[1] == "master" {
		q.League = p[1]
		q.Pokemon = strings.Join(p[2:len(p)-3], " ")
	} else {
		q.League = "great"
		q.Pokemon = strings.Join(p[1:len(p)-3], " ")
	}

	return q
}
