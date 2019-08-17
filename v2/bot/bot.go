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

type Bot struct {
	owner   *discordgo.User
	pm      *discordgo.Channel
	session *discordgo.Session
}

type command func([]string, *discordgo.MessageCreate) string
type commandMap map[string]command

var commands commandMap

func init() {
	commands = make(map[string]command)
}

func registerCommand(key string, f command) {
	commands[key] = f
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
		response = f(pieces[1:], m)
	}

	if m.GuildID != "" {
		response = m.Author.Mention() + ": " + response
	}
	s.ChannelMessageSend(m.ChannelID, response)

	/*




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
	*/
}
