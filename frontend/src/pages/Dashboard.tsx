import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import { useAuth } from '../store/auth'
import type { Server, Stats } from '../types'
import { Badge, Button, Empty, SkeletonRows, StatCard, StatusDot } from '../components/ui'
import { formatBytes, formatNumber, formatRelative } from '../lib/format'

export default function Dashboard() {
  const user = useAuth((s) => s.user)
  const isStaff = user?.role === 'admin' || user?.role === 'operator'

  const [stats, setStats] = useState<Stats | null>(null)
  const [servers, setServers] = useState<Server[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const load = async () => {
    setError(null)
    setLoading(true)
    try {
      const tasks: Promise<unknown>[] = []
      if (isStaff) {
        tasks.push(api.stats.get().then(setStats))
        tasks.push(api.servers.list().then(setServers))
      }
      await Promise.all(tasks)
    } catch (e) {
      if (e instanceof ApiError) setError(e.code ?? `HTTP ${e.status}`)
      else setError('failed to load dashboard')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    if (!isStaff) return
    const t = setInterval(load, 15_000)
    return () => clearInterval(t)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isStaff])

  const degraded = useMemo(() => {
    if (!servers) return 0
    return servers.filter((s) => s.online === false).length
  }, [servers])

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">Control Center</div>
          <div className="page-sub">Operational overview of your VPN fleet and clients.</div>
        </div>
        <Button variant="ghost" onClick={load} disabled={loading}>Refresh</Button>
      </div>

      {error && (
        <div className="state-panel state-error mb-4">
          <strong>Dashboard unavailable:</strong> {error}
        </div>
      )}

      {!isStaff ? (
        <div className="card">
          <div className="card-header">
            <div>
              <div className="card-title">Quick actions</div>
              <div className="card-sub">Most used controls for regular users.</div>
            </div>
          </div>
          <div className="row" style={{ flexWrap: 'wrap' }}>
            <Link to="/clients"><Button variant="primary">Manage my clients</Button></Link>
            <Link to="/configs"><Button>Open configs</Button></Link>
            <Link to="/profile"><Button>Security profile</Button></Link>
          </div>
        </div>
      ) : (
        <>
          <div className="stats-grid">
            <StatCard label="Users" value={loading ? '...' : formatNumber(stats?.users ?? 0)} />
            <StatCard label="Clients" value={loading ? '...' : formatNumber(stats?.peers ?? 0)} tone="violet" />
            <StatCard
              label="Servers online"
              value={loading ? '...' : `${stats?.servers_online ?? 0}/${stats?.servers ?? 0}`}
              meta={loading ? undefined : degraded > 0 ? `${degraded} degraded/offline` : 'all healthy'}
              tone={degraded > 0 ? 'warn' : 'success'}
            />
            <StatCard
              label="Traffic"
              value={loading ? '...' : formatBytes((stats?.bytes_rx_total ?? 0) + (stats?.bytes_tx_total ?? 0))}
              meta={loading ? undefined : `RX ${formatBytes(stats?.bytes_rx_total ?? 0)} / TX ${formatBytes(stats?.bytes_tx_total ?? 0)}`}
              tone="success"
            />
          </div>

          <div className="card">
            <div className="card-header">
              <div>
                <div className="card-title">Server status</div>
                <div className="card-sub">Heartbeat, mode and protocol status for each node.</div>
              </div>
              <div className="row">
                <Link to="/settings"><Button>Fleet ops</Button></Link>
                <Link to="/servers"><Button variant="primary">Manage servers</Button></Link>
              </div>
            </div>
            {loading ? (
              <SkeletonRows rows={5} />
            ) : !servers || servers.length === 0 ? (
              <Empty
                title="No servers yet"
                sub="Register your first VPN node to start provisioning clients."
                action={<Link to="/servers"><Button variant="primary">Register server</Button></Link>}
              />
            ) : (
              <div className="table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Protocol</th>
                      <th>Mode</th>
                      <th>Endpoint</th>
                      <th>Status</th>
                      <th>Last heartbeat</th>
                      <th>Node ID</th>
                    </tr>
                  </thead>
                  <tbody>
                    {servers.map((s) => (
                      <tr key={s.id}>
                        <td><Link to={`/servers/${s.id}`}><strong>{s.name}</strong></Link></td>
                        <td><Badge tone="info">{s.protocol}</Badge></td>
                        <td>
                          {s.mode === 'cascade'
                            ? <Badge tone="violet">cascade</Badge>
                            : <Badge>standalone</Badge>}
                        </td>
                        <td><code>{s.endpoint}</code></td>
                        <td><StatusDot online={s.online} /></td>
                        <td className="text-dim">{formatRelative(s.last_heartbeat)}</td>
                        <td><code>{s.node_id.slice(0, 8)}...</code></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}
