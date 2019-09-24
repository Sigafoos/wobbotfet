package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sigafoos/wobbotfet/bot"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN not found in environment variables")
	}
	auth := "Bot " + token

	wob := bot.New(auth)
	wob.Start()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// quitting time. clean up after ourselves
	wob.Close()
}
