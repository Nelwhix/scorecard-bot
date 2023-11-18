package main

import "time"

type Leaderboards struct {
	id          int
	gameId      int
	phoneNumber string
	score       string
	createdAt   time.Time
	updatedAt   time.Time
}
