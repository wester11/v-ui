import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import type { Peer, Server } from '../types'
import { Badge, Button, CopyField, Empty, Skeleton, StatusDot, toast } from '../components/ui'
import { formatBytes, formatRelative } from '../lib/format'

export default function ServerDetail() {
  const { id } = useParams<{ id: string }>()
  const [server, setServer] = useState<Server | null>(null)
  const [peers, setPeers] = useState<Peer[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [redeploying, setRedeploying] = useState(false)

  async function load() {
    if (!id) return
    try {
      setError(null)
      const list = await api.servers.list()
      const s = list.find((x) => x.id === id)
      if (!s) {
        setError('Server not found')
        return
      }
      setServer(s)
      const allPeers = await api.peers.list().catch(() => [] as Peer[])
      setPeers(allPeers.filter((p) => p.server_id === id))
    } catch (e) {
      setError(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'load failed')
    }
  }

  useEffect(() => { load(); /* eslint-disable-next-line react-hooks/exhaustive-deps */ }, [id])

  const summary = useMemo(() => {
    if (!peers) return null
    const rx = peers.reduce((a, p) => a + p.bytes_rx, 0)
    const tx = peers.reduce((a, p) => a + p.bytes_tx, 0)
    const enabled = peers.filter((p) => p.enabled).length
    return { rx, tx, enabled, total: peers.length }
  }, [peers])

  const redeploy = async () => {
    if (!server || server.protocol !== 'xray') return
    setRedeploying(true)
    try {
      await api.fleet.redeployServer(server.id)
      toast.success('Redeploy requested')
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'redeploy failed')
    } finally {
      setRedeploying(false)
    }
  }

  if (error) {
    return (
      <div className="page">
        <Empty title="Cannot load server" sub={error} action={<Link to="/servers"><Button>Back to servers</Button></Link>} />
      </div>
    )
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="text-sm"><Link to="/servers" className="text-dim">Back to servers</Link></div>
          <div className="page-title">{server?.name ?? <Skeleton width={180} height={28} />}</div>
          <div className="page-sub">{server?.endpoint}</div>
        </div>
        <div className="row">
          {server?.protocol === 'xray' && (
            <Button onClick={redeploy} loading={redeploying} title="Rebuild and push full Xray config">
              Redeploy
            </Button>
          )}
          <Button onClick={load}>Refresh</Button>
        </div>
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-label">Status</div>
          <div className="stat-value">{server ? <StatusDot online={server.online} /> : <Skeleton height={28} width={120} />}</div>
          <div className="stat-meta">{server ? `Last heartbeat: ${formatRelative(server.last_heartbeat)}` : ''}</div>
        </div>
        <div className="stat-card violet">
          <div className="stat-label">Mode</div>
          <div className="stat-value">{server?.mode === 'cascade' ? 'Cascade' : 'Standalone'}</div>
          <div className="stat-meta">{server?.protocol}</div>
        </div>
        <div className="stat-card success">
          <div className="stat-label">Peers</div>
          <div className="stat-value">{summary ? summary.total : <Skeleton height={28} width={60} />}</div>
          <div className="stat-meta">{summary ? `${summary.enabled} enabled` : ''}</div>
        </div>
        <div className="stat-card warn">
          <div className="stat-label">Traffic</div>
          <div className="stat-value">{summary ? formatBytes(summary.rx + summary.tx) : '...'}</div>
          <div className="stat-meta">{summary ? `RX ${formatBytes(summary.rx)} / TX ${formatBytes(summary.tx)}` : ''}</div>
        </div>
      </div>

      <div className="card mb-4">
        <div className="card-header">
          <div>
            <div className="card-title">Configuration</div>
            <div className="card-sub">Read-only node details.</div>
          </div>
        </div>
        {server ? (
          <div className="stack">
            <Field label="Protocol"><Badge tone="info">{server.protocol}</Badge></Field>
            <Field label="Mode">{server.mode === 'cascade' ? <Badge tone="violet">cascade</Badge> : <Badge>standalone</Badge>}</Field>
            <Field label="Endpoint"><code>{server.endpoint}</code></Field>
            <Field label="Listen port"><code>{server.listen_port ?? server.xray_inbound_port ?? '-'}</code></Field>
            <Field label="Subnet"><code>{server.subnet ?? '-'}</code></Field>
            {server.mode === 'cascade' && (
              <Field label="Cascade upstream ID"><CopyField value={server.cascade_upstream_id ?? '-'} /></Field>
            )}
            <Field label="Public key"><CopyField value={server.public_key ?? server.xray_public_key ?? ''} /></Field>
            <Field label="Server ID"><CopyField value={server.id} /></Field>
          </div>
        ) : (
          <div className="stack">{Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} height={20} />)}</div>
        )}
      </div>

      <div className="card">
        <div className="card-header">
          <div>
            <div className="card-title">Clients on this server</div>
            <div className="card-sub">Current client list and traffic summary.</div>
          </div>
          <Link to="/clients"><Button>Manage clients</Button></Link>
        </div>
        {!peers ? (
          <div className="stack">{Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} height={36} />)}</div>
        ) : peers.length === 0 ? (
          <Empty title="No clients on this server" sub="Provision a client from Clients page." />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>IP</th>
                  <th>Status</th>
                  <th>Last handshake</th>
                  <th>RX / TX</th>
                </tr>
              </thead>
              <tbody>
                {peers.map((p) => (
                  <tr key={p.id}>
                    <td><strong>{p.name}</strong></td>
                    <td><code>{p.assigned_ip ?? '-'}</code></td>
                    <td>{p.enabled ? <Badge tone="success">enabled</Badge> : <Badge tone="warn">disabled</Badge>}</td>
                    <td className="text-dim">{formatRelative(p.last_handshake)}</td>
                    <td className="text-mono text-dim">RX {formatBytes(p.bytes_rx)} / TX {formatBytes(p.bytes_tx)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="row" style={{ alignItems: 'center' }}>
      <div className="text-mute text-sm" style={{ width: 180, flexShrink: 0 }}>{label}</div>
      <div style={{ flex: 1 }}>{children}</div>
    </div>
  )
}
