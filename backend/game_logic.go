package main

import (
	"errors"
	"time"
	"github.com/google/uuid"
)

func newGame(p1, p2 string) *Game {
	return &Game{
		ID: uuid.New(),
		P1: p1,
		P2: p2,
		Turn: P1,
		Started: time.Now(),
	}
}

func (g *Game) drop(col int) (row int, err error) {
	if col < 0 || col >= Cols { return -1, errors.New("bad col") }
	for r := Rows-1; r >= 0; r-- {
		if g.Board[r][col] == Empty {
			g.Board[r][col] = g.Turn
			return r, nil
		}
	}
	return -1, errors.New("col full")
}

func (g *Game) switchTurn() { if g.Turn == P1 { g.Turn = P2 } else { g.Turn = P1 } }

func (g *Game) checkWinAt(r, c int) bool {
	// directions: (dr,dc)
	dirs := [][2]int{{0,1},{1,0},{1,1},{1,-1}}
	who := g.Board[r][c]
	if who == Empty { return false }
	for _, d := range dirs {
		cnt := 1
		for step:=1; step<4; step++ {
			r2, c2 := r+step*d[0], c+step*d[1]
			if r2<0||r2>=Rows||c2<0||c2>=Cols||g.Board[r2][c2]!=who { break }
			cnt++
		}
		for step:=1; step<4; step++ {
			r2, c2 := r-step*d[0], c-step*d[1]
			if r2<0||r2>=Rows||c2<0||c2>=Cols||g.Board[r2][c2]!=who { break }
			cnt++
		}
		if cnt >= 4 { return true }
	}
	return false
}

func (g *Game) isFull() bool {
	for c:=0;c<Cols;c++ { if g.Board[0][c]==Empty { return false } }
	return true
}
