import { useState } from 'react'
import { api, ApiError } from '../api/client'
import { useAuth } from '../store/auth'
import { Badge, Button, CopyField, Input, toast } from '../components/ui'
import { formatDate } from '../lib/format'

function roleBadge(role: string) {
  switch (role) {
    case 'admin':    return <Badge tone="violet">admin</Badge>
    case 'operator': return <Badge tone="info">operator</Badge>
    default:         return <Badge>user</Badge>
  }
}

export default function Profile() {
  const me = useAuth((s) => s.user)

  const [oldPwd, setOldPwd] = useState('')
  const [newPwd, setNewPwd] = useState('')
  const [confirmPwd, setConfirmPwd] = useState('')
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const [ok, setOk] = useState(false)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setErr(null)
    setOk(false)
    if (newPwd.length < 8) {
      setErr('New password must be at least 8 characters')
      return
    }
    if (newPwd !== confirmPwd) {
      setErr('Passwords do not match')
      return
    }
    setBusy(true)
    try {
      await api.me.changePassword(oldPwd, newPwd)
      setOldPwd('')
      setNewPwd('')
      setConfirmPwd('')
      setOk(true)
      toast.success('Password updated')
    } catch (e) {
      if (e instanceof ApiError) {
        setErr(
          e.code === 'invalid_credentials' ? 'Current password is wrong'
          : e.code === 'invalid_input' ? 'New password is invalid'
          : (e.code ?? `HTTP ${e.status}`),
        )
      } else {
        setErr('Network error')
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">Profile</div>
          <div className="page-sub">Your account details and security settings.</div>
        </div>
      </div>

      <div className="card mb-4">
        <div className="card-header">
          <div>
            <div className="card-title">Account</div>
            <div className="card-sub">These details are read-only here. Contact an admin to change role.</div>
          </div>
        </div>
        {me ? (
          <div className="stack">
            <Field label="Email"><strong>{me.email}</strong></Field>
            <Field label="Role">{roleBadge(me.role)}</Field>
            <Field label="Status">
              {me.disabled ? <Badge tone="danger">disabled</Badge> : <Badge tone="success">active</Badge>}
            </Field>
            <Field label="User ID"><CopyField value={me.id} /></Field>
            <Field label="Created">{formatDate(me.created_at)}</Field>
          </div>
        ) : (
          <div className="text-dim">Loading…</div>
        )}
      </div>

      <div className="card">
        <div className="card-header">
          <div>
            <div className="card-title">Change password</div>
            <div className="card-sub">Use a strong, unique password (min 8 characters).</div>
          </div>
        </div>
        <form className="stack" onSubmit={submit} style={{ maxWidth: 420 }}>
          <Input
            label="Current password"
            type="password"
            autoComplete="current-password"
            value={oldPwd}
            onChange={(e) => { setOldPwd(e.target.value); setErr(null); setOk(false) }}
            required
          />
          <Input
            label="New password"
            type="password"
            autoComplete="new-password"
            value={newPwd}
            onChange={(e) => { setNewPwd(e.target.value); setErr(null); setOk(false) }}
            minLength={8}
            required
          />
          <Input
            label="Confirm new password"
            type="password"
            autoComplete="new-password"
            value={confirmPwd}
            onChange={(e) => { setConfirmPwd(e.target.value); setErr(null); setOk(false) }}
            minLength={8}
            required
          />
          {err && <div className="text-danger text-sm">{err}</div>}
          {ok && <div className="text-success text-sm">✓ Password updated</div>}
          <div>
            <Button type="submit" variant="primary" loading={busy}>Update password</Button>
          </div>
        </form>
      </div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="row" style={{ alignItems: 'center' }}>
      <div className="text-mute text-sm" style={{ width: 140, flexShrink: 0 }}>{label}</div>
      <div style={{ flex: 1 }}>{children}</div>
    </div>
  )
}
