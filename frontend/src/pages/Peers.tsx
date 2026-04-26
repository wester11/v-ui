import { useEffect, useMemo, useState } from 'react'
import { QRCodeSVG } from 'qrcode.react'
import { api, ApiError } from '../api/client'
import type { CreatePeerResponse, Peer, Server } from '../types'
import {
  Badge, Button, ConfirmDialog, Empty, IconButton, Input, Modal, Select,
  Skeleton, copyToClipboard, downloadFile, toast,
} from '../components/ui'
import { formatBytes, formatRelative, maskKey } from '../lib/format'

export default function Peers() {
  const [peers, setPeers] = useState<Peer[] | null>(null)
  const [servers, setServers] = useState<Server[]>([])
  const [search, setSearch] = useState('')
  const [creating, setCreating] = useState(false)
  const [name, setName] = useState('')
  const [serverID, setServerID] = useState('')
  const [createBusy, setCreateBusy] = useState(false)
  const [createErr, setCreateErr] = useState<string | null>(null)
  const [recent, setRecent] = useState<CreatePeerResponse | null>(null)
  const [confirm, setConfirm] = useState<Peer | null>(null)
  const [busy, setBusy] = useState(false)
  const [view, setView] = useState<{ peer: Peer; config: string } | null>(null)

  async function reload() {
    try {
      const [p, s] = await Promise.all([api.peers.list(), api.servers.list().catch(() => [])])
      setPeers(p)
      setServers(s as Server[])
      if (!serverID && (s as Server[]).length) setServerID((s as Server[])[0].id)
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'load failed')
    }
  }

  useEffect(() => { reload() /* eslint-disable-next-line react-hooks/exhaustive-deps */ }, [])

  const filtered = useMemo(() => {
    if (!peers) return null
    const q = search.trim().toLowerCase()
    if (!q) return peers
    return peers.filter((p) =>
      p.name.toLowerCase().includes(q) ||
      p.assigned_ip.toLowerCase().includes(q) ||
      p.public_key.toLowerCase().includes(q),
    )
  }, [peers, search])

  async function create(e: React.FormEvent) {
    e.preventDefault()
    if (!serverID || !name.trim()) return
    setCreateErr(null)
    setCreateBusy(true)
    try {
      const resp = await api.peers.create(serverID, name.trim())
      setRecent(resp)
      setName('')
      setCreating(false)
      await reload()
      toast.success('Peer created')
    } catch (err) {
      setCreateErr(err instanceof ApiError ? (err.code ?? `HTTP ${err.status}`) : 'create failed')
    } finally {
      setCreateBusy(false)
    }
  }

  async function revoke() {
    if (!confirm) return
    setBusy(true)
    try {
      await api.peers.delete(confirm.id)
      toast.success('Peer revoked')
      setConfirm(null)
      await reload()
    } catch (err) {
      toast.error(err instanceof ApiError ? (err.code ?? `HTTP ${err.status}`) : 'delete failed')
    } finally {
      setBusy(false)
    }
  }

  async function showConfig(p: Peer) {
    try {
      const cfg = await api.peers.config(p.id)
      setView({ peer: p, config: cfg })
    } catch (err) {
      toast.error(err instanceof ApiError ? (err.code ?? `HTTP ${err.status}`) : 'load failed')
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">Peers</div>
          <div className="page-sub">WireGuard peers provisioned for your account.</div>
        </div>
        <div className="row">
          <input
            className="search-input"
            placeholder="Search by name, IP, key…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Button variant="primary" onClick={() => setCreating(true)} disabled={!servers.length}>
            + New peer
          </Button>
        </div>
      </div>

      {recent && (
        <div className="card mb-4" style={{ borderColor: 'rgba(63,185,80,0.3)' }}>
          <div className="card-header">
            <div>
              <div className="card-title">✓ Peer "{recent.peer.name}" provisioned</div>
              <div className="card-sub">Save the config — the private key is shown only once.</div>
            </div>
            <div className="row">
              <Button onClick={() => copyToClipboard(recent.config, 'Config copied')}>⧉ Copy</Button>
              <Button
                variant="primary"
                onClick={() => downloadFile(`${recent.peer.name}.conf`, recent.config)}
              >
                ↓ Download .conf
              </Button>
              <IconButton onClick={() => setRecent(null)} title="Dismiss">✕</IconButton>
            </div>
          </div>
          <div className="row" style={{ alignItems: 'flex-start' }}>
            <div className="qr-box">
              <QRCodeSVG value={recent.config} size={180} level="M" />
            </div>
            <pre style={{ flex: 1 }}>{recent.config}</pre>
          </div>
        </div>
      )}

      <div className="card">
        {peers === null ? (
          <div className="stack">
            {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} height={42} />)}
          </div>
        ) : filtered && filtered.length === 0 ? (
          <Empty
            title={search ? 'No peers match your search' : 'No peers yet'}
            sub={search ? 'Try a different query.' : 'Provision your first peer to get a wg-quick config.'}
            action={!search && servers.length > 0 ? (
              <Button variant="primary" onClick={() => setCreating(true)}>+ New peer</Button>
            ) : undefined}
          />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>IP</th>
                  <th>Server</th>
                  <th>Status</th>
                  <th>Last handshake</th>
                  <th>RX / TX</th>
                  <th>Public key</th>
                  <th className="actions">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered!.map((p) => {
                  const srv = servers.find((s) => s.id === p.server_id)
                  return (
                    <tr key={p.id}>
                      <td><strong>{p.name}</strong></td>
                      <td><code>{p.assigned_ip}</code></td>
                      <td>{srv?.name ?? <code>{maskKey(p.server_id)}</code>}</td>
                      <td>
                        {p.enabled ? <Badge tone="success">enabled</Badge> : <Badge tone="warn">disabled</Badge>}
                      </td>
                      <td className="text-dim">{formatRelative(p.last_handshake)}</td>
                      <td className="text-mono text-dim">
                        ↓ {formatBytes(p.bytes_rx)} · ↑ {formatBytes(p.bytes_tx)}
                      </td>
                      <td><code>{maskKey(p.public_key)}</code></td>
                      <td className="actions">
                        <div className="row-end">
                          <Button size="sm" onClick={() => showConfig(p)}>Config</Button>
                          <Button size="sm" variant="danger" onClick={() => setConfirm(p)}>Revoke</Button>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Create peer modal */}
      <Modal
        open={creating}
        onClose={() => setCreating(false)}
        title="New peer"
        footer={
          <>
            <Button variant="ghost" onClick={() => setCreating(false)}>Cancel</Button>
            <Button variant="primary" onClick={create as never} loading={createBusy}>
              Provision
            </Button>
          </>
        }
      >
        <form className="stack" onSubmit={create}>
          <Input
            label="Name"
            placeholder="my-iphone"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            autoFocus
          />
          <Select
            label="Server"
            value={serverID}
            onChange={(e) => setServerID(e.target.value)}
            disabled={!servers.length}
          >
            {servers.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name} — {s.endpoint}
              </option>
            ))}
          </Select>
          {createErr && <div className="text-danger text-sm">{createErr}</div>}
          {!servers.length && <div className="text-warn text-sm">No servers available. Register one first.</div>}
        </form>
      </Modal>

      {/* View config modal */}
      {view && (
        <Modal
          open
          onClose={() => setView(null)}
          title={`${view.peer.name} — wg-quick config`}
          footer={
            <>
              <Button onClick={() => copyToClipboard(view.config, 'Config copied')}>⧉ Copy</Button>
              <Button
                variant="primary"
                onClick={() => downloadFile(`${view.peer.name}.conf`, view.config)}
              >
                ↓ Download
              </Button>
            </>
          }
        >
          <div className="stack">
            <div className="row" style={{ justifyContent: 'center' }}>
              <div className="qr-box">
                <QRCodeSVG value={view.config} size={220} level="M" />
              </div>
            </div>
            <pre>{view.config}</pre>
          </div>
        </Modal>
      )}

      <ConfirmDialog
        open={!!confirm}
        title="Revoke peer?"
        body={
          <>This will remove peer <strong>{confirm?.name}</strong> from the server. This action cannot be undone.</>
        }
        confirmText="Revoke"
        destructive
        loading={busy}
        onConfirm={revoke}
        onClose={() => setConfirm(null)}
      />
    </div>
  )
}
