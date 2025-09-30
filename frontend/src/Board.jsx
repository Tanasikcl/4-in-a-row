import React from 'react'

export default function Board({ board, onMove }){
  if (!board) return null

  return (
    <div>
      <div className="controls">
        {[...Array(7)].map((_, c) =>
          <button key={c} className="btn secondary" onClick={() => onMove(c)}>↓ {c+1}</button>
        )}
      </div>
      <div className="grid">
        {board.flatMap((row,r)=>
          row.map((cell,c)=>(
            <div key={`${r}-${c}`} className={`cell ${cell===1?'p1':cell===2?'p2':''}`} />
          ))
        )}
      </div>
    </div>
  )
}
