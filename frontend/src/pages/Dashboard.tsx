import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import { useAuth } from '../store/auth'
import { useI18n } from '../i18n'
import type { Server, Stats } from '../types'
import { Badge, Button, Empty, SkeletonRows, StatCard } from '../components/ui'
import { formatBytes, formatNumber, formatRelative } from '../lib/format'

function statusTone(s: string): 'success' | 'warn' | 'danger' | 'info' | 'violet' | 'default' {
  switch (s) {
    case 'online':    return 'success'
    case 'pending':   return 'warn'
    case 'deploying': return 'violet'
    case 'drifted':   return 'warn'
    case 'error':     return 'danger'
    case 'degraded':  return 'warn'
    case 'offline':   return 'default'
    default:          return 'default'
  }
}

function ServerCard({ server, t }: { server: Server; t: (k: string, fb?: string) => string }) {
  const status   = server.status ?? (server.online ? 'online' : 'offline')
  const tone     = statusTone(status)
  const isOnline = server.online

  const stripeColor =
    tone === 'success' ? 'var(--success)'
    : tone === 'danger'  ? 'var(--danger)'
    : tone === 'warn'    ? 'var(--warning)'
    : tone === 'violet'  ? 'var(--accent)'
    : 'var(--border-strong)'

  return (
    <Link to={`/servers/${server.id}`} style={{ textDecoration: 'none', display: 'block' }}>
      <div className="server-card">
        <div className="server-card-stripe" style={{ background: stripeColor }} />
        <div className="server-card-body">
          <div className="server-card-header">
            <div className="row gap-2">
              <span className="server-card-dot" style={{
                background: isOnline ? 'var(--success)' : 'var(--text-mute)',
                boxShadow: isOnline ? '0 0 8px var(--success)' : 'none',
              }} />
              <strong className="server-card-name">{server.name}</strong>
            </div>
            <Badge tone={tone}>{t('status_' + status, status)}</Badge>
          </div>
          <div className="server-card-meta">
            <span className="text-mono text-mute">{server.ip || server.endpoint}</span>
            {server.hostname && (
              <span className="text-xs text-mute" style={{ marginLeft: 6 }}>· {server.hostname}</span>
            )}
          </div>
          <div className="server-card-footer">
            <div className="row gap-2">
              {server.protocol && server.protocol !== 'none' && (
                <Badge tone="violet">{server.protocol}</Badge>
              )}
              {server.mode && <Badge>{server.mode}</Badge>}
            </div>
            <span className="text-xs text-mute">{formatRelative(server.last_heartbeat)}</span>
          </div>
        </div>
      </div>
    </Link>
  )
}

export default function Dashboard() {
  const user    = useAuth((s) => s.user)
  const { t }   = useI18n()
  const isStaff = user?.role === 'admin' || user?.role === 'operator'

  const [stats,   setStats]   = useState<Stats | null>(null)
  const [servers, setServers] = useState<Server[] | null>(null)
  const [error,   setError]   = useState<string | null>(null)
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
    const interval = setInterval(load, 15_000)
    return () => clearInterval(interval)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isStaff])

  const degraded = useMemo(
    () => (servers ?? []).filter((s) => s.online === false).length,
    [servers],
  )

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">{t('dashboard_title')}</div>
          <div className="page-sub">{t('dashboard_sub')}</div>
        </div>
        <Button variant="ghost" onClick={load} disabled={loading}>{t('action_refresh')}</Button>
      </div>

      {error && (
        <div className="state-panel state-error mb-4">
          <strong>{t('dashboard_unavailable')}:</strong> {error}
        </div>
      )}

      {!isStaff ? (
        <div className="card">
          <div className="card-header">
            <div>
              <div className="card-title">{t('dashboard_quick_title')}</div>
              <div className="card-sub">{t('dashboard_quick_sub')}</div>
            </div>
          </div>
          <div className="row" style={{ flexWrap: 'wrap' }}>
            <Link to="/clients"><Button variant="primary">{t('dashboard_quick_clients')}</Button></Link>
            <Link to="/configs"><Button>{t('dashboard_quick_configs')}</Button></Link>
            <Link to="/profile"><Button>{t('dashboard_quick_profile')}</Button></Link>
          </div>
        </div>
      ) : (
        <>
          <div className="stats-grid">
            <StatCard label={t('dashboard_users')} value={loading ? '...' : formatNumber(stats?.users ?? 0)} />
            <StatCard label={t('dashboard_clients')} value={loading ? '...' : formatNumber(stats?.peers ?? 0)} tone="violet" />
            <StatCard
              label={t('dashboard_servers')}
              value={loading ? '...' : `${stats?.servers_online ?? 0}/${stats?.servers ?? 0}`}
              meta={loading ? undefined : degraded > 0 ? t('dashboard_degraded').replace('{n}', String(degraded)) : t('dashboard_all_healthy')}
              tone={degraded > 0 ? 'warn' : 'success'}
            />
            <StatCard
              label={t('dashboard_traffic')}
              value={loading ? '...' : formatBytes((stats?.bytes_rx_total ?? 0) + (stats?.bytes_tx_total ?? 0))}
              meta={loading ? undefined : `↓ ${formatBytes(stats?.bytes_rx_total ?? 0)}  ↑ ${formatBytes(stats?.bytes_tx_total ?? 0)}`}
              tone="success"
            />
          </div>

          <div className="card">
            <div className="card-header">
              <div>
                <div className="card-title">{t('servers_status_card')}</div>
                <div className="card-sub">{t('servers_status_sub')}</div>
              </div>
              <div className="row">
                <Link to="/settings"><Button>{t('servers_fleet_ops')}</Button></Link>
                <Link to="/servers"><Button variant="primary">{t('servers_manage')}</Button></Link>
              </div>
            </div>
            {loading ? (
              <SkeletonRows rows={4} />
            ) : !servers || servers.length === 0 ? (
              <Empty
                title={t('dashboard_no_servers')}
                sub={t('dashboard_no_servers_sub')}
                action={<Link to="/servers"><Button variant="primary">{t('dashboard_register')}</Button></Link>}
              />
            ) : (
              <div className="servers-grid">
                {servers.map((s) => (
                  <ServerCard key={s.id} server={s} t={t} />
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}
