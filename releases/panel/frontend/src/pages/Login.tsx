import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import { useAuth } from '../store/auth'
import { Button, Input } from '../components/ui'

export default function Login() {
  const [email, setEmail] = useState('admin@local')
  const [password, setPassword] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const setTokens = useAuth((s) => s.setTokens)
  const setUser = useAuth((s) => s.setUser)
  const nav = useNavigate()

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setErr(null)
    setLoading(true)
    try {
      const t = await api.auth.login(email, password)
      setTokens(t.access_token, t.refresh_token)
      try {
        const me = await api.me.get()
        setUser(me)
      } catch { /* ignore */ }
      nav('/', { replace: true })
    } catch (e) {
      if (e instanceof ApiError) setErr(e.code === 'invalid_credentials' ? 'Wrong email or password' : (e.code ?? `HTTP ${e.status}`))
      else setErr('Network error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-screen">
      <form className="login-card stack" onSubmit={submit}>
        <div className="login-title">
          <span className="brand-dot" /> void-wg
        </div>
        <div className="login-sub">Sign in to your control panel.</div>
        <Input
          label="Email"
          type="email"
          autoComplete="username"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        <Input
          label="Password"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        {err && <div className="text-danger text-sm">{err}</div>}
        <Button type="submit" variant="primary" loading={loading} style={{ marginTop: 4 }}>
          Sign in
        </Button>
      </form>
    </div>
  )
}
