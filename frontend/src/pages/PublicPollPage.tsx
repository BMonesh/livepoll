import { type FormEvent, useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'

import { api, ApiError, message, type Poll, type PollView } from '../api'
import LiveResults from '../components/LiveResults'
import usePollEvents from '../hooks/usePollEvents'

export default function PublicPollPage() {
  const { slug = '' } = useParams()
  return <BallotPage key={slug} slug={slug} />
}

function BallotPage({ slug }: { slug: string }) {
  const [poll, setPoll] = useState<PollView>()
  const [option, setOption] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [retry, setRetry] = useState(0)

  useEffect(() => {
    let current = true
    api.poll(slug).then(view => { if (current) setPoll(view) }).catch(error => {
      if (current) setError(message(error))
    })
    return () => { current = false }
  }, [slug, retry])

  const update = useCallback((result: Poll) => {
    setPoll(current => current && result.revision >= current.revision ? { ...current, ...result } : current)
  }, [])
  const connection = usePollEvents(slug, !!poll && !poll.closed, update)

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!option || busy) return
    setBusy(true)
    setError('')
    try {
      const voted = await api.vote(slug, option)
      setPoll(current => current && current.revision > voted.revision
        ? { ...current, selectedOptionId: voted.selectedOptionId }
        : voted)
    } catch (error) {
      setError(message(error))
      if (error instanceof ApiError && error.status === 409) {
        api.poll(slug).then(view => {
          setPoll(current => current && current.revision > view.revision ? current : view)
        }).catch(() => {})
      }
    } finally {
      setBusy(false)
    }
  }

  if (!poll) return <section className="card narrow">
    {error ? <><h1>Poll unavailable</h1><p role="alert">{error}</p>
      <button className="button" onClick={() => { setError(''); setRetry(value => value + 1) }}>Try again</button></>
      : <p role="status">Loading poll…</p>}
  </section>

  const selected = poll.options.find(item => item.id === poll.selectedOptionId)

  return (
    <section className="public-poll">
      <div className="poll-heading">
        <h1>{poll.question}</h1>
        <p className="muted">{poll.closed ? 'Voting is closed.' : 'Choose one option. Results update live.'}</p>
      </div>
      <div className="public-grid">
        <section className="card ballot" aria-labelledby="ballot-title">
          <h2 id="ballot-title">{selected ? 'Your vote' : poll.closed ? 'Poll options' : 'Cast your vote'}</h2>
          {selected && <p className="confirmation" role="status">✓ Vote counted</p>}
          <form className="fields" onSubmit={submit}>
            <fieldset className="ballot-options" disabled={busy || !!selected || poll.closed}>
              <legend className="sr-only">Choose one voting option</legend>
              {poll.options.map(item => (
                <label className={'ballot-option' + ((selected?.id ?? option) === item.id ? ' option-selected' : '')} key={item.id}>
                  <input name="option" type="radio" value={item.id} checked={(selected?.id ?? option) === item.id} onChange={() => setOption(item.id)} />
                  <span>{item.label}</span>
                </label>
              ))}
            </fieldset>
            {error && <p role="alert" className="error">{error}</p>}
            {!selected && !poll.closed && <button className="button primary" disabled={!option || busy}>{busy ? 'Submitting…' : 'Submit vote'}</button>}
          </form>
        </section>
        <LiveResults poll={poll} connection={connection} />
      </div>
    </section>
  )
}
