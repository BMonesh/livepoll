import { Navigate, Route, Routes, Link } from 'react-router-dom'

import AuthPage from './pages/AuthPage'
import CreatePollPage from './pages/CreatePollPage'
import DashboardPage from './pages/DashboardPage'
import PublicPollPage from './pages/PublicPollPage'

export default function App() {
  return (
    <>
      <header className="site-header">
        <div className="header-inner">
          <Link className="brand" to="/dashboard">
            <span className="brand-mark" aria-hidden="true"><i /><i /><i /></span>
            Live<span>Poll</span>
          </Link>
          <Link className="header-link" to="/dashboard">Creator dashboard <span aria-hidden="true">↗</span></Link>
        </div>
      </header>
      <main className="page">
        <Routes>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/login" element={<AuthPage key="login" mode="login" />} />
          <Route path="/signup" element={<AuthPage key="signup" mode="signup" />} />
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/polls/new" element={<CreatePollPage />} />
          <Route path="/p/:slug" element={<PublicPollPage />} />
          <Route path="*" element={<section className="card narrow"><h1>Page unavailable</h1><Link to="/dashboard">Go to dashboard</Link></section>} />
        </Routes>
      </main>
    </>
  )
}
