package main

import (
	"os"
	"fmt"
)

type Config struct {
	Port string
	PostgresDSN string
	KafkaBrokers string
	KafkaTopic string
	ReconnectGraceSeconds int
}

func LoadConfig() Config {
	return Config{
		Port: getenv("PORT","8080"),
		PostgresDSN: getenv("POSTGRES_DSN","postgres://4inarow:4inarow@localhost:5432/4inarow?sslmode=disable"),
		KafkaBrokers: getenv("KAFKA_BROKERS","localhost:9092"),
		KafkaTopic: getenv("KAFKA_TOPIC","game.analytics"),
		ReconnectGraceSeconds: atoi(getenv("RECONNECT_GRACE_SECONDS","30")),
	}
}

func getenv(k, def string) string { v:=os.Getenv(k); if v=="" { return def }; return v }
func atoi(s string) int { var n int; _,_ = fmt.Sscan(s,&n); return n }
