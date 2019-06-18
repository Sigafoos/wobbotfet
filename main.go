package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sigafoos/wobbotfet/bot"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/viper"
)

var sigafoos = discordgo.User{Username: "Sigafoos#6538", ID: "193777776543662081"}

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	var env, auth string
	if len(os.Args) < 2 {
		env = "prod"
	} else {
		env = os.Args[1]
	}
	token := viper.GetString(fmt.Sprintf("token.%s", env))
	if token == "" {
		log.Fatalf("no token found for environment '%s'", env)
	}
	auth = "Bot " + token

	wob := bot.New(auth, &sigafoos)
	wob.Start()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// quitting time. clean up after ourselves
	wob.Close()
}
