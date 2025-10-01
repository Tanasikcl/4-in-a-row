# 4 in a Row — Real-time Multiplayer (Go) + Bot + Kafka Analytics

A real-time **Connect Four** with:
- 1v1 matchmaking (WebSockets) + **10s bot fallback**
- Competitive bot (blocks wins, takes winning moves)
- **Rejoin** within 30s or **forfeit**
- Postgres persistence (finished games) + Leaderboard + Recent Games
- Simple React frontend
- **Kafka analytics** (bonus) via a tiny consumer (`analytics-consumer2`)


## Live Links
- Click on Frontend url for the live link, that has the complete game linked to the backend 
- To play **multiplayer** open the live link of frontend to access the game in two tabs and enter players in both and then you can begin the game
- **Frontend:** https://four-in-a-row-go.netlify.app/
- **Backend:**  https://four-in-a-row-y4xi.onrender.com  
  

> Backend may cold-start on free Render (first hit can take ~30s).

## Local Dev (quick)
```bat
:: 1) Infra (optional for analytics only, consists of postgres and redPanda)
docker compose up -d

:: 2) Backend
cd backend
set PORT=8081
set POSTGRES_DSN=postgres://4inarow:4inarow@localhost:5432/4inarow?sslmode=disable
set KAFKA_BROKERS=localhost:9092
set KAFKA_TOPIC=game-analytics
set RECONNECT_GRACE_SECONDS=30
go run .

:: 3) Frontend
cd frontend
npm install
npm run dev  :: open http://localhost:5173
```
## Kafka Setup for analytics
```bat
docker compose exec redpanda rpk topic create game-analytics -p 1 -r 1
docker compose exec redpanda rpk topic list
```
## Run Analytics Consumer
```bat
cd analytics-consumer2

:: First time setup
go mod init analytics-consumer2
go get github.com/segmentio/kafka-go@v0.4.45

:: Environment
set KAFKA_BROKERS=localhost:9092
set KAFKA_TOPIC=game-analytics

:: Optional: set KAFKA_GROUP=analytics-demo

:: Run
go run .
```
- Play a local game and see line-per-event logs plus rolling summary blocks.
- Note: Hosted backend on Render does not set KAFKA_BROKERS. Analytics are local-only.