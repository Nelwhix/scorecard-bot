package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Nelwhix/scorecard-bot/handlers"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	client *whatsmeow.Client
)

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if v.Info.IsGroup && v.Info.Sender.User == "2349016607485" {
			db, err := sql.Open("sqlite3", "app.db")
			if err != nil {
				fmt.Println("Error opening database file", err)
				return
			}
			defer func(db *sql.DB) {
				err = db.Close()
				if err != nil {
					fmt.Println("Error closing database file", err)
					return
				}
			}(db)

			message := *v.Message.GetExtendedTextMessage().Text
			switch {
			case strings.Contains(v.Message.GetConversation(), "chloe \\start"):
				err = handlers.StartGameSession(db, client)
				if err != nil {
					fmt.Println("error starting game session", err)
					return
				}
			case strings.Contains(message, "chloe \\add"):
				err = handlers.AwardScore(db, message, client)
				if err != nil {
					fmt.Println("error awarding score", err)
					return
				}
			}
		}
	}
}

func main() {
	container, err := sqlstore.New("sqlite3", "file:app.db?_foreign_keys=on", nil)
	if err != nil {
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	client = whatsmeow.NewClient(deviceStore, nil)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}
