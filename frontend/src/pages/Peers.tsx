import { useEffect, useMemo, useState } from 'react'
import { api, ApiError } from '../api/client'
import type { CreatePeerResponse, Peer, Server } from '../types'
import {
  Button, ConfirmDialog, Empty, Input, Modal, Skeleton, toast,
} from '../components/ui'
import { useI18n } from '../i18n'
import { formatBytes, formatRelative } from '../lib/format'

/* ── Traffic bar ─────────────────────────────────────────── */
function TrafficBar({ rx, tx }: { rx: number; tx: number }) {
  const total = rx + tx
  // Soft cap at 50 GB for bar width
  const CAP = 50 * 1024 ** 3
  const pct = Math.min((total / CAP) * 100, 100)
  const tone = pct > 90 ? 'danger' : pct > 65 ? 'warn' : ''
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, minWidth: 100 }}>
      <div className="traffic-bar">
        <div className={`traffic-fill${tone ? ` ${tone}` : ''}`} style={{ width: `${pct}%` }} />
      </div>
      <div className="traffic-label">
        ↓{formatBytes(rx)} ↑{formatBytes(tx)}
      </div>
    </div>
  )
}

/* ── QR modal ────────────────────────────────────────────── */
function QRModal({ config, onClose }: { config: string; onClose: () => void }) {
  const { t } = useI18n()
  return (
    <Modal open onClose={onClose} title="QR-код" footer={
      <Button variant="primary" onClick={onClose}>{t('action_cancel')}</Button>
    }>
      <div style={{ textAlign: 'center', padding: '8px 0' }}>
        <img
          src={`https://api.qrserver.com/v1/create-qr-code/?data=${encodeURIComponent(config)}&size=260x260&color=ffffff&bgcolor=131325`}
          alt="QR"
          width={260}
          height={260}
          style={{ borderRadius: 8 }}
        />
        <div className="text-xs text-mute" style={{ marginTop: 8 }}>
          Отсканируйте в мобильном клиенте
        </div>
      </div>
    </Modal>
  )
}

/* ── Peers page ──────────────────────────────────────────── */
export default function Peers() {
  const { t } = useI18n()

  const [peers,      setPeers]      = useState<Peer[] | null>(null)
  const [servers,    setServers]    = useState<Server[]>([])
  const [search,     setSearch]     = useState('')
  const [creating,   setCreating]   = useState(false)
  const [name,       setName]       = useState('')
  const [serverID,   setServerID]   = useState('')
  const [createBusy, setCreateBusy] = useState(false)
  const [createErr,  setCreateErr]  = useState<string | null>(null)
  const [recent,     setRecent]     = useState<CreatePeerResponse | null>(null)
  const [confirm,    setConfirm]    = useState<Peer | null>(null)
  const [delBusy,    setDelBusy]    = useState(false)
  const [toggling,   setToggling]   = useState<string | null>(null)
  const [qrConfig,   setQrConfig]   = useState<string | null>(null)

  const readyServers = useMemo(
    () => servers.filter((s) => s.protocol && s.protocol !== 'none'),
    [servers],
  )

  const serverMap = useMemo(
    () => Object.fromEntries(servers.map((s) => [s.id, s])),
    [servers],
  )

  async function reload() {
    try {
      const [p, s] = await Promise.all([
        api.peers.list(),
        api.servers.list().catch(() => []),
      ])
      setPeers(p)
      setServers(s as Server[])
      if (!serverID && (s as Server[]).length) {
        const first = (s as Server[]).find((srv) => srv.protocol && srv.protocol !== 'none')
        if (first) setServerID(first.id)
      }
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'load failed')
    }
  }

  useEffect(() => { reload() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = useMemo(() => {
    if (!peers) return null
    const q = search.trim().toLowerCase()
    if (!q) return peers
    return peers.filter((p) =>
      p.name.toLowerCase().includes(q) ||
      (p.assigned_ip ?? '').toLowerCase().includes(q),
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
      toast.success(t('toast_peer_created'))
    } catch (err) {
      setCreateErr(err instanceof ApiError ? (err.code ?? `HTTP ${err.status}`) : 'create failed')
    } finally {
      setCreateBusy(false)
    }
  }

  async function remove() {
    if (!confirm) return
    setDelBusy(true)
    try {
      await api.peers.delete(confirm.id)
      toast.success(t('toast_peer_deleted'))
      setConfirm(null)
      await reload()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'delete failed')
    } finally {
      setDelBusy(false)
    }
  }

  async function togglePeer(p: Peer) {
    setToggling(p.id)
    try {
      await api.peers.toggle(p.id, !p.enabled)
      await reload()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'toggle failed')
    } finally {
      setToggling(null)
    }
  }

  async function downloadConfig(p: Peer) {
    try {
      const cfg = await api.peers.config(p.id)
      const blob = new Blob([cfg], { type: 'text/plain' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${p.name}.conf`
      a.click()
      URL.revokeObjectURL(url)
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'download failed')
    }
  }

  async function showQR(p: Peer) {
    try {
      const cfg = await api.peers.config(p.id)
      setQrConfig(cfg)
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'qr failed')
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">{t('peers_title')}</div>
          <div className="page-sub">{t('peers_sub')}</div>
        </div>
        <div className="row" style={{ flexWrap: 'wrap' }}>
          <input
            className="search-input"
            placeholder={t('peers_search')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Button variant="primary" onClick={() => { setName(''); setCreateErr(null); setCreating(true) }}>
            {t('peers_add')}
          </Button>
        </div>
      </div>

      {/* Recent peer config */}
      {recent && (
        <div className="card mb-4">
          <div className="card-header">
            <div>
              <div className="card-title">{recent.peer.name}</div>
              <div className="card-sub text-success">Клиент создан — скачайте конфиг или отсканируйте QR</div>
            </div>
            <div className="row" style={{ gap: 8 }}>
              <Button onClick={() => {
                const blob = new Blob([recent.config], { type: 'text/plain' })
                const url = URL.createObjectURL(blob)
                const a = document.createElement('a')
                a.href = url; a.download = `${recent.peer.name}.conf`; a.click()
                URL.revokeObjectURL(url)
              }}>{t('peers_download')}</Button>
              <Button onClick={() => setQrConfig(recent.config)}>QR</Button>
              <Button variant="ghost" onClick={() => setRecent(null)}>{t('action_cancel')}</Button>
            </div>
          </div>
          <pre style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-dim)', overflowX: 'auto', padding: '8px 0', margin: 0 }}>
            {recent.config}
          </pre>
        </div>
      )}

      <div className="card">
        {peers === null ? (
          <div className="stack">
            {Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} height={52} />)}
          </div>
        ) : filtered && filtered.length === 0 ? (
          <Empty
            title={search ? t('peers_no_match') : t('peers_empty')}
            sub={search ? t('peers_no_match_sub') : t('peers_empty_sub')}
            action={!search ? (
              <Button variant="primary" onClick={() => setCreating(true)}>{t('peers_add')}</Button>
            ) : undefined}
          />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>{t('peers_col_client')}</th>
                  <th>{t('peers_col_server')}</th>
                  <th>{t('peers_col_traffic')}</th>
                  <th>{t('peers_col_status')}</th>
                  <th>{t('peers_col_last_seen')}</th>
                  <th className="actions">{t('peers_col_actions')}</th>
                </tr>
              </thead>
              <tbody>
                {filtered!.map((p) => {
                  const srv = serverMap[p.server_id]
                  return (
                    <tr key={p.id}>
                      <td>
                        <div style={{ fontWeight: 500 }}>{p.name}</div>
                        {p.assigned_ip && (
                          <div className="text-xs text-mute" style={{ fontFamily: 'var(--font-mono)' }}>
                            {p.assigned_ip}
                          </div>
                        )}
                      </td>
                      <td>
                        <div className="text-sm">{srv?.name ?? '—'}</div>
                        {srv?.protocol && srv.protocol !== 'none' && (
                          <div className="text-xs text-mute">{srv.protocol}</div>
                        )}
                      </td>
                      <td>
                        <TrafficBar rx={p.bytes_rx} tx={p.bytes_tx} />
                      </td>
                      <td>
                        <label className="toggle">
                          <input
                            type="checkbox"
                            checked={p.enabled}
                            disabled={toggling === p.id}
                            onChange={() => togglePeer(p)}
                          />
                          <span className="toggle-track" />
                          <span className="toggle-thumb" />
                        </label>
                      </td>
                      <td className="text-dim text-sm">
                        {formatRelative(p.last_handshake)}
                      </td>
                      <td className="actions">
                        <div className="row-end" style={{ gap: 4 }}>
                          <Button size="sm" onClick={() => downloadConfig(p)} title={t('peers_download')}>↓</Button>
                          <Button size="sm" onClick={() => showQR(p)} title="QR">QR</Button>
                          <Button size="sm" variant="danger" onClick={() => setConfirm(p)}>
                            {t('peers_delete')}
                          </Button>
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

      {/* Create modal */}
      <Modal
        open={creating}
        onClose={() => { setCreating(false); setCreateErr(null) }}
        title={t('peers_modal_title')}
        footer={(
          <>
            <Button variant="ghost" onClick={() => { setCreating(false); setCreateErr(null) }}>
              {t('action_cancel')}
            </Button>
            <Button variant="primary" onClick={create as never} loading={createBusy}>
              {t('action_create')}
            </Button>
          </>
        )}
      >
        <form className="stack" onSubmit={create}>
          <Input
            label={t('peers_field_name')}
            placeholder={t('peers_field_name_ph')}
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            autoFocus
          />
          {readyServers.length === 0 ? (
            <div className="state-panel state-warn">
              <strong>{t('peers_no_servers')}</strong>
              <div className="text-sm text-mute">{t('peers_no_servers_sub')}</div>
            </div>
          ) : (
            <div className="stack-sm">
              <label className="label">{t('peers_field_server')}</label>
              <select
                className="select"
                value={serverID}
                onChange={(e) => setServerID(e.target.value)}
              >
                {readyServers.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name} · {s.protocol}
                  </option>
                ))}
              </select>
            </div>
          )}
          {createErr && <div className="text-danger text-sm">{createErr}</div>}
        </form>
      </Modal>

      {/* Delete confirm */}
      <ConfirmDialog
        open={!!confirm}
        title={t('peers_delete_title')}
        body={(
          <>
            {t('peers_delete_body')} <strong>{confirm?.name}</strong>
          </>
        )}
        confirmText={t('peers_delete')}
        destructive
        loading={delBusy}
        onConfirm={remove}
        onClose={() => setConfirm(null)}
      />

      {qrConfig && <QRModal config={qrConfig} onClose={() => setQrConfig(null)} />}
    </div>
  )
}
