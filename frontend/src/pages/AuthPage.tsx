import { type FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { api, message } from '../api'

export default function AuthPage({ mode }: { mode: 'signup' | 'login' }) {
  const signup = mode === 'signup'
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const navigate = useNavigate()

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      await api.auth(mode, email, password)
      navigate('/dashboard', { replace: true })
    } catch (error) {
      setError(message(error))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="auth-layout">
      <div className="auth-intro">
        <span className="auth-illustration" aria-hidden="true"><i /><i /><i /></span>
        <h1>Put the question to the room.</h1>
        <p>Create a poll, share a link, and watch the votes come in live.</p>
      </div>
      <div className="card auth-card">
        <h2>{signup ? 'Create your account' : 'Sign in'}</h2>
        <form onSubmit={submit} className="fields">
          <label>Email address
            <input type="email" autoComplete="email" value={email} required maxLength={254} onChange={event => setEmail(event.target.value)} />
          </label>
          <label>Password
            <input type="password" autoComplete={signup ? 'new-password' : 'current-password'} value={password} required minLength={signup ? 10 : undefined} onChange={event => setPassword(event.target.value)} />
          </label>
          {signup && <p className="muted field-hint">Use at least 10 characters.</p>}
          {error && <p className="error" role="alert">{error}</p>}
          <button className="button primary" disabled={busy}>{busy ? 'Please wait…' : signup ? 'Create account' : 'Sign in'}</button>
        </form>
        <p className="auth-switch">{signup ? 'Already have an account?' : 'New to LivePoll?'} <Link to={signup ? '/login' : '/signup'}>{signup ? 'Sign in' : 'Create an account'}</Link></p>
      </div>
    </section>
  )
}
