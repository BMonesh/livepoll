import { type FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { api, ApiError, message } from '../api'

type Choice = { id: number; label: string }

export default function CreatePollPage() {
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState<Choice[]>([{ id: 1, label: '' }, { id: 2, label: '' }])
  const [nextId, setNextId] = useState(3)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setBusy(true)
    try {
      const poll = await api.create(question, options.map(option => option.label))
      navigate('/dashboard', { state: { created: poll.slug } })
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) navigate('/login', { replace: true })
      else setError(message(error))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="narrow">
      <Link className="back-link" to="/dashboard">← My polls</Link>
      <h1>Create a poll</h1>
      <form onSubmit={submit} className="card fields create-card">
        <label>Your question
          <input value={question} onChange={event => setQuestion(event.target.value)} placeholder="What should we decide?" maxLength={160} required />
        </label>
        <fieldset className="fields">
          <legend>Voting options <span className="muted">2–6 options</span></legend>
          {options.map((option, index) => (
            <div className="option-field" key={option.id}>
              <label htmlFor={'option-' + option.id} className="sr-only">Option {index + 1}</label>
              <span className="option-index" aria-hidden="true">{String.fromCharCode(65 + index)}</span>
              <input id={'option-' + option.id} placeholder={'Option ' + (index + 1)} required maxLength={80} value={option.label}
                onChange={event => setOptions(current => current.map(item => item.id === option.id ? { ...item, label: event.target.value } : item))} />
              {options.length > 2 && <button className="icon-button" type="button" aria-label={'Remove option ' + (index + 1)} onClick={() => setOptions(current => current.filter(item => item.id !== option.id))}>×</button>}
            </div>
          ))}
          <button className="button subtle add-button" type="button" disabled={options.length === 6} onClick={() => {
            setOptions(current => [...current, { id: nextId, label: '' }])
            setNextId(id => id + 1)
          }}>+ Add option</button>
        </fieldset>
        {error && <p role="alert" className="error">{error}</p>}
        <div className="form-end">
          <p className="muted">Anyone with the link can vote and see results.</p>
          <button className="button primary" disabled={busy}>{busy ? 'Publishing…' : 'Publish poll'}</button>
        </div>
      </form>
    </section>
  )
}
