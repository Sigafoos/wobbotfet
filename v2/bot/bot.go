package bot

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	client    = &http.Client{}
	mentionre = regexp.MustCompile(`<@!?\d+>`)
)

var (
	access *log.Logger
	aLog   *os.File
)

const (
	FloorHatched = "hatched"
)

var FloorMap map[string]string

type Bot struct {
	owner   *discordgo.User
	pm      *discordgo.Channel
	session *discordgo.Session
}

type command func([]string, *discordgo.MessageCreate, *discordgo.Session) string
type commandMap map[string]command

var (
	commands  commandMap
	activePMs map[string]command
)

var help []string

func init() {
	FloorMap = make(map[string]string)
	FloorMap["raid"] = FloorHatched
	FloorMap["hatch"] = FloorHatched
	FloorMap["hatched"] = FloorHatched
	FloorMap["research"] = FloorHatched

	commands = make(map[string]command)
	activePMs = make(map[string]command)
}

func registerCommand(key string, f command, helpText string) {
	commands[key] = f
	help = append(help, fmt.Sprintf("**%s**: %s", key, helpText))
}

// say "hey I'm expecting a PM from this user about something"
func expectPM(pm string, next command) {
	activePMs[pm] = next
}

func startPM(s *discordgo.Session, user string) *discordgo.Channel {
	pm, err := s.UserChannelCreate(user)
	if err != nil {
		log.Printf("error starting PM: %s", err.Error())
		return nil
	}
	return pm
}

type Query struct {
	League  string
	Pokemon string
	Atk     string
	Def     string
	HP      string
	Floor   string
}

func New(auth string) *Bot {
	var err error
	aLog, err = os.OpenFile("access.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("cannot open access log for writing: %s\n", err)
	}
	access = log.New(aLog, "", log.Ldate|log.Ltime)

	session, err := discordgo.New(auth)
	if err != nil {
		log.Fatal(err)
	}

	b := &Bot{session: session}
	session.AddHandler(b.readMessage)

	if owner := os.Getenv("DISCORD_OWNER"); owner != "" {
		b.owner = &discordgo.User{ID: owner}
	} else {
		log.Println("no DISCORD_OWNER specified; will not PM owner")
	}

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
	if os.Getenv("VERSION") != "" {
		b.session.UpdateStatus(0, os.Getenv("VERSION"))
	}
	cmds := "known commands:\n"
	for k := range commands {
		cmds += "- " + k + "\n"
	}
	b.PM(cmds)
	fmt.Println("wooooooooobbotfett!")
}

func (b *Bot) Close() {
	b.PM("going down")
	b.session.Close()

	err := aLog.Close()
	if err != nil {
		log.Printf("error closing access log: %s\n", err)
	}
}

func (b *Bot) PM(message string) {
	if b.pm == nil {
		return
	}
	b.session.ChannelMessageSend(b.pm.ID, message)
}

// Servers returns a list of server ids the bot is connected to. It will use its session to fill in the server details.
func (b *Bot) Servers() ([]*discordgo.UserGuild, error) {
	servers, err := b.session.UserGuilds(100, "", "")
	return servers, err
}

// ActivePMs returns a list of open PM channels.
func (b *Bot) ActivePMs() []string {
	pms := make([]string, len(activePMs))
	i := 0
	for k := range activePMs {
		pms[i] = k
		i++
	}
	return pms
}

// Roles returns the roles for a server.
func (b *Bot) Roles(server string) ([]*discordgo.Role, error) {
	return b.session.GuildRoles(server)
}

func (b *Bot) readMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	defer func() {
		if r := recover(); r != nil {
			b.PM(fmt.Sprintf("recovered from panic: %v\n\n`%v`", r, string(debug.Stack())))
		}
	}()

	// ignore messages posted by wobbotfet
	if m.Author.ID == s.State.User.ID {
		return
	}

	// only check for a mention if it's not a PM
	if m.GuildID != "" {
		mentioned := false
		for _, u := range m.Mentions {
			if u.ID == s.State.User.ID {
				mentioned = true
				break
			}
		}

		if !mentioned {
			return
		}
	}

	err := s.ChannelTyping(m.ChannelID)
	if err != nil {
		log.Printf("error sending typing call: %s", err.Error())
	}

	message := mentionre.ReplaceAllString(m.Content, "")
	message = strings.Replace(message, "  ", " ", -1)
	message = strings.TrimSpace(message)
	pieces := strings.Split(strings.ToLower(message), " ")

	var response string
	if pm, ok := activePMs[m.ChannelID]; ok {
		delete(activePMs, m.ChannelID)
		// we don't want to lowercase PM responses
		response = pm(strings.Split(message, " "), m, s)
	} else if f, ok := commands[pieces[0]]; !ok {
		response = fmt.Sprintf("I don't have a `%s` command", pieces[0])
	} else {
		response = f(pieces[1:], m, s)
	}

	messages := b.splitResponse(response)

	for _, message := range messages {
		if m.GuildID != "" {
			message = m.Author.Mention() + ": " + message
		}
		if _, err := s.ChannelMessageSend(m.ChannelID, message); err != nil {
			log.Printf("error sending message: %s", err)
			s.ChannelMessageSend(m.ChannelID, "sorry, something's gone wrong")
			return
		}
	}

}

// discord has a 2000 character limit. if it's longer than that, find a good way to split it into multiple messages.
func (b *Bot) splitResponse(response string) []string {
	if len(response) <= 2000 {
		return []string{response}
	}

	var messages []string
	splitResponse := strings.Split(response, "\n")

	// ain't gonna lie, haven't tested this bit
	if len(splitResponse) == 1 {
		log.Printf("%v character message has no line breaks", len(response))
		for i := 0; i <= len(response)/2000; i++ {
			start := 2000 * i
			end := 2000 * (i + 1)
			if end > len(response) {
				end = len(response)
				messages = append(messages, response[start:end])
			}
		}
		return messages
	}

	// send the fewest possible messages, split by line breaks
	var current string
	for i, line := range splitResponse {
		if len(current)+len(line) > 2000 {
			messages = append(messages, current)
			current = ""
		}
		current += line + "\n"

		if i == len(splitResponse)-1 {
			messages = append(messages, current)
		}
	}
	return messages
}
