import { useEffect, useMemo, useState } from 'react'
import { api, ApiError } from '../api/client'
import type { User } from '../types'
import {
  Badge, Button, ConfirmDialog, Empty, Input, Modal, Skeleton, toast,
} from '../components/ui'
import { useAuth } from '../store/auth'
import { useI18n } from '../i18n'
import { formatDate } from '../lib/format'

const ROLES = ['user', 'operator', 'admin'] as const
type Role = (typeof ROLES)[number]

const empty = { email: '', password: '', role: 'user' as Role }

function roleTone(role: string): 'default' | 'info' | 'violet' {
  if (role === 'admin') return 'violet'
  if (role === 'operator') return 'info'
  return 'default'
}

function UserInitials({ email }: { email: string }) {
  const ch = email.slice(0, 2).toUpperCase()
  return <div className="user-initials">{ch}</div>
}

export default function Users() {
  const me  = useAuth((s) => s.user)
  const { t } = useI18n()

  const [list,     setList]     = useState<User[] | null>(null)
  const [search,   setSearch]   = useState('')
  const [creating, setCreating] = useState(false)
  const [form,     setForm]     = useState(empty)
  const [busy,     setBusy]     = useState(false)
  const [err,      setErr]      = useState<string | null>(null)
  const [confirm,  setConfirm]  = useState<User | null>(null)
  const [delBusy,  setDelBusy]  = useState(false)
  const [toggling, setToggling] = useState<string | null>(null)

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
      toast.success(t('toast_user_created'))
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
      toast.success(t('toast_user_deleted'))
      setConfirm(null)
      await reload()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'delete failed')
    } finally {
      setDelBusy(false)
    }
  }

  async function toggleDisabled(u: User) {
    setToggling(u.id)
    try {
      await api.users.setDisabled(u.id, !u.disabled)
      await reload()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'toggle failed')
    } finally {
      setToggling(null)
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">{t('users_title')}</div>
          <div className="page-sub">{t('users_sub')}</div>
        </div>
        <div className="row" style={{ flexWrap: 'wrap' }}>
          <input
            className="search-input"
            placeholder={t('users_search')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Button variant="primary" onClick={() => { setForm(empty); setErr(null); setCreating(true) }}>
            {t('users_add')}
          </Button>
        </div>
      </div>

      <div className="card">
        {list === null ? (
          <div className="stack">
            {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} height={52} />)}
          </div>
        ) : filtered && filtered.length === 0 ? (
          <Empty
            title={search ? t('users_no_match') : t('users_empty')}
            sub={search ? t('users_no_match_sub') : t('users_empty_sub')}
          />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>{t('users_col_user')}</th>
                  <th>{t('users_col_role')}</th>
                  <th>{t('users_col_status')}</th>
                  <th>{t('users_col_created')}</th>
                  <th className="actions">{t('users_col_actions')}</th>
                </tr>
              </thead>
              <tbody>
                {filtered!.map((u) => (
                  <tr key={u.id}>
                    <td>
                      <div className="row gap-2" style={{ alignItems: 'center' }}>
                        <UserInitials email={u.email} />
                        <div>
                          <div style={{ fontWeight: 500, fontSize: 13 }}>{u.email}</div>
                          {me?.id === u.id && (
                            <div className="text-xs text-mute">{t('users_you')}</div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td>
                      <Badge tone={roleTone(u.role)}>{u.role}</Badge>
                    </td>
                    <td>
                      <label className="toggle" title={u.disabled ? t('users_status_disabled') : t('users_status_active')}>
                        <input
                          type="checkbox"
                          checked={!u.disabled}
                          disabled={me?.id === u.id || toggling === u.id}
                          onChange={() => toggleDisabled(u)}
                        />
                        <span className="toggle-track" />
                        <span className="toggle-thumb" />
                      </label>
                    </td>
                    <td className="text-dim">{formatDate(u.created_at)}</td>
                    <td className="actions">
                      <div className="row-end">
                        <Button
                          size="sm"
                          variant="danger"
                          disabled={me?.id === u.id}
                          onClick={() => setConfirm(u)}
                        >
                          {t('users_delete')}
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
        onClose={() => { setCreating(false); setErr(null) }}
        title={t('users_modal_title')}
        footer={(
          <>
            <Button variant="ghost" onClick={() => { setCreating(false); setErr(null) }}>
              {t('action_cancel')}
            </Button>
            <Button variant="primary" onClick={submit as never} loading={busy}>
              {t('action_create')}
            </Button>
          </>
        )}
      >
        <form className="stack" onSubmit={submit}>
          <Input
            label={t('users_field_email')}
            type="email"
            placeholder={t('users_field_email_ph')}
            value={form.email}
            onChange={(e) => setForm({ ...form, email: e.target.value })}
            required
            autoFocus
          />
          <Input
            label={t('users_field_password')}
            type="password"
            placeholder={t('users_field_password_ph')}
            value={form.password}
            minLength={8}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
            required
          />
          <div className="stack-sm">
            <label className="label">{t('users_field_role')}</label>
            <select
              className="select"
              value={form.role}
              onChange={(e) => setForm({ ...form, role: e.target.value as Role })}
            >
              <option value="user">{t('users_role_user')}</option>
              <option value="operator">{t('users_role_operator')}</option>
              <option value="admin">{t('users_role_admin')}</option>
            </select>
          </div>
          {err && <div className="text-danger text-sm">{err}</div>}
        </form>
      </Modal>

      <ConfirmDialog
        open={!!confirm}
        title={t('users_delete_title')}
        body={(
          <>
            {t('users_delete_body')} <strong>{confirm?.email}</strong>
          </>
        )}
        confirmText={t('users_delete')}
        destructive
        loading={delBusy}
        onConfirm={remove}
        onClose={() => setConfirm(null)}
      />
    </div>
  )
}
