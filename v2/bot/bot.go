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
	version   string
	mentionre = regexp.MustCompile(`<@\d+>`)
)

var (
	access     *log.Logger
	aLog, eLog *os.File
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

var commands commandMap

var help []string

func init() {
	FloorMap = make(map[string]string)
	FloorMap["raid"] = FloorHatched
	FloorMap["hatch"] = FloorHatched
	FloorMap["hatched"] = FloorHatched
	FloorMap["research"] = FloorHatched

	commands = make(map[string]command)
}

func registerCommand(key string, f command, helpText string) {
	commands[key] = f
	help = append(help, fmt.Sprintf("**%s**: %s", key, helpText))
}

type Query struct {
	League  string
	Pokemon string
	Atk     string
	Def     string
	HP      string
	Floor   string
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
	message = strings.ToLower(message)
	pieces := strings.Split(message, " ")

	var response string
	f, ok := commands[pieces[0]]
	if !ok {
		response = fmt.Sprintf("I don't have a `%s` command", pieces[0])
	} else {
		response = f(pieces[1:], m, s)
	}

	if m.GuildID != "" {
		response = m.Author.Mention() + ": " + response
	}
	s.ChannelMessageSend(m.ChannelID, response)
}
