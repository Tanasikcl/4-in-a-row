package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct{ Pool *pgxpool.Pool }


func MustOpenDB(dsn string) *DB {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil { panic(err) }
	return &DB{Pool: pool}
}


func (db *DB) AutoMigrate() {
	ctx := context.Background()
	_, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS games (
			id         UUID PRIMARY KEY,
			p1         TEXT NOT NULL,
			p2         TEXT NOT NULL,
			winner     TEXT,
			is_draw    BOOLEAN NOT NULL DEFAULT FALSE,
			started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			ended_at   TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS idx_games_ended_at ON games(ended_at);
	`)
	if err != nil { log.Println("migrate err:", err) }
}


func (a *App) PersistGame(g *Game) {
	ctx := context.Background()
	_, err := a.DB.Pool.Exec(ctx, `
		INSERT INTO games (id, p1, p2, winner, is_draw, started_at, ended_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			winner     = EXCLUDED.winner,
			is_draw    = EXCLUDED.is_draw,
			started_at = EXCLUDED.started_at,
			ended_at   = EXCLUDED.ended_at
	`, g.ID, g.P1, g.P2, g.Winner, g.IsDraw, g.Started, g.Ended)
	if err != nil { log.Println("persist game err:", err) }
}



type RecentGameRow struct {
	ID      string     `json:"id"`
	P1      string     `json:"p1"`
	P2      string     `json:"p2"`
	Winner  *string    `json:"winner,omitempty"`
	IsDraw  bool       `json:"is_draw"`
	Started time.Time  `json:"started"`
	Ended   *time.Time `json:"ended,omitempty"`
}


func (db *DB) QueryRecentGames(ctx context.Context, limit int) ([]RecentGameRow, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, p1, p2, winner, is_draw, started_at, ended_at
		FROM games
		ORDER BY ended_at DESC NULLS LAST, started_at DESC
		LIMIT $1
	`, limit)
	if err != nil { return nil, err }
	defer rows.Close()

	out := []RecentGameRow{}
	for rows.Next() {
		var r RecentGameRow
		if err := rows.Scan(&r.ID, &r.P1, &r.P2, &r.Winner, &r.IsDraw, &r.Started, &r.Ended); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if rows.Err() != nil { return nil, rows.Err() }
	return out, nil
}
