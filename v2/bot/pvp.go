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
	session     *discordgo.Session
	registering map[string]pvp.Player
	friendship  map[string]string
	battling    map[string]map[string]*time.Timer
}

func newPVP() *PVP {
	registering := make(map[string]pvp.Player)
	friendship := make(map[string]string)
	battling := make(map[string]map[string]*time.Timer)
	return &PVP{
		registering: registering,
		friendship:  friendship,
		battling:    battling,
	}
}

func (p *PVP) Handle(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	// the session pointer is identical between messages. if we don't have one, or it's changed, use the current one
	if s != p.session {
		p.session = s
	}

	switch pieces[0] {
	case "help":
		return p.Help()
	case "register":
		if m.GuildID == "" {
			return "You can only do this in a server!"
		}

		user := p.getUser(m.Author.ID)

		if user == nil {
			p.AskForIGN(m)
			return "I'll PM you for details!"
		}

		user.Server = m.GuildID
		resp := p.RegisterPlayer(user)
		if resp != "" {
			return resp
		}
		return "You're all set! I'll PM you the friend codes."

	case "list":
		if m.GuildID != "" {
			return "You have to ask this in a PM, sorry!"
		}

		return p.ListPlayers(m)

	case "ultra":
		user := p.getUser(m.Author.ID)
		if user == nil {
			return "you aren't registered!"
		}
		if len(user.Servers) == 0 {
			return "you aren't in any servers!"
		}
		toFriend := p.NotUltraForPlayer(user)

		if pieces[1] == "todo" {
			if m.GuildID != "" {
				return "you can only ask for your to-be-ultra list in a PM"
			}
			var todo []string
			for _, player := range toFriend {
				todo = append(todo, player.ToString())
			}
			return fmt.Sprintf("Here's who you still need to reach ultra with:\n\n%s\n\nTell me `pvp ultra (IGN)` to list yourself as ultra (I'll confirm with them first!)", strings.Join(todo, "\n"))
		}

		// are you able to friend this person?
		for ign, player := range toFriend {
			if strings.ToLower(ign) == pieces[1] {
				return p.ConfirmFriendship(user, &player)
			}
		}
		return fmt.Sprintf("`%s` isn't someone you're in a PVP server with. Did you mean `pvp ultra todo` for the list of players you need to friend?", pieces[1])

	case "battle":
		if m.GuildID == "" {
			return "You can only do this in a server!"
		}

		return p.LookingForBattle(pieces[1:], m)
	}
	return fmt.Sprintf("I don't have a command `pvp %s`", pieces[0])
}

func (p *PVP) Help() string {
	return "`pvp register`: sign up for PVP battles! I'll PM you to ask for your information."
}

func (p *PVP) LookingForBattle(pieces []string, m *discordgo.MessageCreate) string {
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
	return p.StartBattling(length, m)
}

func (p *PVP) StartBattling(length int, m *discordgo.MessageCreate) string {
	log.Printf("%v minute timer starting", length)
	if _, ok := p.battling[m.GuildID]; !ok {
		p.battling[m.GuildID] = make(map[string]*time.Timer)
	}

	p.battling[m.GuildID][m.Author.ID] = time.AfterFunc(time.Duration(length)*time.Minute, func() {
		p.session.ChannelMessageSend(m.ChannelID, "youre not battling anymore")
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
func (p *PVP) deprecatedListPlayers(m *discordgo.MessageCreate, s *discordgo.Session) string {
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

func (p *PVP) AskForIGN(m *discordgo.MessageCreate) {
	// TODO timeout
	p.registering[m.Author.ID] = pvp.Player{
		ID:       m.Author.ID,
		Username: m.Author.Username + "#" + m.Author.Discriminator,
		Server:   m.GuildID,
	}
	message := "Thanks for your interest in PVP"
	if guild, err := p.session.Guild(m.GuildID); err == nil {
		message += " at " + guild.Name
	}
	message += "! What's your in-game name (IGN)?"
	pm := startPM(p.session, m.Author.ID)
	if pm == nil {
		p.session.ChannelMessageSend(m.ChannelID, "uh oh, something went wrong")
		return
	}
	expectPM(pm.ID, p.AskForFriendCode)
	p.session.ChannelMessageSend(pm.ID, message)
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
	player, ok := p.registering[m.Author.ID]
	if !ok {
		return "Well this is awkward. You need to start the registration process over. Sorry!"
	}
	response := p.parseAnswer(pieces)
	if response == AnswerYes {
		// either way they'll need to start over
		delete(p.registering, m.Author.ID)
		success := p.CreatePlayer(player)
		// this is also where you should fix this shitty solution
		if success != "" {
			return success
		}
		return p.RegisterPlayer(&player)
	}
	if response == AnswerNo {
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

func (p *PVP) RegisterPlayer(player *pvp.Player) string {
	resp := p.RegisterOnServer(player)
	if resp != "" {
		return resp
	}

	return p.pmFriendList(player)
}

// TODO returning a string sucks
func (p *PVP) CreatePlayer(player pvp.Player) string {
	b, err := json.Marshal(&player)
	if err != nil {
		log.Printf("error marshalling player json: %s", err.Error())
		return "sorry, something's gone wrong"
	}
	req, err := http.NewRequest(http.MethodPost, pvpURL+"/player", bytes.NewReader(b))
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

	return ""
}

func (p *PVP) RegisterOnServer(player *pvp.Player) string {
	// this is actually sending along all of a player's data when we really just need the server and user id
	b, err := json.Marshal(&player)
	if err != nil {
		log.Printf("error marshalling register json: %s", err.Error())
		return "sorry, something's gone wrong"
	}
	req, err := http.NewRequest(http.MethodPost, pvpURL+"/register", bytes.NewReader(b))
	if err != nil {
		log.Printf("error creating register request: %s", err.Error())
		return "sorry, something's gone wrong"
	}
	req.Header.Add("Content-Type", applicationJSON)
	req.Header.Add("Accept", applicationJSON)

	response, err := client.Do(req)
	if err != nil {
		log.Printf("error performing register request: %s", err.Error())
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

	return ""
}

func (p *PVP) pmFriendList(player *pvp.Player) string {
	players := p.GetPlayers(player.Server)

	var guildName string
	guild, err := p.session.Guild(player.Server)
	if err != nil {
		log.Printf("error getting guild id for %s: %s", player.Server, err.Error())
		guildName = "a server you're in"
	} else {
		guildName = guild.Name
	}
	message := fmt.Sprintf("You're registered for PVP on %s!", guildName)

	if len(players) > 1 {
		message += " Here's who you need to send a friend request to (they've been told it's coming):\n\n"
		for _, opponent := range players {
			if opponent.ID != player.ID {
				message += opponent.ToString() + "\n"

				pm := startPM(p.session, opponent.ID)
				if pm != nil {
					joinPM := fmt.Sprintf("Hey there, %s! You'll be getting a friend request from %s soon, because they just signed up for PVP on %s!", opponent.IGN, player.IGN, guildName)
					p.session.ChannelMessageSend(pm.ID, joinPM)
				}
			}
		}
	}
	pm := startPM(p.session, player.ID)
	if pm != nil {
		p.session.ChannelMessageSend(pm.ID, message)
	}
	return ""
}

func (p *PVP) getUser(id string) *pvp.Player {
	req, err := http.NewRequest(http.MethodGet, pvpURL+"/player?id="+id, nil)
	if err != nil {
		log.Printf("error getting player request: %s", err.Error())
		return nil
	}
	req.Header.Add("Content-Type", applicationJSON)
	req.Header.Add("Accept", applicationJSON)

	response, err := client.Do(req)
	if err != nil {
		log.Printf("error performing player list request: %s", err.Error())
		return nil
	}
	if response.Body != nil {
		defer response.Body.Close()
	}
	var user pvp.Player
	b, err := ioutil.ReadAll(response.Body)

	if len(b) == 0 {
		return nil
	}

	if err != nil {
		log.Printf("error reading player list body: %s", err.Error())
		return nil
	}
	err = json.Unmarshal(b, &user)
	if err != nil {
		log.Printf("error decoding player list body: %s (%s)", err.Error(), string(b))
		return nil
	}
	return &user
}

func (p *PVP) ListPlayers(m *discordgo.MessageCreate) string {
	user := p.getUser(m.Author.ID)
	if user == nil {
		return "you aren't registered!"
	}
	if len(user.Servers) == 0 {
		return "you aren't in any servers!"
	}

	var list string
	for _, server := range user.Servers {
		var guildName string
		guild, err := p.session.Guild(server)
		if err != nil {
			log.Printf("error getting guild name from id: %s", err.Error())
			guildName = "Unknown server"
		} else {
			guildName = guild.Name
		}

		list += fmt.Sprintf("**%s**\n", guildName)
		for _, player := range p.GetPlayers(server) {
			list += player.ToString() + "\n"
		}
		list += "\n"
	}
	return list
}

func (p *PVP) NotUltraForPlayer(user *pvp.Player) map[string]pvp.Player {
	friends := make(map[string]bool)
	toFriend := make(map[string]pvp.Player)
	for _, friend := range p.getFriends(user.ID) {
		friends[friend.IGN] = true
	}

	for _, server := range user.Servers {
		for _, player := range p.GetPlayers(server) {
			if player.ID == user.ID {
				continue
			}
			// are they an ultra friend already?
			if _, isUltra := friends[player.IGN]; !isUltra {
				// do we already know they need to be friends?
				if _, known := toFriend[player.IGN]; !known {
					toFriend[player.IGN] = player
				}
			}
		}
	}
	return toFriend
}

// ConfirmFriendship asks the person being friended if they are in fact ultra. If multiple people ask at the same time it will overwrite all but the most recent. I could do this better, but don't expect it will be an issue for now.
func (p *PVP) ConfirmFriendship(user, friend *pvp.Player) string {
	p.friendship[friend.ID] = user.ID
	log.Println("about to start confirm OM")
	pm := startPM(p.session, friend.ID)
	if pm != nil {
		expectPM(pm.ID, p.AddFriend)
		message := fmt.Sprintf("Hi! %s (%s) says you're ultra friends. Can you confirm this?", user.IGN, user.Username)
		p.session.ChannelMessageSend(pm.ID, message)
		return "Okay, I'll confirm with them that you're ultra friends"
	}
	delete(p.friendship, friend.ID)
	return "Sorry, something went wrong and I can't PM them to confirm"
}

func (p *PVP) AddFriend(pieces []string, m *discordgo.MessageCreate, s *discordgo.Session) string {
	response := p.parseAnswer(pieces)
	if response == AnswerYes {
		id, ok := p.friendship[m.Author.ID]
		if !ok {
			return "This is strange, but I seem to have lost your friendship request. Can you and your friend try agaun? Sorry!"
		}

		// even if it fails they'll need to start again
		delete(p.friendship, m.Author.ID)

		friendship := pvp.Friendship{
			User:   id,
			Friend: m.Author.ID,
		}

		b, err := json.Marshal(&friendship)
		if err != nil {
			log.Printf("error marshalling friendship json: %s", err.Error())
			return "sorry, something's gone wrong"
		}
		req, err := http.NewRequest(http.MethodPost, pvpURL+"/player/friend", bytes.NewReader(b))
		if err != nil {
			log.Printf("error creating friendship request: %s", err.Error())
			return "sorry, something's gone wrong"
		}
		req.Header.Add("Content-Type", applicationJSON)
		req.Header.Add("Accept", applicationJSON)

		response, err := client.Do(req)
		if err != nil {
			log.Printf("error performing friendship request: %s", err.Error())
			return "sorry, something's gone wrong"
		}
		if response.Body != nil {
			defer response.Body.Close()
		}

		if response.StatusCode == http.StatusConflict {
			return "You two seem to be friends already. This is weird."
		}

		if response.StatusCode != http.StatusCreated {
			log.Printf("error registering friendship: got %v\n", response.StatusCode)
			return "uh oh, something went wrong"
		}

		// if there's an error PMing it's not the end of the world
		log.Println("about to start has confirmed OM")
		pm := startPM(s, id)
		if pm != nil {
			user := p.getUser(m.Author.ID)
			message := fmt.Sprintf("Hi! %s (%s) has confirmed your friendship", user.IGN, user.Username)
			p.session.ChannelMessageSend(pm.ID, message)
		}

		// TODO check both for "core member" status (#8)
		return "Thanks for confirming!"
	}

	if response == AnswerNo || response == AnswerCancel {
		id, ok := p.friendship[m.Author.ID]
		if ok {
			log.Println("about to start mo OM")
			pm := startPM(s, id)
			if pm != nil {
				user := p.getUser(m.Author.ID)
				message := fmt.Sprintf("Hi! %s (%s) says you're aren't actually ultra friends. Please confer with them and try again.", user.IGN, user.Username)
				p.session.ChannelMessageSend(pm.ID, message)
			}
		}
		delete(p.friendship, m.Author.ID)
		return "Sorry for bothering you! I've let them know."
	}

	expectPM(m.ChannelID, p.AddFriend)
	return fmt.Sprintf("Sorry, I don't understand the answer '%s'. Please say 'yes' or 'no'.", strings.Join(pieces, " "))
}

func (p *PVP) getFriends(ID string) []pvp.Player {
	var friends []pvp.Player
	req, err := http.NewRequest(http.MethodGet, pvpURL+"/player/friend?id="+ID, nil)
	if err != nil {
		log.Printf("error getting player friend request: %s", err.Error())
		return friends
	}
	req.Header.Add("Content-Type", applicationJSON)
	req.Header.Add("Accept", applicationJSON)

	response, err := client.Do(req)
	if err != nil {
		log.Printf("error performing friend list request: %s", err.Error())
		return friends
	}
	if response.Body != nil {
		defer response.Body.Close()
	}
	b, err := ioutil.ReadAll(response.Body)

	if len(b) == 0 {
		return friends
	}

	if err != nil {
		log.Printf("error reading friend list body: %s", err.Error())
		return friends
	}
	err = json.Unmarshal(b, &friends)
	if err != nil {
		log.Printf("error decoding player list body: %s (%s)", err.Error(), string(b))
	}
	return friends
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
