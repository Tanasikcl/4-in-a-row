import React, { useEffect, useState } from 'react'
import Board from './Board'
import './styles.css'


const API_BASE =
  import.meta.env.VITE_API_BASE
  || (window.location.hostname.includes("netlify.app")
      ? "https://four-in-a-row-y4xi.onrender.com"
      : "http://localhost:8081");

const WS_BASE =
  import.meta.env.VITE_WS_BASE
  || (window.location.hostname.includes("netlify.app")
      ? "wss://four-in-a-row-y4xi.onrender.com"
      : "ws://localhost:8081");



export default function App(){
  const [username, setUsername] = useState('')
  const [ws, setWs] = useState(null)
  const [state, setState] = useState(null)
  const [status, setStatus] = useState('')
  const [ended, setEnded] = useState(false)
  const [lb, setLb] = useState([])
  const [recent, setRecent] = useState([])  

  useEffect(()=>{
    const tick = () => {
      fetch(`${API_BASE}/leaderboard`).then(r=>r.json()).then(setLb).catch(()=>{})
      fetch(`${API_BASE}/recent?limit=10`).then(r=>r.json()).then(setRecent).catch(()=>{})
    }
    tick()
    const id = setInterval(tick, 3000)
    return ()=>clearInterval(id)
  },[])

  const connect = () => {
    if (!username) return
    try { ws?.close() } catch {}
    const sock = new WebSocket(`${WS_BASE}/ws?username=${encodeURIComponent(username)}`)
    sock.onopen = () => setStatus('Connected. Matching… (bot joins in ~10s if no player)')
    sock.onmessage = ev => {
      const msg = JSON.parse(ev.data)

      if (msg.type === 'state') {
        setState(msg.data)
        if (msg.data?.waiting) {
          setEnded(false)
          setStatus('Waiting for an opponent… (bot in ~10s)')
        } else if (msg.data?.board) {
          setEnded(false)
          const yourTurn = msg.data.turn === msg.data.you
          setStatus(yourTurn ? 'Play — your move' : `Play — waiting for ${msg.data.opponent}`)
        }
      }

      if (msg.type === 'end') {
        setStatus(`Game over: ${msg.data.reason}. Winner: ${msg.data.winner || '—'}`)
        setEnded(true)
      }
    }
    sock.onclose = () => setStatus('Disconnected')
    setWs(sock)
  }

  const sendMove = (col) => {
    if(!ws) return
    ws.send(JSON.stringify({ type:'move', data:{ col } }))
  }

  const regame = (mode) => {
    if (!ws) return
    setStatus(mode === 'bot' ? 'Creating bot game…' : 'Re-queueing for match…')
    setEnded(false)
    if (mode === 'bot') setState(null); else setState({ waiting: true })
    ws.send(JSON.stringify({ type: 'regame', data: { mode } }))
  }

  return (
    <div className="page">
      <div className="card">
        <h1 className="title">🎯 <span>4 in a Row</span></h1>

        <div className="join">
          <input
            className="input"
            placeholder="Enter username"
            value={username}
            onChange={e=>setUsername(e.target.value)}
          />
          <button className="btn" onClick={connect} disabled={!username}>Join Game</button>
        </div>

        <div className="status">
          {status || (state?.waiting ? 'Waiting for an opponent… (bot in ~10s)' : 'Idle')}
        </div>

        {state?.board && (
          <>
            <p className="meta">
              Opponent: <b>{state.opponent ?? '—'}</b> • Your side: <b>{state.you===1?'P1':'P2'}</b> • Turn: <b>{state.turn===1?'P1':'P2'}</b>
            </p>
            <Board board={state.board} onMove={sendMove} />
          </>
        )}

        {ended && (
          <div className="join" style={{marginTop:12}}>
            <button className="btn" onClick={() => regame('bot')}>Play Again vs Bot</button>
            <button className="btn secondary" onClick={() => regame('matchmaking')}>Find New Opponent</button>
          </div>
        )}

        {/* Wins leaderboard (unchanged) */}
        <div className="panel">
          <h3>🏆 Leaderboard</h3>
          {lb.length===0 ? (
            <p className="muted">No games played yet. Be the first to win! 🎮</p>
          ) : (
            <ol className="lb">
              {lb.map(r => <li key={r.username}>{r.username}: {r.wins}</li>)}
            </ol>
          )}
        </div>

        {/* NEW: Recent games list */}
        <div className="panel">
          <h3>📜 Recent Games</h3>
          {recent.length === 0 ? (
            <p className="muted">No games yet.</p>
          ) : (
            <ul className="games">
              {recent.map((g, i) => (
                <li key={g.id} className="gameRow">
                  <div className="gameNo">Game {recent.length - i}</div>
                  <div className="gameInfo">
                    <span>P1: <b>{g.p1}</b></span>
                    <span>•</span>
                    <span>P2: <b>{g.p2}</b></span>
                    <span>•</span>
                    <span>Winner: <b>{g.is_draw ? 'Draw' : (g.winner || '—')}</b></span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
