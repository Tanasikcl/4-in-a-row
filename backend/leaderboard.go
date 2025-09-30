package main

import "context"

type LBRow struct { Username string `json:"username"`; Wins int `json:"wins"` }

func (db *DB) QueryLeaderboard(ctx context.Context) ([]LBRow, error) {
	rows, err := db.Pool.Query(ctx, `SELECT winner AS username, COUNT(*) AS wins
		FROM games WHERE winner IS NOT NULL GROUP BY winner ORDER BY wins DESC LIMIT 50`)
	if err != nil { return nil, err }
	defer rows.Close()
	out := []LBRow{}
	for rows.Next() {
		var r LBRow; _ = rows.Scan(&r.Username, &r.Wins); out = append(out, r)
	}
	return out, nil
}
