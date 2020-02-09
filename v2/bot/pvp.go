package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Sigafoos/pvpservice/pvp"
	"github.com/bwmarrin/discordgo"
)

const (
	BattleTime = 30
)

var (
	pvpURL = os.Getenv("PVP_URL")
	p      *PVP
)

func init() {
	if pvpURL == "" {
		log.Println("no PVP_URL specified; cannot run pvp command")
		return
	}
	p := newPVP()
	registerCommand("pvp", p.Handle, "PVP friend tracking/battle announcing. `pvp help` for more details")
}

type Answer int

const (
	AnswerUnknown Answer = iota
	AnswerYes
	AnswerNo
	AnswerCancel
)

type PVP struct {
	registering map[string]pvp.Player
	battling    map[string]map[string]*time.Timer
}

func newPVP() *PVP {
	registering := make(map[string]pvp.Player)
	battling := make(map[string]map[string]*time.Timer)
	return &PVP{
		registering: registering,
		battling:    battling,
	}
}

func (p *PVP) Handle(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	if m.GuildID == "" {
		return "You can only ask this in a server!"
	}

	switch pieces[0] {
	case "help":
		return p.Help()
	case "register":
		p.AskForIGN(m, s)
		return "I'll PM you for details!"
	case "battle":
		return p.LookingForBattle(pieces[1:], m, s)
	}
	return fmt.Sprintf("I don't have a command `pvp %s`", pieces[0])
}

func (p *PVP) Help() string {
	return "`pvp register`: sign up for PVP battles! I'll PM you to ask for your information."
}

func (p *PVP) LookingForBattle(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	length := BattleTime
	if len(pieces) > 0 {
		if pieces[0] == "0" || pieces[0] == "stop" || pieces[0] == "end" {
			return p.StopBattling(m)
		}
		l, err := strconv.Atoi(pieces[0])
		if err != nil {
			return fmt.Sprintf("`%s` doesn't seem to be a number", pieces[0])
		}
		length = l
	}
	return p.StartBattling(length, m, s)
}

func (p *PVP) StartBattling(length int, m *discordgo.MessageCreate, s *discordgo.Session) string {
	log.Printf("%v minute timer starting", length)
	if _, ok := p.battling[m.GuildID]; !ok {
		p.battling[m.GuildID] = make(map[string]*time.Timer)
	}

	p.battling[m.GuildID][m.Author.ID] = time.AfterFunc(time.Duration(length)*time.Minute, func() {
		s.ChannelMessageSend(m.ChannelID, "youre not battling anymore")
		log.Println("timer up")
		delete(p.battling[m.GuildID], m.Author.ID)
	})

	// actually dont return a string, and have it ping the channel
	return fmt.Sprintf("you're looking for battles for the next %v minutes!", length)
}

func (p *PVP) StopBattling(m *discordgo.MessageCreate) string {
	server, ok := p.battling[m.GuildID]
	if !ok {
		return "You aren't currently looking for battles!"
	}
	timer, ok := server[m.Author.ID]
	if !ok {
		return "You aren't currently looking for battles!"
	}
	timer.Stop()
	delete(p.battling[m.GuildID], m.Author.ID)
	// TODO the group
	return "You're no longer looking for battles"
}

func (p *PVP) GetPlayers(server string) []pvp.Player {
	var players []pvp.Player

	req, err := http.NewRequest(http.MethodGet, pvpURL+"/player/list?server="+server, nil)
	if err != nil {
		log.Printf("error creating player list request: %s", err.Error())
		return players
	}
	req.Header.Add("Content-Type", applicationJSON)
	req.Header.Add("Accept", applicationJSON)

	response, err := client.Do(req)
	if err != nil {
		log.Printf("error executing player list: %s", err.Error())
		return players
	}
	defer response.Body.Close()

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("error reading player list body: %s", err.Error())
		return players
	}

	err = json.Unmarshal(b, &players)
	if err != nil {
		log.Printf("error unmarshaling player list: %s", err.Error())
	}
	return players
}

// ListPlayers returns a list of participants in PVP. It's currently unused, as we decided
// it was better to not allow randos to see friend codes.
func (p *PVP) ListPlayers(m *discordgo.MessageCreate, s *discordgo.Session) string {
	var guildName string
	guild, err := s.Guild(m.GuildID)
	if err != nil {
		log.Printf("error getting guild name from id: %s", err.Error())
		guildName = "This server"
	} else {
		guildName = guild.Name
	}

	players := p.GetPlayers(m.GuildID)
	if len(players) == 0 {
		return guildName + " has no active players! Use `pvp register` to be the first!"
	}

	message := fmt.Sprintf("Here are the PVP players of %s:\n\n", guildName)
	for _, player := range players {
		message += player.ToString() + "\n"
	}
	return message
}

func (p *PVP) AskForIGN(m *discordgo.MessageCreate, s *discordgo.Session) {
	// TODO timeout
	p.registering[m.Author.ID] = pvp.Player{
		ID:       m.Author.ID,
		Username: m.Author.Username + "#" + m.Author.Discriminator,
		Server:   m.GuildID,
	}
	message := "Thanks for your interest in PVP"
	if guild, err := s.Guild(m.GuildID); err == nil {
		message += " at " + guild.Name
	}
	message += "! What's your in-game name (IGN)?"
	pm := startPM(s, m.Author.ID)
	if pm == nil {
		s.ChannelMessageSend(m.ChannelID, "uh oh, something went wrong")
		return
	}
	expectPM(pm.ID, p.AskForFriendCode)
	s.ChannelMessageSend(pm.ID, message)
}

func (p *PVP) AskForFriendCode(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	player, ok := p.registering[m.Author.ID]
	if !ok {
		return "Well this is awkward. You need to start the registration process over. Sorry!"
	}
	player.IGN = pieces[0]
	p.registering[m.Author.ID] = player

	expectPM(m.ChannelID, p.SaveFriendCode)
	return "Great! What's your friend code? You can put in spaces or not; I don't care."
}

func (p *PVP) SaveFriendCode(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	player, ok := p.registering[m.Author.ID]
	if !ok {
		return "Well this is awkward. You need to start the registration process over. Sorry!"
	}
	player.FriendCode = strings.Join(pieces, "")
	p.registering[m.Author.ID] = player
	expectPM(m.ChannelID, p.EggForUltra)
	return "Do you use a lucky egg for ultra friendships?"
}

func (p *PVP) EggForUltra(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	player, ok := p.registering[m.Author.ID]
	if !ok {
		return "Well this is awkward. You need to start the registration process over. Sorry!"
	}
	response := p.parseAnswer(pieces)
	if response == AnswerUnknown {
		expectPM(m.ChannelID, p.EggForUltra)
		return fmt.Sprintf("Sorry, I don't understand the answer '%s'. Please say 'yes' or 'no'.", strings.Join(pieces, " "))
	}
	if response == AnswerCancel {
		delete(p.registering, m.Author.ID)
		return "Okay, start again when you're ready"
	}
	if response == AnswerYes {
		player.EggUltra = true
	}
	if response == AnswerNo {
		player.EggUltra = false
	}

	expectPM(m.ChannelID, p.ConfirmInfo)
	return fmt.Sprintf("Does this look right?\n\nIn-game name: %s\nFriend code: %s\nEgg for ultra: %v", player.IGN, player.FriendCode, player.EggUltra)
}

func (p *PVP) ConfirmInfo(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	if _, ok := p.registering[m.Author.ID]; !ok {
		return "Well this is awkward. You need to start the registration process over. Sorry!"
	}
	response := p.parseAnswer(pieces)
	if response == AnswerYes {
		return p.RegisterPlayer(m.Author.ID, s)
	}
	if response == AnswerNo {
		delete(p.registering, m.Author.ID)
		expectPM(m.ChannelID, p.AskForFriendCode)
		return "Okay, let's start over. What's your in-game name?"
	}
	if response == AnswerCancel {
		delete(p.registering, m.Author.ID)
		return "Okay, start again when you're ready"
	}

	expectPM(m.ChannelID, p.ConfirmInfo)
	return fmt.Sprintf("Sorry, I don't understand the answer '%s'. Please say 'yes' or 'no'.", strings.Join(pieces, " "))
}

func (p *PVP) RegisterPlayer(ID string, s *discordgo.Session) string {
	player, ok := p.registering[ID]
	if !ok {
		return "Well this is awkward. You need to start the registration process over. Sorry!"
	}
	b, err := json.Marshal(&player)
	if err != nil {
		log.Printf("error marshalling player json: %s", err.Error())
		return "sorry, something's gone wrong"
	}
	req, err := http.NewRequest(http.MethodPost, pvpURL+"/register", bytes.NewReader(b))
	if err != nil {
		log.Printf("error creating player request: %s", err.Error())
		return "sorry, something's gone wrong"
	}
	req.Header.Add("Content-Type", applicationJSON)
	req.Header.Add("Accept", applicationJSON)

	response, err := client.Do(req)
	if err != nil {
		log.Printf("error performing player register request: %s", err.Error())
		return "sorry, something's gone wrong"
	}
	if response.Body != nil {
		defer response.Body.Close()
	}

	if response.StatusCode == http.StatusConflict {
		return "Wait, you're registered already!"
	}

	if response.StatusCode != http.StatusCreated {
		log.Printf("error registering user: got %v\n", response.StatusCode)
		return "uh oh, something went wrong"
	}

	players := p.GetPlayers(player.Server)

	message := "You're all set!"

	if len(players) > 1 {
		message += " Here's who you need to send a friend request to (they've been told it's coming):\n\n"
		var guildName string
		guild, err := s.Guild(player.Server)
		if err != nil {
			log.Printf("error getting guild id for %s: %s", player.Server, err.Error())
			guildName = "a server you're in"
		} else {
			guildName = guild.Name
		}
		for _, opponent := range players {
			if opponent.ID != player.ID {
				message += opponent.ToString() + "\n"

				pm := startPM(s, opponent.ID)
				if pm != nil {
					joinPM := fmt.Sprintf("Hey there, %s! You'll be getting a friend request from %s soon, because they just signed up for PVP on %s!", opponent.IGN, player.IGN, guildName)
					s.ChannelMessageSend(pm.ID, joinPM)
				}
			}
		}
	}
	return message
}

func (p *PVP) parseAnswer(pieces []string) Answer {
	response := strings.ToLower(strings.Join(pieces, " "))
	if response == "y" || response == "yes" || response == "yep" || response == "yup" || response == "hell yeah" {
		return AnswerYes
	}
	if response == "n" || response == "no" || response == "nope" || response == "nah" || response == "fuck you" {
		return AnswerNo
	}
	if response == "c" || response == "cancel" || response == "stop" || response == "unsubscribe" {
		return AnswerCancel
	}

	return AnswerUnknown
}
