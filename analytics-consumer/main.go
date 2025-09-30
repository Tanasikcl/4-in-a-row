package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type GameMetrics struct {
	TotalGames     int
	TotalMoves     int
	GamesByHour    map[int]int
	PlayerStats    map[string]PlayerStats
	AverageDuration time.Duration
	BotWins        int
	PlayerWins     int
	Draws          int
}

type PlayerStats struct {
	GamesPlayed int
	Wins        int
	Moves       int
}

func main(){
	brokers := "localhost:9092"
	topic := "game.analytics"
	r := kafka.NewReader(kafka.ReaderConfig{Brokers: []string{brokers}, Topic: topic, GroupID: "analytics"})
	defer r.Close()
	
	metrics := &GameMetrics{
		GamesByHour: make(map[int]int),
		PlayerStats: make(map[string]PlayerStats),
	}
	
	gameStarts := make(map[string]time.Time)
	
	log.Println("Analytics consumer started. Listening for game events...")
	
	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil { 
			log.Printf("Error reading message: %v", err)
			continue 
		}
		
		var ev map[string]any
		if err := json.Unmarshal(m.Value, &ev); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}
		
		eventType := fmt.Sprint(ev["event"])
		timestamp := time.Now()
		
		switch eventType {
		case "match.start":
			gameId := fmt.Sprint(ev["gameId"])
			gameStarts[gameId] = timestamp
			metrics.TotalGames++
			hour := timestamp.Hour()
			metrics.GamesByHour[hour]++
			
			p1 := fmt.Sprint(ev["p1"])
			p2 := fmt.Sprint(ev["p2"])
			
			// Update player stats
			if p1 != "BOT" {
				stats := metrics.PlayerStats[p1]
				stats.GamesPlayed++
				metrics.PlayerStats[p1] = stats
			}
			if p2 != "BOT" {
				stats := metrics.PlayerStats[p2]
				stats.GamesPlayed++
				metrics.PlayerStats[p2] = stats
			}
			
		case "match.paired":
			// Additional logic for when players are paired
			log.Printf("Players paired: %v", ev)
			
		case "move":
			metrics.TotalMoves++
			player := fmt.Sprint(ev["by"])
			if player != "BOT" {
				stats := metrics.PlayerStats[player]
				stats.Moves++
				metrics.PlayerStats[player] = stats
			}
			
		case "game.end":
			gameId := fmt.Sprint(ev["gameId"])
			if startTime, exists := gameStarts[gameId]; exists {
				duration := timestamp.Sub(startTime)
				metrics.AverageDuration = (metrics.AverageDuration*time.Duration(metrics.TotalGames-1) + duration) / time.Duration(metrics.TotalGames)
				delete(gameStarts, gameId)
			}
			
			winner := fmt.Sprint(ev["winner"])
			if winner == "BOT" {
				metrics.BotWins++
			} else if winner != "" {
				metrics.PlayerWins++
				if stats, exists := metrics.PlayerStats[winner]; exists {
					stats.Wins++
					metrics.PlayerStats[winner] = stats
				}
			} else {
				metrics.Draws++
			}
		}
		
		// Print metrics every 30 seconds
		if time.Now().Second()%30 == 0 {
			printMetrics(metrics)
		}
	}
}

func printMetrics(m *GameMetrics) {
	log.Println("=== GAME ANALYTICS ===")
	log.Printf("Total Games: %d", m.TotalGames)
	log.Printf("Total Moves: %d", m.TotalMoves)
	log.Printf("Average Game Duration: %v", m.AverageDuration)
	log.Printf("Bot Wins: %d, Player Wins: %d, Draws: %d", m.BotWins, m.PlayerWins, m.Draws)
	
	log.Println("Games by Hour:")
	for hour, count := range m.GamesByHour {
		log.Printf("  %02d:00 - %d games", hour, count)
	}
	
	log.Println("Top Players:")
	count := 0
	for player, stats := range m.PlayerStats {
		if count >= 5 { break }
		log.Printf("  %s: %d games, %d wins, %d moves", player, stats.GamesPlayed, stats.Wins, stats.Moves)
		count++
	}
	log.Println("=====================")
}
