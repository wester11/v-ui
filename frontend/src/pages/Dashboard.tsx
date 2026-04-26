import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import { useAuth } from '../store/auth'
import type { Server, Stats } from '../types'
import { Badge, Button, Empty, SkeletonRows, StatCard, StatusDot } from '../components/ui'
import { formatBytes, formatNumber, formatRelative, maskKey } from '../lib/format'

export default function Dashboard() {
  const user = useAuth((s) => s.user)
  const isStaff = user?.role === 'admin' || user?.role === 'operator'

  const [stats, setStats] = useState<Stats | null>(null)
  const [servers, setServers] = useState<Server[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const load = async () => {
    setError(null)
    try {
      const tasks: Promise<unknown>[] = []
      if (isStaff) {
        tasks.push(api.stats.get().then(setStats))
        tasks.push(api.servers.list().then(setServers))
      }
      await Promise.all(tasks)
    } catch (e) {
      if (e instanceof ApiError) setError(e.code ?? `HTTP ${e.status}`)
      else setError('failed to load')
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

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">Welcome back{user?.email ? `, ${user.email.split('@')[0]}` : ''}</div>
          <div className="page-sub">Overview of your VPN fleet.</div>
        </div>
        <Button variant="ghost" onClick={load} disabled={loading}>↻ Refresh</Button>
      </div>

      {error && (
        <div className="card mb-4" style={{ borderColor: 'rgba(248,81,73,0.3)' }}>
          <span className="text-danger">Error: {error}</span>
        </div>
      )}

      {!isStaff ? (
        <div className="card">
          <div className="card-header"><div className="card-title">Quick links</div></div>
          <div className="row">
            <Link to="/peers"><Button variant="primary">My peers</Button></Link>
            <Link to="/profile"><Button>Account</Button></Link>
          </div>
        </div>
      ) : (
        <>
          <div className="stats-grid">
            <StatCard label="Users" value={loading ? '—' : formatNumber(stats?.users ?? 0)} />
            <StatCard label="Peers" value={loading ? '—' : formatNumber(stats?.peers ?? 0)} tone="violet" />
            <StatCard
              label="Servers"
              value={loading ? '—' : formatNumber(stats?.servers ?? 0)}
              meta={stats ? `${stats.servers_online}/${stats.servers} online` : undefined}
              tone={stats && stats.servers > 0 && stats.servers_online === stats.servers ? 'success' : 'warn'}
            />
            <StatCard
              label="Total RX"
              value={loading ? '—' : formatBytes(stats?.bytes_rx_total ?? 0)}
              meta={stats ? `TX: ${formatBytes(stats.bytes_tx_total)}` : undefined}
              tone="success"
            />
          </div>

          <div className="card">
            <div className="card-header">
              <div>
                <div className="card-title">Servers</div>
                <div className="card-sub">Status across the fleet</div>
              </div>
              <Link to="/servers"><Button>Manage</Button></Link>
            </div>
            {loading ? (
              <SkeletonRows rows={4} />
            ) : !servers || servers.length === 0 ? (
              <Empty
                title="No servers yet"
                sub="Register your first VPN node to start provisioning peers."
                action={<Link to="/servers"><Button variant="primary">Register server</Button></Link>}
              />
            ) : (
              <div className="table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Endpoint</th>
                      <th>Subnet</th>
                      <th>Obfs</th>
                      <th>Status</th>
                      <th>Last heartbeat</th>
                      <th>Public key</th>
                    </tr>
                  </thead>
                  <tbody>
                    {servers.map((s) => (
                      <tr key={s.id}>
                        <td>
                          <Link to={`/servers/${s.id}`}>{s.name}</Link>
                        </td>
                        <td><code>{s.endpoint}</code></td>
                        <td><code>{s.subnet}</code></td>
                        <td>{s.obfs_enabled ? <Badge tone="info">on</Badge> : <Badge>off</Badge>}</td>
                        <td><StatusDot online={s.online} /></td>
                        <td className="text-dim">{formatRelative(s.last_heartbeat)}</td>
                        <td><code>{maskKey(s.public_key)}</code></td>
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
