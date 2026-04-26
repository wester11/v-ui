import { useEffect, useMemo, useState } from 'react'
import { api, ApiError } from '../api/client'
import type { User } from '../types'
import {
  Badge, Button, ConfirmDialog, Empty, Input, Modal, Select, Skeleton, toast,
} from '../components/ui'
import { useAuth } from '../store/auth'
import { formatDate } from '../lib/format'

const empty = { email: '', password: '', role: 'user' as 'user' | 'operator' | 'admin' }

function roleBadge(role: string) {
  switch (role) {
    case 'admin':    return <Badge tone="violet">admin</Badge>
    case 'operator': return <Badge tone="info">operator</Badge>
    default:         return <Badge>user</Badge>
  }
}

export default function Users() {
  const me = useAuth((s) => s.user)
  const [list, setList] = useState<User[] | null>(null)
  const [search, setSearch] = useState('')
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState(empty)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const [confirm, setConfirm] = useState<User | null>(null)
  const [delBusy, setDelBusy] = useState(false)

  async function reload() {
    try {
      setList(await api.users.list())
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'load failed')
    }
  }

  useEffect(() => { reload() }, [])

  const filtered = useMemo(() => {
    if (!list) return null
    const q = search.trim().toLowerCase()
    if (!q) return list
    return list.filter((u) =>
      u.email.toLowerCase().includes(q) ||
      u.role.toLowerCase().includes(q),
    )
  }, [list, search])

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setErr(null)
    setBusy(true)
    try {
      await api.users.create(form)
      setForm(empty)
      setCreating(false)
      await reload()
      toast.success('User created')
    } catch (e) {
      setErr(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'create failed')
    } finally {
      setBusy(false)
    }
  }

  async function remove() {
    if (!confirm) return
    setDelBusy(true)
    try {
      await api.users.delete(confirm.id)
      toast.success('User deleted')
      setConfirm(null)
      await reload()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'delete failed')
    } finally {
      setDelBusy(false)
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">Users</div>
          <div className="page-sub">Accounts that can access this control plane.</div>
        </div>
        <div className="row">
          <input
            className="search-input"
            placeholder="Search by email…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Button variant="primary" onClick={() => setCreating(true)}>+ New user</Button>
        </div>
      </div>

      <div className="card">
        {list === null ? (
          <div className="stack">
            {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} height={42} />)}
          </div>
        ) : filtered && filtered.length === 0 ? (
          <Empty
            title={search ? 'No users match your search' : 'No users yet'}
            sub={search ? 'Try a different query.' : 'Create the first user to grant access.'}
          />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Email</th>
                  <th>Role</th>
                  <th>Status</th>
                  <th>Created</th>
                  <th className="actions">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered!.map((u) => (
                  <tr key={u.id}>
                    <td>
                      <strong>{u.email}</strong>
                      {me?.id === u.id && <span className="text-mute text-xs"> · you</span>}
                    </td>
                    <td>{roleBadge(u.role)}</td>
                    <td>
                      {u.disabled
                        ? <Badge tone="danger">disabled</Badge>
                        : <Badge tone="success">active</Badge>}
                    </td>
                    <td className="text-dim">{formatDate(u.created_at)}</td>
                    <td className="actions">
                      <div className="row-end">
                        <Button
                          size="sm"
                          variant="danger"
                          disabled={me?.id === u.id}
                          title={me?.id === u.id ? 'Cannot delete yourself' : undefined}
                          onClick={() => setConfirm(u)}
                        >
                          Delete
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <Modal
        open={creating}
        onClose={() => setCreating(false)}
        title="New user"
        footer={
          <>
            <Button variant="ghost" onClick={() => setCreating(false)}>Cancel</Button>
            <Button variant="primary" onClick={submit as never} loading={busy}>Create</Button>
          </>
        }
      >
        <form className="stack" onSubmit={submit}>
          <Input
            label="Email"
            type="email"
            placeholder="alice@example.com"
            value={form.email}
            onChange={(e) => setForm({ ...form, email: e.target.value })}
            required
            autoFocus
          />
          <Input
            label="Password"
            type="password"
            placeholder="min 8 chars"
            value={form.password}
            minLength={8}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
            required
          />
          <Select
            label="Role"
            value={form.role}
            onChange={(e) => setForm({ ...form, role: e.target.value as typeof empty.role })}
          >
            <option value="user">user — can only manage own peers</option>
            <option value="operator">operator — can manage users, servers, peers</option>
            <option value="admin">admin — full access</option>
          </Select>
          {err && <div className="text-danger text-sm">{err}</div>}
        </form>
      </Modal>

      <ConfirmDialog
        open={!!confirm}
        title="Delete user?"
        body={
          <>
            User <strong>{confirm?.email}</strong> and all their peers will be removed permanently.
          </>
        }
        confirmText="Delete"
        destructive
        loading={delBusy}
        onConfirm={remove}
        onClose={() => setConfirm(null)}
      />
    </div>
  )
}
