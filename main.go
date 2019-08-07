package main

import (
	"context"
	"fmt"
	"github.com/codefordenver/scout/global"
	"github.com/codefordenver/scout/pkg/discord"
	"github.com/codefordenver/scout/pkg/gdrive"
	"github.com/codefordenver/scout/pkg/github"
	"go.mozilla.org/sops/decrypt"
	"gopkg.in/yaml.v2"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Brigades struct {
	Brigades []global.Brigade `yaml:"Brigades"`
}

func init() {
	global.LocationString = os.Getenv("SCOUT_LOCATION_STRING")
}

func main() {
	config, err := decrypt.File("config.yaml", "yaml")
	if err != nil {
		log.Fatal("error decoding configuration, ", err)
		return
	}

	var b Brigades

	if err = yaml.Unmarshal(config, &b); err != nil {
		log.Fatal("error parsing configuration file,", err)
		return
	}

	global.Brigades = b.Brigades

	global.DriveClient, err = gdrive.Create()
	if err != nil {
		log.Fatal("error creating Google Drive session, ", err)
		return
	}

	global.GithubClient, err = github.Create()

	global.DiscordClient, err = discord.Create()
	if err != nil {
		log.Fatal("error creating Discord session,", err)
		return
	}

	global.DiscordClient.AddHandler(discord.MessageCreate)
	global.DiscordClient.AddHandler(discord.UserJoin)
	global.DiscordClient.AddHandler(discord.ConnectToGuild)
	global.DiscordClient.AddHandler(discord.UserReact)

	err = global.DiscordClient.Open()
	if err != nil {
		log.Fatal("error opening connection,", err)
		return
	}

	port := os.Getenv("PORT")
	var server *http.Server
	if port == "" {
		server = &http.Server{Addr: ":3000", Handler: http.HandlerFunc(github.HandleRepositoryEvent)}
	} else {
		server = &http.Server{Addr: ":" + port, Handler: http.HandlerFunc(github.HandleRepositoryEvent)}
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal("error starting github webhook,", err)
		}
	}()

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	if err = global.DiscordClient.Close(); err != nil {
		fmt.Println("error closing Discord session, ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println("error shutting down github webhook,", err)
	} else {
		cancel()
	}
}
