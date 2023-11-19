package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Nelwhix/scorecard-bot/entity"
	"github.com/Nelwhix/scorecard-bot/utils"
	"github.com/olekukonko/tablewriter"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"strconv"
	"time"
)

func StartGameSession(db *sql.DB, client *whatsmeow.Client) error {
	stmt := `INSERT INTO games(created_at, updated_at) VALUES(?, ?)`
	_, err := db.Exec(stmt, time.Now(), time.Now())
	if err != nil {
		return errors.Join(errors.New("error inserting new row"), err)
	}

	msg := &waProto.Message{Conversation: proto.String("Started a new games night session...")}
	recipient := types.NewJID("120363194522631267", types.GroupServer)
	_, err = client.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		return errors.Join(errors.New("error sending message"), err)
	}

	return nil
}

func AwardScore(db *sql.DB, message string, client *whatsmeow.Client) error {
	// get the latest game id
	rows, err := db.Query("SELECT id FROM games ORDER BY created_at DESC LIMIT 1;")
	if err != nil {
		return errors.Join(errors.New("error fetching latest game"), err)
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
			return errors.Join(errors.New("error fetching latest game"), err)
		}
	}
	// check if there is a leaderboard record with that game_id and the user's phone
	phoneNumber := utils.GetPhoneNumber(message)
	newScore := utils.GetAwardedScore(message)
	stmt, err := db.Prepare("SELECT id,game_id,phone_number,score,created_at,updated_at FROM leaderboards WHERE game_id = ? AND phone_number = ?")
	if err != nil {
		return errors.Join(errors.New("error preparing statement"), err)
	}

	var leaderboard entity.Leaderboards
	err = stmt.QueryRow(gameId, phoneNumber).Scan(&leaderboard.Id, &leaderboard.GameId, &leaderboard.PhoneNumber, &leaderboard.Score, &leaderboard.CreatedAt, &leaderboard.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			stmt := `INSERT INTO leaderboards(game_id, phone_number,score,created_at,updated_at) VALUES(?, ?, ?, ?, ?)`
			_, err = db.Exec(stmt, gameId, phoneNumber, newScore, time.Now(), time.Now())
			if err != nil {
				return errors.Join(errors.New("error inserting leaderboard record"), err)
			}
		} else {
			return errors.Join(errors.New("error fetching leaderboard record"), err)
		}

	} else {
		initialScore, _ := strconv.Atoi(leaderboard.Score)
		updatedScore := initialScore + newScore
		_, err = db.Exec("UPDATE leaderboards SET score=?, updated_at=? WHERE id=?", updatedScore, time.Now(), leaderboard.Id)
		if err != nil {
			return errors.Join(errors.New("error updating leaderboard record"), err)
		}
	}

	msg := &waProto.Message{Conversation: proto.String(fmt.Sprintf("Awarded %v points to %v", newScore, phoneNumber))}
	recipient := types.NewJID("120363194522631267", types.GroupServer)
	_, err = client.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		return errors.Join(errors.New("error sending message"), err)
	}

	// send the new leaderboard
	stmt, err = db.Prepare("SELECT id, game_id, phone_number, score, created_at, updated_at FROM leaderboards WHERE game_id = ? ORDER BY score DESC;")
	if err != nil {
		return errors.Join(errors.New("error setting up prepared statement"), err)
	}

	rows, err = stmt.Query(gameId)
	if err != nil {
		return errors.Join(errors.New("error making query"), err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Println("error closing row")
			return
		}
	}(rows)

	var leaderboards []entity.Leaderboards

	for rows.Next() {
		var leaderboard entity.Leaderboards
		if err := rows.Scan(&leaderboard.Id, &leaderboard.GameId, &leaderboard.PhoneNumber, &leaderboard.Score, &leaderboard.CreatedAt, &leaderboard.UpdatedAt); err != nil {
			fmt.Println("Error scanning row:", err)
			return errors.Join(errors.New("error scanning leaderboard row"), err)
		}
		leaderboards = append(leaderboards, leaderboard)
	}

	if err := rows.Err(); err != nil {
		return errors.Join(errors.New("error iterating over rows"), err)
	}

	// Now 'leaderboards' contains the fetched rows' data
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"S/N", "Phone", "Score"})

	serialNum := 1
	for _, row := range leaderboards {
		value := []string{strconv.Itoa(serialNum), row.PhoneNumber, row.Score}
		table.Append(value)
		serialNum++
	}
	table.Render()

	msg = &waProto.Message{Conversation: proto.String(buf.String())}
	recipient = types.NewJID("120363194522631267", types.GroupServer)
	_, err = client.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		return errors.Join(errors.New("error sending message"), err)
	}

	return nil
}
