package main

import (
	"context"
	"fmt"
	"github.com/codefordenver/codefordenver-scout/migrations"
	"github.com/codefordenver/codefordenver-scout/pkg/discord"
	"github.com/codefordenver/codefordenver-scout/pkg/gdrive"
	"github.com/codefordenver/codefordenver-scout/pkg/github"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	db, err := gorm.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=disable", os.Getenv("SCOUT_DB_HOST"), os.Getenv("SCOUT_DB_PORT") , os.Getenv("SCOUT_DB_USER"), os.Getenv("SCOUT_DB_NAME"), os.Getenv("SCOUT_DB_PASSWORD")))
	if err != nil {
		return
	}
	migrations.Migrate(db)
	err = gdrive.New(db)
	if err != nil {
		return
	}

	dg, err := discord.New(db)
	if err != nil {
		return
	}

	err = github.New(db, dg)
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
