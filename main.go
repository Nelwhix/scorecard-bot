package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	client *whatsmeow.Client
)

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		//120363194522631267
		if v.Info.IsGroup && v.Info.Sender.User == "2349016607485" {
			switch {
			case strings.Contains(v.Message.GetConversation(), "chloe \\start"):
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

				stmt := `INSERT INTO games(created_at, updated_at) VALUES(?, ?)`
				_, err = db.Exec(stmt, time.Now(), time.Now())
				if err != nil {
					fmt.Println(err)
					return
				}

				msg := &waProto.Message{Conversation: proto.String("Started a new games night session...")}
				recipient := types.NewJID("120363194522631267", types.GroupServer)
				_, err = client.SendMessage(context.Background(), recipient, msg)
				if err != nil {
					fmt.Println("Error sending message", err)
				}
			case strings.Contains(v.Message.GetConversation(), "chloe \\add"):
				// get the latest game id
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

				rows, err := db.Query("SELECT id FROM games ORDER BY created_at DESC LIMIT 1;")
				if err != nil {
					fmt.Println("error fetching row", err)
				}
				defer func(rows *sql.Rows) {
					err = rows.Close()
					if err != nil {
						fmt.Println("error closing row", err)
					}
				}(rows)

				var gameId int
				for rows.Next() {
					err = rows.Scan(&gameId)
					if err != nil {
						fmt.Println("error scanning row", err)
					}
				}
				// check if there is a leaderboard record with that game_id and the user's phone
				phoneNumber := getPhoneNumber(v.Message.GetConversation())
				newScore := getAwardedScore(v.Message.GetConversation())
				stmt, err := db.Prepare("SELECT id,game_id,phone_number,score,created_at,updated_at FROM leaderboards WHERE game_id = ? AND phone_number = ?")
				if err != nil {
					fmt.Println("error setting up prep statement", err)
				}

				var leaderboard *Leaderboards
				err = stmt.QueryRow(gameId, phoneNumber).Scan(&leaderboard.id, &leaderboard.gameId, &leaderboard.phoneNumber, &leaderboard.score, &leaderboard.createdAt, &leaderboard.updatedAt)
				if err != nil {
					fmt.Println("error fetching leaderboard record", err)
					return
				}

				// record exists
				if leaderboard.id != 0 {
					leaderboard.score += newScore
					_, err = db.Exec("UPDATE leaderboards SET score=?, SET updated_at=? WHERE id=?", leaderboard.score, time.Now(), leaderboard.id)
					if err != nil {
						fmt.Println("error updating leaderboard record", err)
						return
					}
				} else {
					stmt := `INSERT INTO leaderboards(game_id, phone_number,score,created_at,updated_at) VALUES(?, ?, ?, ?, ?)`
					_, err = db.Exec(stmt, gameId, phoneNumber, newScore, time.Now(), time.Now())
					if err != nil {
						fmt.Println(err)
						return
					}
				}

				msg := &waProto.Message{Conversation: proto.String(fmt.Sprintf("Awarded %v points to %v", newScore, phoneNumber))}
				recipient := types.NewJID("120363194522631267", types.GroupServer)
				_, err = client.SendMessage(context.Background(), recipient, msg)
				if err != nil {
					fmt.Println("Error sending message", err)
				}

				// send the new leaderboard
			}
		}
	}
}

func main() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:app.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)
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
