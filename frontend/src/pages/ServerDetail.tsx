import { useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import type { Config, Server } from '../types'
import { Badge, Button, CopyField, Empty, Skeleton, StatusDot, toast } from '../components/ui'
import { formatRelative } from '../lib/format'

export default function ServerDetail() {
  const { id } = useParams<{ id: string }>()
  const [server, setServer] = useState<Server | null>(null)
  const [configs, setConfigs] = useState<Config[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [checking, setChecking] = useState(false)

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
      setConfigs(await api.configs.listByServer(s.id))
    } catch (e) {
      setError(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'load failed')
    }
  }

  useEffect(() => { load(); /* eslint-disable-next-line react-hooks/exhaustive-deps */ }, [id])

  const check = async () => {
    if (!server) return
    setChecking(true)
    try {
      await api.servers.check(server.id)
      toast.success('Connection check completed')
      await load()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'check failed')
    } finally {
      setChecking(false)
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
          <Button onClick={check} loading={checking}>Check connection</Button>
          <Button onClick={load}>Refresh</Button>
        </div>
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-label">Node status</div>
          <div className="stat-value">{server ? <StatusDot online={server.online} /> : <Skeleton height={28} width={120} />}</div>
          <div className="stat-meta">{server ? `Last heartbeat: ${formatRelative(server.last_heartbeat)}` : ''}</div>
        </div>
        <div className="stat-card violet">
          <div className="stat-label">Endpoint</div>
          <div className="stat-value">{server?.ip || '-'}</div>
          <div className="stat-meta">{server?.hostname || 'hostname not reported'}</div>
        </div>
        <div className="stat-card success">
          <div className="stat-label">Active protocol</div>
          <div className="stat-value">{server?.protocol || 'none'}</div>
          <div className="stat-meta">{server?.mode || '-'}</div>
        </div>
        <div className="stat-card warn">
          <div className="stat-label">Agent version</div>
          <div className="stat-value">{server?.agent_version || '-'}</div>
          <div className="stat-meta">{server?.status || 'pending'}</div>
        </div>
      </div>

      <div className="card mb-4">
        <div className="card-header">
          <div>
            <div className="card-title">Node identity</div>
            <div className="card-sub">Use Node ID for troubleshooting and onboarding.</div>
          </div>
        </div>
        {server ? (
          <div className="stack">
            <Field label="Node ID"><CopyField value={server.node_id} /></Field>
            <Field label="Server ID"><CopyField value={server.id} /></Field>
            <Field label="Status"><Badge tone={server.status === 'online' ? 'success' : server.status === 'pending' ? 'warn' : 'danger'}>{server.status}</Badge></Field>
          </div>
        ) : (
          <div className="stack">{Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} height={20} />)}</div>
        )}
      </div>

      <div className="card">
        <div className="card-header">
          <div>
            <div className="card-title">Configs on this server</div>
            <div className="card-sub">Separated VPN logic attached to this infrastructure node.</div>
          </div>
          <Link to="/configs"><Button>Create or manage configs</Button></Link>
        </div>
        {!configs ? (
          <div className="stack">{Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} height={36} />)}</div>
        ) : configs.length === 0 ? (
          <Empty title="No configs yet" sub="Create config from Configs page." />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Template</th>
                  <th>Mode</th>
                  <th>Routing</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {configs.map((cfg) => (
                  <tr key={cfg.id}>
                    <td><strong>{cfg.name}</strong></td>
                    <td>{cfg.template}</td>
                    <td>{cfg.setup_mode}</td>
                    <td>{cfg.routing_mode}</td>
                    <td>{cfg.is_active ? <Badge tone="success">active</Badge> : <Badge tone="warn">inactive</Badge>}</td>
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

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="row" style={{ alignItems: 'center' }}>
      <div className="text-mute text-sm" style={{ width: 180, flexShrink: 0 }}>{label}</div>
      <div style={{ flex: 1 }}>{children}</div>
    </div>
  )
}
