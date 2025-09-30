package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type App struct {
	Hub       *Hub
	DB        *DB
	Analytics *Analytics
}


func (a *App) HandleMove(username string, col int) {
	
	a.Hub.mu.Lock()
	gid, ok := a.Hub.playersInGame[username]
	if !ok {
		a.Hub.mu.Unlock()
		return
	}
	g := a.Hub.games[gid]
	side := a.Hub.sideOf(g, username)
	
	if (g.Turn == P1 && side != P1) || (g.Turn == P2 && side != P2) {
		a.Hub.mu.Unlock()
		return
	}
	row, err := g.drop(col)
	if err != nil {
		a.Hub.mu.Unlock()
		return
	}
	a.Hub.mu.Unlock()

	a.Analytics.Emit("move", map[string]any{"gameId": g.ID.String(), "by": username, "col": col, "row": row})

	
	a.Hub.mu.Lock()
	if g.checkWinAt(row, col) {
		now := time.Now()
		g.Ended = &now
		g.Winner = &username
		a.Hub.mu.Unlock()

		
		a.BroadcastState(g)

		go a.PersistGame(g)
		a.BroadcastEnd(g, "win", username)
		a.Hub.RemoveGame(g.ID)
		return
	}
	if g.isFull() {
		now := time.Now()
		g.Ended = &now
		g.IsDraw = true
		a.Hub.mu.Unlock()

		
		a.BroadcastState(g)

		go a.PersistGame(g)
		a.BroadcastEnd(g, "draw", "")
		a.Hub.RemoveGame(g.ID)
		return
	}

	
	g.switchTurn()
	a.Hub.mu.Unlock()
	a.BroadcastState(g)

	
	a.Hub.mu.RLock()
	botTurn := (g.Turn == P1 && g.P1 == "BOT") || (g.Turn == P2 && g.P2 == "BOT")
	a.Hub.mu.RUnlock()
	if botTurn {
		a.botMove(g)
	}
}

func (a *App) botMove(g *Game) {
	a.Hub.mu.Lock()
	botSide := g.Turn
	col := chooseBotMove(g, botSide)
	row, _ := g.drop(col)
	a.Hub.mu.Unlock()

	a.Analytics.Emit("move", map[string]any{"gameId": g.ID.String(), "by": "BOT", "col": col, "row": row})

	a.Hub.mu.Lock()
	if g.checkWinAt(row, col) {
		now := time.Now()
		g.Ended = &now
		w := "BOT"
		g.Winner = &w
		a.Hub.mu.Unlock()

		a.BroadcastState(g)

		go a.PersistGame(g)
		a.BroadcastEnd(g, "win", "BOT")
		a.Hub.RemoveGame(g.ID)
		return
	}
	if g.isFull() {
		now := time.Now()
		g.Ended = &now
		g.IsDraw = true
		a.Hub.mu.Unlock()

		a.BroadcastState(g)

		go a.PersistGame(g)
		a.BroadcastEnd(g, "draw", "")
		a.Hub.RemoveGame(g.ID)
		return
	}
	g.switchTurn()
	a.Hub.mu.Unlock()
	a.BroadcastState(g)
}


func (a *App) HandleRegame(username, mode string) {
	p := &Player{Username: username, Conn: a.Hub.Conn(username), LastSeen: time.Now()}

	if mode == "bot" {
		// immediate bot match
		g := newGame(username, "BOT")
		a.Hub.mu.Lock()
		a.Hub.games[g.ID] = g
		a.Hub.playersInGame[g.P1] = g.ID
		a.Hub.playersInGame[g.P2] = g.ID
		a.Hub.mu.Unlock()

		a.Analytics.Emit("match.start", map[string]any{"gameId": g.ID.String(), "p1": g.P1, "p2": g.P2})
		a.pushStateToUser(g, username)
		a.BroadcastState(g)
		return
	}

	g, side, rejoined := a.Hub.EnqueueOrMatch(p)
	if g == nil {
		go a.Hub.StartBotIfStillWaiting(func(p *Player) {
			g := newGame(p.Username, "BOT")
			a.Hub.mu.Lock()
			a.Hub.games[g.ID] = g
			a.Hub.playersInGame[g.P1] = g.ID
			a.Hub.playersInGame[g.P2] = g.ID
			a.Hub.mu.Unlock()

			a.Analytics.Emit("match.start", map[string]any{"gameId": g.ID.String(), "p1": g.P1, "p2": g.P2})
			a.pushStateToUser(g, p.Username)
			a.BroadcastState(g)
		})
		if ws := a.Hub.Conn(username); ws != nil {
			_ = ws.SafeWriteJSON(WSMessage{Type: "state", Data: map[string]any{"waiting": true}})
		}
		return
	}

	you := side
	opponent := g.P1
	if you == P1 {
		opponent = g.P2
	}
	if ws := a.Hub.Conn(username); ws != nil {
		_ = ws.SafeWriteJSON(WSMessage{
			Type: "state",
			Data: StatePayload{GameID: g.ID.String(), Board: g.Board, Turn: g.Turn, You: you, Opponent: opponent},
		})
	}
	if !rejoined {
		a.Analytics.Emit("match.paired", map[string]any{"gameId": g.ID.String(), "p1": g.P1, "p2": g.P2})
	}
}


func (a *App) BroadcastState(g *Game) {
	for _, u := range []string{g.P1, g.P2} {
		if u == "BOT" {
			continue
		}
		you := P1
		opp := g.P2
		if u == g.P2 {
			you = P2
			opp = g.P1
		}
		ws := a.Hub.Conn(u)
		if ws != nil {
			log.Printf("Sending game state to player %s: you=%d, opponent=%s, turn=%d", u, you, opp, g.Turn)
			ws.SafeWriteJSON(WSMessage{Type: "state", Data: StatePayload{
				GameID: g.ID.String(), Board: g.Board, Turn: g.Turn, You: you, Opponent: opp,
			}})
		} else {
			log.Printf("No WebSocket connection found for player: %s", u)
		}
	}
}

func (a *App) BroadcastEnd(g *Game, reason, winner string) {
	
	a.Analytics.Emit("game.end", map[string]any{
		"gameId":   g.ID.String(),
		"winner":   winner,
		"reason":   reason,
		"duration": time.Since(g.Started),
		"p1":       g.P1,
		"p2":       g.P2,
	})
	for _, u := range []string{g.P1, g.P2} {
		if u == "BOT" {
			continue
		}
		if ws := a.Hub.Conn(u); ws != nil {
			ws.SafeWriteJSON(WSMessage{Type: "end", Data: map[string]any{"reason": reason, "winner": winner}})
		}
	}
}

func (a *App) ForfeitIfNotRejoined(username string) {
	ttl := a.Hub.reconnectTTL
	time.Sleep(ttl)

	if a.Hub.Conn(username) != nil {
		return
	}

	a.Hub.mu.Lock()
	gid, ok := a.Hub.playersInGame[username]
	if !ok {
		a.Hub.mu.Unlock()
		return
	}
	g := a.Hub.games[gid]
	if g == nil || g.Ended != nil {
		a.Hub.mu.Unlock()
		return
	}


	var winner string
	if g.P1 == username {
		winner = g.P2
	} else {
		winner = g.P1
	}
	now := time.Now()
	g.Ended = &now
	g.Winner = &winner
	a.Hub.mu.Unlock()

	a.BroadcastState(g)
	a.BroadcastEnd(g, "forfeit", winner)
	go a.PersistGame(g)
	a.Hub.RemoveGame(g.ID)
}

func (a *App) leaderboardHandler(c *gin.Context) {
	rows, err := a.DB.QueryLeaderboard(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Access-Control-Allow-Origin", "*")
	c.JSON(http.StatusOK, rows)
}

func (a *App) recentHandler(c *gin.Context) {
	limit := 10
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	rows, err := a.DB.QueryRecentGames(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Access-Control-Allow-Origin", "*")
	c.JSON(http.StatusOK, rows)
}

func (a *App) routes() http.Handler {
	r := gin.Default()
	// simple CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}
		c.Next()
	})
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	r.GET("/ws", func(c *gin.Context) { wsHandler(a, c.Writer, c.Request) })
	r.GET("/leaderboard", a.leaderboardHandler)
	r.GET("/recent", a.recentHandler)
	return r
}

func main() {
	_ = os.Setenv("TZ", "UTC")
	cfg := LoadConfig()
	app := &App{
		Hub:       NewHub(time.Duration(cfg.ReconnectGraceSeconds) * time.Second),
		DB:        MustOpenDB(cfg.PostgresDSN),
		Analytics: NewAnalytics(cfg.KafkaBrokers, cfg.KafkaTopic),
	}
	go app.DB.AutoMigrate()
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: app.routes()}
	log.Println("backend listening on", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
