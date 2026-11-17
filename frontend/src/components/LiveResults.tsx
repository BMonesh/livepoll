import type { Poll } from '../api'
import type { Connection } from '../hooks/usePollEvents'

export default function LiveResults({ poll, connection }: { poll: Poll; connection: Connection }) {
  return (
    <section className="card results" aria-labelledby="results-title">
      <div className="results-heading">
        <h2 id="results-title">{poll.closed ? 'Final results' : 'Live results'}</h2>
        {!poll.closed && <span className={'status ' + (connection === 'live' ? 'status-live' : 'status-wait')} role="status">
          <span aria-hidden="true" />
          {connection === 'live' ? 'Live' : connection === 'reconnecting' ? 'Reconnecting' : 'Connecting'}
        </span>}
      </div>
      <ul className="result-list">
        {poll.options.map(option => {
          const share = poll.total > 0 ? 100 * option.count / poll.total : 0
          return (
            <li key={option.id}>
              <div className="result-label"><strong>{option.label}</strong><span>{option.count} · {Math.round(share)}%</span></div>
              <div className="result-track" aria-hidden="true"><div style={{ width: share + '%' }} /></div>
            </li>
          )
        })}
      </ul>
      <p className="result-total">{poll.total.toLocaleString()} {poll.total === 1 ? 'vote' : 'votes'}</p>
    </section>
  )
}
