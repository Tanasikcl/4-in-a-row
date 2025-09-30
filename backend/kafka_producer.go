package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type Analytics struct { writer *kafka.Writer }

func NewAnalytics(brokers, topic string) *Analytics {
	w := &kafka.Writer{ Addr: kafka.TCP(brokers), Topic: topic, Balancer: &kafka.LeastBytes{} }
	return &Analytics{ writer: w }
}

func (a *Analytics) Emit(event string, payload map[string]any) {
	if a == nil || a.writer == nil { return }
	payload["event"] = event
	payload["ts"] = time.Now().UTC()
	b, _ := json.Marshal(payload)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := a.writer.WriteMessages(ctx, kafka.Message{Value: b}); err != nil {
		log.Println("kafka emit err:", err)
	}
}
