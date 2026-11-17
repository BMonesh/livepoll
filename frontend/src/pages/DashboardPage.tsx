import { useEffect, useRef, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'

import { api, ApiError, message, type Poll, type User } from '../api'
import PollQrCode from '../components/PollQrCode'

function pollUrl(slug: string) {
  return new URL('/p/' + slug, window.location.origin).href
}

export default function DashboardPage() {
  const [user, setUser] = useState<User>()
  const [polls, setPolls] = useState<Poll[]>([])
  const [more, setMore] = useState(false)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [selected, setSelected] = useState<Poll>()
  const [dialogMode, setDialogMode] = useState<'qr' | 'close'>('qr')
  const dialog = useRef<HTMLDialogElement>(null)
  const shareInput = useRef<HTMLInputElement>(null)
  const navigate = useNavigate()
  const location = useLocation()
  const created = (location.state as { created?: string } | null)?.created

  useEffect(() => {
    let current = true
    Promise.all([api.me(), api.list()]).then(([account, items]) => {
      if (!current) return
      setUser(account)
      setPolls(items)
      setMore(items.length === 50)
      setLoading(false)
    }).catch(error => {
      if (!current) return
      if (error instanceof ApiError && error.status === 401) navigate('/login', { replace: true })
      else { setError(message(error)); setLoading(false) }
    })
    return () => { current = false }
  }, [navigate])

  function openDialog(poll: Poll, mode: 'qr' | 'close') {
    setError('')
    setSelected(poll)
    setDialogMode(mode)
    dialog.current?.showModal()
  }

  async function copy(slug: string) {
    try {
      await navigator.clipboard.writeText(pollUrl(slug))
      setNotice('Link copied.')
    } catch {
      const poll = polls.find(item => item.slug === slug)
      if (poll) openDialog(poll, 'qr')
      requestAnimationFrame(() => shareInput.current?.select())
      setNotice('Select and copy the poll link.')
    }
  }

  async function closeSelected() {
    if (!selected) return
    setBusy(true)
    setError('')
    try {
      const closed = await api.close(selected.slug)
      setPolls(current => current.map(poll => poll.slug === closed.slug ? closed : poll))
      dialog.current?.close()
      setNotice('Voting closed.')
    } catch (error) {
      setError(message(error))
    } finally {
      setBusy(false)
    }
  }

  async function loadMore() {
    setBusy(true)
    setError('')
    try {
      const next = await api.list(polls.at(-1)?.createdAt)
      setPolls(current => [...current, ...next])
      setMore(next.length === 50)
    } catch (error) {
      setError(message(error))
    } finally {
      setBusy(false)
    }
  }

  async function signOut() {
    setBusy(true)
    setError('')
    try {
      await api.logout()
      navigate('/login', { replace: true })
    } catch (error) {
      setError(message(error))
      setBusy(false)
    }
  }

  if (loading) return <section className="card narrow"><p role="status">Loading your polls…</p></section>
  if (!user) return <section className="card narrow"><p role="alert">{error}</p><button className="button" onClick={() => locationReload()}>Try again</button></section>

  return (
    <section className="dashboard">
      <div className="dashboard-title">
        <div><h1>My polls</h1><p className="muted dashboard-account">{user.email} <span aria-hidden="true">·</span> <button className="text-button" onClick={signOut} disabled={busy}>Sign out</button></p></div>
        <Link className="button primary" to="/polls/new">+ Create poll</Link>
      </div>
      <div className="dashboard-feedback" aria-live="polite">{notice || (created ? 'Poll published. Share the link or show its QR code.' : '')}</div>
      {error && <p className="error" role="alert">{error}</p>}

      {polls.length === 0 && <div className="card empty">
        <span className="brand-mark" aria-hidden="true"><i /><i /><i /></span>
        <h2>No polls yet</h2>
        <p className="muted">Create a question to share with your audience.</p>
        <Link className="button primary" to="/polls/new">Create your first poll</Link>
      </div>}

      <div className="poll-list">
        {polls.map(poll => (
          <article className={'card poll-summary' + (poll.slug === created ? ' just-created' : '')} key={poll.slug}>
            <div className="poll-summary-main">
              <div className="poll-summary-status"><span className={'status ' + (poll.closed ? 'status-closed' : 'status-live')}><span aria-hidden="true" />{poll.closed ? 'Closed' : 'Open'}</span>
                <span className="muted">{new Date(poll.createdAt).toLocaleDateString(undefined, { dateStyle: 'medium' })}</span>
              </div>
              <h2><Link to={'/p/' + poll.slug}>{poll.question}</Link></h2>
              <p className="muted">{poll.options.length} options · {poll.total} {poll.total === 1 ? 'vote' : 'votes'}</p>
            </div>
            <div className="poll-actions">
              <button className="button" onClick={() => copy(poll.slug)}>Copy link</button>
              <button className="button" onClick={() => openDialog(poll, 'qr')}>Show QR code</button>
              {!poll.closed && <button className="button subtle" onClick={() => openDialog(poll, 'close')}>Close voting</button>}
            </div>
          </article>
        ))}
      </div>
      {more && <button className="button load-more" onClick={loadMore} disabled={busy}>Load older polls</button>}

      <dialog ref={dialog} aria-labelledby="dialog-title" onClose={() => { setSelected(undefined); setError('') }} onCancel={event => { if (busy) event.preventDefault() }}>
        {selected && <div className="dialog-content">
          <div className="dialog-heading"><h2 id="dialog-title">{dialogMode === 'qr' ? 'Share this poll' : 'Close voting?'}</h2><button className="icon-button" aria-label="Close dialog" disabled={busy} onClick={() => dialog.current?.close()}>×</button></div>
          <p className="muted">{selected.question}</p>
          {dialogMode === 'qr' ? <>
            <PollQrCode key={selected.slug} url={pollUrl(selected.slug)} />
            <label className="sr-only" htmlFor="poll-share-url">Poll link</label>
            <input ref={shareInput} id="poll-share-url" readOnly value={pollUrl(selected.slug)} onFocus={event => event.currentTarget.select()} />
            <button className="button primary" onClick={() => copy(selected.slug)}>Copy link</button>
            <span role="status">{notice}</span>
          </> : <>
            <p>Voting will end immediately. The link will continue to show the final results.</p>
            {error && <p className="error" role="alert">{error}</p>}
            <div className="dialog-buttons">
              <button className="button" disabled={busy} onClick={() => dialog.current?.close()}>Cancel</button>
              <button className="button danger" disabled={busy} onClick={closeSelected}>{busy ? 'Closing…' : 'Close voting'}</button>
            </div>
          </>}
        </div>}
      </dialog>
    </section>
  )
}

function locationReload() {
  window.location.reload()
}
