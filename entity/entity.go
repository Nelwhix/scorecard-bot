package entity

import "time"

type Leaderboards struct {
	Id          int
	GameId      int
	PhoneNumber string
	Score       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
