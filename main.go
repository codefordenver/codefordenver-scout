package main

import (
	"context"
	"fmt"
	"github.com/codefordenver/codefordenver-scout/global"
	"github.com/codefordenver/codefordenver-scout/pkg/discord"
	"github.com/codefordenver/codefordenver-scout/pkg/gdrive"
	"github.com/codefordenver/codefordenver-scout/pkg/github"
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
	global.AirtableKey = os.Getenv("AIRTABLE_API_KEY")
}

func main() {
	config, err := decrypt.File("config.yaml", "yaml")
	if err != nil {
		log.Fatal("error decoding configuration, ", err)
	}

	var b Brigades

	if err = yaml.Unmarshal(config, &b); err != nil {
		log.Fatal("error parsing configuration file,", err)
	}

	global.Brigades = b.Brigades

	err = gdrive.Create()
	if err != nil {
		return
	}

	dg, err := discord.Create()
	if err != nil {
		return
	}

	err = github.Create(dg)
	if err != nil {
		return
	}

	dg.AddHandler(discord.MessageCreate)
	dg.AddHandler(discord.UserJoin)
	dg.AddHandler(discord.ConnectToGuild)
	dg.AddHandler(discord.UserReact)

	err = dg.Open()
	if err != nil {
		log.Fatal("error opening connection,", err)
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
			log.Fatal("error starting GitHub webhook,", err)
		}
	}()

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	if err = dg.Close(); err != nil {
		fmt.Println("error closing Discord session, ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println("error shutting down github webhook,", err)
	} else {
		cancel()
	}
}
