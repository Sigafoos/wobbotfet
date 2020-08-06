package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sigafoos/wobbotfet/api"
	"github.com/Sigafoos/wobbotfet/bot"

	"github.com/gorilla/mux"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN not found in environment variables")
	}
	auth := "Bot " + token

	wob := bot.New(auth)
	wob.Start()

	// DO NOT ENABLE THE API SERVER IF YOU ARE ON A PUBLIC NETWORK
	host := os.Getenv("WOB_HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("WOB_PORT")
	if port == "" {
		port = "8081"
	}
	r := mux.NewRouter()
	s := &http.Server{
		Addr:    net.JoinHostPort(host, port),
		Handler: r,
	}
	if os.Getenv("WOB_API") == "1" {
		a := api.New(wob)
		r.HandleFunc("/servers", a.GetServers).Methods(http.MethodGet)
		r.HandleFunc("/pms", a.GetActivePMs).Methods(http.MethodGet)

		s.ListenAndServe()
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// quitting time. clean up after ourselves
	wob.Close()
	s.Shutdown(context.Background())
}
