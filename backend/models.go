package main

import (
	"time"
	"github.com/google/uuid"
)

type Player struct {
	Username string
	Conn     *WSConn // wrapper to send safely
	LastSeen time.Time
}

type GameID = uuid.UUID

const (
	Cols = 7
	Rows = 6
)

type Cell int8
const (
	Empty Cell = 0
	P1    Cell = 1
	P2    Cell = 2
)

type Game struct {
	ID      GameID
	P1      string
	P2      string // can be "BOT"
	Board   [Rows][Cols]Cell
	Turn    Cell // P1 or P2
	Started time.Time
	Ended   *time.Time
	Winner  *string
	IsDraw  bool
}

// WebSocket message payloads

type WSMessage struct {
	Type string      `json:"type"` // join|state|move|error|end|ping|leaderboard
	Data interface{} `json:"data"`
}

type Move struct {
	Col int `json:"col"`
}

type StatePayload struct {
	GameID   string          `json:"gameId"`
	Board    [Rows][Cols]Cell `json:"board"`
	Turn     Cell            `json:"turn"`
	You      Cell            `json:"you"`
	Opponent string          `json:"opponent"`
}
