# 4 in a Row — Real-time Multiplayer (Go) + Bot + Kafka Analytics

A real-time **Connect Four** with:
- 1v1 matchmaking (WebSockets) + **10s bot fallback**
- Competitive bot (blocks wins, takes winning moves)
- **Rejoin** within 30s or **forfeit**
- Postgres persistence (finished games) + Leaderboard + Recent Games
- Simple React frontend
- **Kafka analytics** (bonus) via a tiny consumer (`analytics-consumer2`)

## Live Links
- **Frontend:** https://<YOUR-NETLIFY>.netlify.app
- **Backend:**  https://<YOUR-BACKEND>.onrender.com  
  - Health: `/health`  
  - Leaderboard: `/leaderboard`  
  - Recent: `/recent?limit=10`  
  - WebSocket: `/ws?username=NAME`

> Backend may cold-start on free Render (first hit can take ~30s).

## Local Dev (quick)
```bat
:: 1) Infra (optional for analytics only)
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
