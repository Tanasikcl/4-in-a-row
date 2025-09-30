package main

import "math/rand"

// chooseBotMove: try to win, then block, then pick best scoring column (center bias)
func chooseBotMove(g *Game, botSide Cell) int {
	opp := P1
	if botSide == P1 { opp = P2 }

	// 1) winning move
	for c:=0;c<Cols;c++ { if canWin(g, c, botSide) { return c } }
	// 2) block opponent
	for c:=0;c<Cols;c++ { if canWin(g, c, opp) { return c } }
	// 3) heuristic: prefer center, then adjacent columns
	order := []int{3,2,4,1,5,0,6}
	best, bestScore := order[0], -1
	for _, c := range order {
		s := scoreCol(g, c, botSide)
		if s > bestScore { bestScore, best = s, c }
	}
	// fallback to any legal col
	if bestScore < 0 {
		cands := make([]int,0)
		for c:=0;c<Cols;c++ { if colHasSpace(g,c) { cands = append(cands,c) } }
		if len(cands)==0 { return 3 }
		return cands[rand.Intn(len(cands))]
	}
	return best
}

func canWin(g *Game, col int, who Cell) bool {
	if !colHasSpace(g, col) { return false }
	// simulate
	r := topRow(g, col)
	g.Board[r][col] = who
	win := g.checkWinAt(r, col)
	g.Board[r][col] = Empty
	return win
}

func colHasSpace(g *Game, col int) bool { return g.Board[0][col] == Empty }
func topRow(g *Game, col int) int { for r:=Rows-1;r>=0;r-- { if g.Board[r][col]==Empty { return r } } ; return -1 }

func scoreCol(g *Game, col int, who Cell) int {
	if !colHasSpace(g,col) { return -1 }
	// center preference plus potential 3-in-a-row opportunities
	score := 3 - abs(3-col)
	r := topRow(g,col)
	g.Board[r][col] = who
	if g.checkWinAt(r,col) { score += 100 }
	
	// Check for potential 3-in-a-row opportunities
	score += countConsecutive(g, r, col, who) * 10
	
	g.Board[r][col] = Empty
	return score
}

func countConsecutive(g *Game, r, c int, who Cell) int {
	dirs := [][2]int{{0,1},{1,0},{1,1},{1,-1}}
	maxCount := 0
	for _, d := range dirs {
		count := 1
		// Count in positive direction
		for step := 1; step < 4; step++ {
			r2, c2 := r+step*d[0], c+step*d[1]
			if r2 < 0 || r2 >= Rows || c2 < 0 || c2 >= Cols || g.Board[r2][c2] != who {
				break
			}
			count++
		}
		// Count in negative direction
		for step := 1; step < 4; step++ {
			r2, c2 := r-step*d[0], c-step*d[1]
			if r2 < 0 || r2 >= Rows || c2 < 0 || c2 >= Cols || g.Board[r2][c2] != who {
				break
			}
			count++
		}
		if count > maxCount {
			maxCount = count
		}
	}
	return maxCount
}

func abs(x int) int { if x<0 { return -x } ; return x }
