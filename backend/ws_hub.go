package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)


func (a *App) pushStateToUser(g *Game, username string) {
	ws := a.Hub.Conn(username)
	if ws == nil {
		return
	}
	you := P1
	opp := g.P2
	if username == g.P2 {
		you = P2
		opp = g.P1
	}
	_ = ws.SafeWriteJSON(WSMessage{
		Type: "state",
		Data: StatePayload{
			GameID:   g.ID.String(),
			Board:    g.Board,
			Turn:     g.Turn,
			You:      you,
			Opponent: opp,
		},
	})
}

type WSConn struct {
	*websocket.Conn
	mu sync.Mutex
}

func (c *WSConn) SafeWriteJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.WriteJSON(v)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	mu            sync.RWMutex
	waiting       *Player           
	games         map[GameID]*Game   
	playersInGame map[string]GameID  
	conns         map[string]*WSConn 
	reconnectTTL  time.Duration
}

func NewHub(ttl time.Duration) *Hub {
	return &Hub{
		games:         make(map[GameID]*Game),
		playersInGame: make(map[string]GameID),
		conns:         make(map[string]*WSConn),
		reconnectTTL:  ttl,
	}
}

func (h *Hub) EnqueueOrMatch(p *Player) (*Game, Cell, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	p.LastSeen = time.Now()

	// Rejoin existing game
	if gid, ok := h.playersInGame[p.Username]; ok {
		g := h.games[gid]
		return g, h.sideOf(g, p.Username), true
	}

	// Pair with waiting player
	if h.waiting != nil && h.waiting.Username != p.Username {
		g := newGame(h.waiting.Username, p.Username)
		h.games[g.ID] = g
		h.playersInGame[g.P1] = g.ID
		h.playersInGame[g.P2] = g.ID
		h.waiting = nil
		return g, P2, false
	}


	h.waiting = p
	return nil, Empty, false
}

// StartBotIfStillWaiting runs a 10s time, if SAME player still waiting,

func (h *Hub) StartBotIfStillWaiting(create func(player *Player)) {
	h.mu.RLock()
	w := h.waiting
	h.mu.RUnlock()
	if w == nil {
		log.Printf("No waiting player found, bot timer not started")
		return
	}

	log.Printf("Bot timer started for player: %s, waiting 10 seconds...", w.Username)
	time.Sleep(10 * time.Second)

	
	h.mu.Lock()
	if h.waiting == nil || h.waiting.Username != w.Username {
		h.mu.Unlock()
		log.Printf("Player %s is no longer waiting (matched or left)", w.Username)
		return
	}
	
	p := *h.waiting
	h.waiting = nil
	h.mu.Unlock()

	log.Printf("Starting bot game for waiting player: %s", p.Username)
	create(&p)
}

func (h *Hub) sideOf(g *Game, username string) Cell {
	if g.P1 == username {
		return P1
	}
	return P2
}

func (h *Hub) RemoveGame(gid GameID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	g := h.games[gid]
	if g == nil {
		return
	}
	delete(h.playersInGame, g.P1)
	delete(h.playersInGame, g.P2)
	delete(h.games, gid)
}

func (h *Hub) GetGame(gid GameID) (*Game, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	g := h.games[gid]
	return g, g != nil
}

func (h *Hub) SetConn(username string, ws *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[username] = ws
}

func (h *Hub) DelConn(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, username)
}

func (h *Hub) Conn(username string) *WSConn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.conns[username]
}

func wsHandler(app *App, w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	ws := &WSConn{Conn: conn}

	
	app.Hub.SetConn(username, ws)

	player := &Player{Username: username, Conn: ws, LastSeen: time.Now()}

	// register or rejoin
	game, side, rejoined := app.Hub.EnqueueOrMatch(player)
	if game == nil {
		// waiting: start bot timer but DO NOT return; keep socket open
		log.Printf("Player %s is waiting for opponent, starting bot timer", username)
		go app.Hub.StartBotIfStillWaiting(func(p *Player) {
			// create bot game and push state directly to this user
			g := newGame(p.Username, "BOT")
			app.Hub.mu.Lock()
			app.Hub.games[g.ID] = g
			app.Hub.playersInGame[g.P1] = g.ID
			app.Hub.playersInGame[g.P2] = g.ID
			app.Hub.mu.Unlock()

			log.Printf("Created bot game: ID=%s, P1=%s, P2=%s", g.ID.String(), g.P1, g.P2)
			app.Analytics.Emit("match.start", map[string]any{"gameId": g.ID.String(), "p1": g.P1, "p2": g.P2})

			
			app.pushStateToUser(g, p.Username)
			app.BroadcastState(g)
		})
		_ = ws.SafeWriteJSON(WSMessage{Type: "state", Data: map[string]any{"waiting": true}})
	} else {
		
		you := side
		opponent := game.P1
		if you == P1 {
			opponent = game.P2
		}
		_ = ws.SafeWriteJSON(WSMessage{
			Type: "state",
			Data: StatePayload{GameID: game.ID.String(), Board: game.Board, Turn: game.Turn, You: you, Opponent: opponent},
		})
		if !rejoined {
			app.Analytics.Emit("match.paired", map[string]any{"gameId": game.ID.String(), "p1": game.P1, "p2": game.P2})
		}
	}

	
	go func() {
		defer func() {
			conn.Close()
			app.Hub.DelConn(username)
			go app.ForfeitIfNotRejoined(username)
		}()
		for {
			var incoming WSMessage
			if err := conn.ReadJSON(&incoming); err != nil {
				log.Println("read err:", err)
				return
			}

			switch incoming.Type {
			case "move":
				col := -1
				if m, ok := incoming.Data.(map[string]any); ok {
					if f, ok2 := m["col"].(float64); ok2 {
						col = int(f)
					}
				}
				if col >= 0 {
					app.HandleMove(username, col)
				}

			case "regame":
				mode := "matchmaking"
				if m, ok := incoming.Data.(map[string]any); ok {
					if s, ok2 := m["mode"].(string); ok2 {
						mode = s
					}
				}
				log.Printf("Received regame from %s (mode=%s)", username, mode)
				app.HandleRegame(username, mode)

			default:
				
			}
		}
	}()
}
