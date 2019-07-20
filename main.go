package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/codefordenver/scout/global"
	"github.com/codefordenver/scout/pkg/discord"
	"github.com/codefordenver/scout/pkg/gdrive"
	"github.com/codefordenver/scout/pkg/github"
	"gopkg.in/yaml.v2"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func init() {
	global.LocationString = os.Getenv("SCOUT_LOCATION_STRING")
}

func main() {

	encodedConfig := os.Getenv("SCOUT_CONFIG")
	if encodedConfig == "" {
		log.Fatal("configuration environment variable is empty or does not exist")
	}
	config, err := base64.StdEncoding.DecodeString(encodedConfig)
	if err != nil {
		log.Fatal("error decoding configuration string", err)
		return
	}

	if err = yaml.Unmarshal(config, &global.Brigades); err != nil {
		log.Fatal("error parsing configuration file,", err)
		return
	}
	
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
