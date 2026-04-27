import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import type { CreateServerResponse, NodeCheckResult, Server } from '../types'
import {
  Badge, Button, ConfirmDialog, CopyField, Empty, Input, Modal, Skeleton, StatusDot, toast,
} from '../components/ui'
import { formatRelative } from '../lib/format'

interface FormState {
  name: string
  endpoint: string
}

const empty: FormState = { name: '', endpoint: '' }

export default function Servers() {
  const [list, setList] = useState<Server[] | null>(null)
  const [search, setSearch] = useState('')
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState<FormState>(empty)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const [onboarding, setOnboarding] = useState<CreateServerResponse | null>(null)
  const [checkResult, setCheckResult] = useState<NodeCheckResult | null>(null)
  const [checkLoading, setCheckLoading] = useState(false)
  const [confirm, setConfirm] = useState<Server | null>(null)
  const [delBusy, setDelBusy] = useState(false)

  async function reload() {
    try {
      setList(await api.servers.list())
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'load failed')
    }
  }

  useEffect(() => { reload() }, [])

  const filtered = useMemo(() => {
    if (!list) return null
    const q = search.trim().toLowerCase()
    if (!q) return list
    return list.filter((s) =>
      s.name.toLowerCase().includes(q) ||
      s.endpoint.toLowerCase().includes(q) ||
      (s.ip ?? '').toLowerCase().includes(q),
    )
  }, [list, search])

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setErr(null)
    setBusy(true)
    try {
      const resp = await api.servers.create({
        name: form.name.trim(),
        endpoint: form.endpoint.trim(),
      })
      setOnboarding(resp)
      setCheckResult(null)
      setForm(empty)
      setCreating(false)
      await reload()
      toast.success('Server created')
    } catch (e) {
      setErr(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'create failed')
    } finally {
      setBusy(false)
    }
  }

  async function checkConnection(serverID: string) {
    setCheckLoading(true)
    try {
      const res = await api.servers.check(serverID)
      setCheckResult(res)
      if (res.status === 'online') toast.success('Node is online')
      else toast.warn('Node is still offline')
      await reload()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'check failed')
    } finally {
      setCheckLoading(false)
    }
  }

  async function remove() {
    if (!confirm) return
    setDelBusy(true)
    try {
      await api.servers.delete(confirm.id)
      toast.success('Server removed')
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
          <div className="page-title">Servers</div>
          <div className="page-sub">Add infrastructure nodes first. VPN configs are created separately in Configs.</div>
        </div>
        <div className="row" style={{ flexWrap: 'wrap' }}>
          <input
            className="search-input"
            placeholder="Search servers..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Button variant="primary" onClick={() => setCreating(true)}>+ Add server</Button>
        </div>
      </div>

      <div className="card">
        {list === null ? (
          <div className="stack">
            {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} height={42} />)}
          </div>
        ) : filtered && filtered.length === 0 ? (
          <Empty
            title={search ? 'No servers match your search' : 'No servers yet'}
            sub={search ? 'Try a different query.' : 'Create your first node to start onboarding.'}
            action={!search ? <Button variant="primary" onClick={() => setCreating(true)}>+ Add server</Button> : undefined}
          />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Node ID</th>
                  <th>Endpoint</th>
                  <th>Status</th>
                  <th>Version</th>
                  <th>Last heartbeat</th>
                  <th className="actions">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered!.map((s) => (
                  <tr key={s.id}>
                    <td><Link to={`/servers/${s.id}`}><strong>{s.name}</strong></Link></td>
                    <td><code>{s.node_id.slice(0, 8)}...</code></td>
                    <td><code>{s.ip || s.endpoint}</code></td>
                    <td>
                      <div className="row" style={{ gap: 8 }}>
                        <StatusDot online={s.online} />
                        <Badge tone={s.status === 'online' ? 'success' : s.status === 'pending' ? 'warn' : 'danger'}>
                          {s.status}
                        </Badge>
                      </div>
                    </td>
                    <td><code>{s.agent_version || '-'}</code></td>
                    <td className="text-dim">{formatRelative(s.last_heartbeat)}</td>
                    <td className="actions">
                      <div className="row-end">
                        <Link to={`/servers/${s.id}`}><Button size="sm">Open</Button></Link>
                        <Button size="sm" onClick={() => checkConnection(s.id)} loading={checkLoading}>Check</Button>
                        <Button size="sm" variant="danger" onClick={() => setConfirm(s)}>Remove</Button>
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
        title="Add infrastructure server"
        footer={(
          <>
            <Button variant="ghost" onClick={() => setCreating(false)}>Cancel</Button>
            <Button variant="primary" onClick={submit as never} loading={busy}>Create</Button>
          </>
        )}
      >
        <form className="stack" onSubmit={submit}>
          <Input
            label="Name"
            placeholder="eu-1"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            required
            autoFocus
          />
          <Input
            label="IP or hostname"
            placeholder="2.27.32.141 or eu-1.example.com"
            value={form.endpoint}
            onChange={(e) => setForm({ ...form, endpoint: e.target.value })}
            required
          />
          {err && <div className="text-danger text-sm">{err}</div>}
        </form>
      </Modal>

      {onboarding && (
        <Modal
          open
          onClose={() => setOnboarding(null)}
          title={`Node onboarding: ${onboarding.name}`}
          footer={(
            <>
              <Button onClick={() => checkConnection(onboarding.id)} loading={checkLoading}>Check connection</Button>
              <Button variant="primary" onClick={() => setOnboarding(null)}>Done</Button>
            </>
          )}
        >
          <div className="stack">
            <div className="state-panel">
              <div className="stack-sm">
                <div><strong>Step 1</strong>: copy command</div>
                <CopyField value={onboarding.install_command} />
              </div>
              <div className="stack-sm">
                <div><strong>Step 2</strong>: run on target VPS</div>
                <pre>{onboarding.compose_snippet}</pre>
              </div>
              <div className="stack-sm">
                <div><strong>Step 3</strong>: click check connection</div>
                {checkResult ? (
                  <div className="text-sm">
                    Status: <Badge tone={checkResult.status === 'online' ? 'success' : 'warn'}>{checkResult.status}</Badge>
                  </div>
                ) : (
                  <div className="text-sm text-mute">Waiting for first connection...</div>
                )}
              </div>
            </div>
            <div className="text-xs text-mute">
              After node is online, go to <Link to="/configs">Configs</Link> and create VPN logic.
            </div>
          </div>
        </Modal>
      )}

      <ConfirmDialog
        open={!!confirm}
        title="Remove server?"
        body={(
          <>
            This will remove server <strong>{confirm?.name}</strong> and associated configs.
          </>
        )}
        confirmText="Remove"
        destructive
        loading={delBusy}
        onConfirm={remove}
        onClose={() => setConfirm(null)}
      />
    </div>
  )
}

