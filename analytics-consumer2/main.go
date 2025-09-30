package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
)

type GameEnd struct {
	Reason   string  `json:"reason"`
	Winner   string  `json:"winner"`
	Duration string  `json:"duration"` 
	P1       string  `json:"p1"`
	P2       string  `json:"p2"`
}

type Aggregates struct {
	mu          sync.Mutex
	totalGames  int
	totalDurNS  time.Duration
	wins        map[string]int
	perHour     map[time.Time]int
	lastPrint   time.Time
}

func (a *Aggregates) addEnd(evt GameEnd, ts time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.totalGames++
	if d, err := time.ParseDuration(evt.Duration); err == nil {
		a.totalDurNS += d
	}
	if evt.Winner != "" {
		if a.wins == nil { a.wins = map[string]int{} }
		a.wins[evt.Winner]++
	}
	hr := ts.Truncate(time.Hour)
	if a.perHour == nil { a.perHour = map[time.Time]int{} }
	a.perHour[hr]++
}

func (a *Aggregates) print() {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	if now.Sub(a.lastPrint) < 10*time.Second {
		return
	}
	a.lastPrint = now
	avg := time.Duration(0)
	if a.totalGames > 0 {
		avg = a.totalDurNS / time.Duration(a.totalGames)
	}
	fmt.Println("---- Analytics Snapshot ----")
	fmt.Printf("Games finished: %d\n", a.totalGames)
	fmt.Printf("Avg duration : %v\n", avg)
	fmt.Println("Wins by user:")
	for u, c := range a.wins {
		fmt.Printf("  %s: %d\n", u, c)
	}
	fmt.Println("Games per hour:")
	// show last 5 hours windows that exist
	type kv struct {
		T time.Time
		N int
	}
	var buckets []kv
	for t, n := range a.perHour {
		buckets = append(buckets, kv{t, n})
	}
	// simple bubble-ish sort for tiny set
	for i := 0; i < len(buckets); i++ {
		for j := i+1; j < len(buckets); j++ {
			if buckets[j].T.Before(buckets[i].T) {
				buckets[i], buckets[j] = buckets[j], buckets[i]
			}
		}
	}
	for _, b := range buckets {
		fmt.Printf("  %s : %d\n", b.T.Format("2006-01-02 15:00"), b.N)
	}
	fmt.Println("----------------------------")
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}

func main() {
	brokers := getenv("KAFKA_BROKERS", "localhost:9092")
	topic   := getenv("KAFKA_TOPIC",   "game-analytics")
	groupID := os.Getenv("KAFKA_GROUP")

	if topic == "" {
		log.Fatal("KAFKA_TOPIC is required")
	}

	log.Printf("Analytics consumer started. brokers=%s topic=%s group=%s", brokers, topic, groupID)

	agg := &Aggregates{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		<-sigc
		log.Println("shutting down…")
		cancel()
	}()

	var r *kafka.Reader
	if groupID == "" {
		
		r = kafka.NewReader(kafka.ReaderConfig{
			Brokers:   strings.Split(brokers, ","),
			Topic:     topic,
			MinBytes:  1,
			MaxBytes:  10e6,
			StartOffset: kafka.FirstOffset, 
		})
	} else {
		
		r = kafka.NewReader(kafka.ReaderConfig{
			Brokers:   strings.Split(brokers, ","),
			Topic:     topic,
			GroupID:   groupID,
			MinBytes:  1,
			MaxBytes:  10e6,
			
		})
	}
	defer r.Close()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				agg.print()
			}
		}
	}()

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("read error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		// Print raw event
		fmt.Printf("[%s] key=%s val=%s\n",
			time.Now().Format(time.RFC3339),
			string(m.Key), string(m.Value),
		)

		if string(m.Key) == "game.end" || looksLikeGameEnd(m.Value) {
			var ge GameEnd
			if err := json.Unmarshal(m.Value, &ge); err == nil {
				agg.addEnd(ge, time.Now())
			}
		}
	}
}

func looksLikeGameEnd(b []byte) bool {
	s := string(b)
	return strings.Contains(s, `"reason"`) && (strings.Contains(s, `"winner"`) || strings.Contains(s, `"duration"`))
}
