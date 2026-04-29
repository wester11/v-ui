import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import { useAuth } from '../store/auth'
import { useI18n } from '../i18n'
import type { Server, Stats } from '../types'
import { Badge, Button, Empty } from '../components/ui'
import { formatBytes, formatNumber, formatRelative } from '../lib/format'

// ─── Metric card (Remnawave-style) ──────────────────────────────────────────

function MetricCard({
  icon, iconBg, label, value, sub, subTone, loading,
}: {
  icon: ReactNode
  iconBg: string
  label: string
  value: ReactNode
  sub?: ReactNode
  subTone?: 'success' | 'warn' | 'danger' | 'mute'
  loading?: boolean
}) {
  const subColor =
    subTone === 'success' ? 'var(--success)'
    : subTone === 'warn'  ? 'var(--warning)'
    : subTone === 'danger' ? 'var(--danger)'
    : 'var(--text-mute)'

  return (
    <div className="metric-card">
      <div className="metric-icon-wrap" style={{ background: iconBg }}>
        {icon}
      </div>
      <div className="metric-body">
        <div className="metric-label">{label}</div>
        {loading ? (
          <div className="metric-skeleton" />
        ) : (
          <div className="metric-value">{value}</div>
        )}
        {sub && !loading && (
          <div className="metric-sub" style={{ color: subColor }}>{sub}</div>
        )}
      </div>
    </div>
  )
}

// ─── Section title ───────────────────────────────────────────────────────────

function SectionTitle({ children }: { children: ReactNode }) {
  return <div className="dash-section-title">{children}</div>
}

// ─── SVG icons ───────────────────────────────────────────────────────────────

const IC = {
  download: (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
      <polyline points="7 10 12 15 17 10"/>
      <line x1="12" y1="15" x2="12" y2="3"/>
    </svg>
  ),
  upload: (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
      <polyline points="17 8 12 3 7 8"/>
      <line x1="12" y1="3" x2="12" y2="15"/>
    </svg>
  ),
  traffic: (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
    </svg>
  ),
  server: (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="2" y="2" width="20" height="8" rx="2"/>
      <rect x="2" y="14" width="20" height="8" rx="2"/>
      <line x1="6" y1="6" x2="6.01" y2="6"/>
      <line x1="6" y1="18" x2="6.01" y2="18"/>
    </svg>
  ),
  users: (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
      <circle cx="9" cy="7" r="4"/>
      <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
      <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
    </svg>
  ),
  peers: (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="18" cy="5" r="3"/>
      <circle cx="6" cy="12" r="3"/>
      <circle cx="18" cy="19" r="3"/>
      <line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/>
      <line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/>
    </svg>
  ),
  nodeOnline: (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M5 12.55a11 11 0 0 1 14.08 0"/>
      <path d="M1.42 9a16 16 0 0 1 21.16 0"/>
      <path d="M8.53 16.11a6 6 0 0 1 6.95 0"/>
      <line x1="12" y1="20" x2="12.01" y2="20"/>
    </svg>
  ),
}

// ─── Status tone ─────────────────────────────────────────────────────────────

function statusTone(s: string): 'success' | 'warn' | 'danger' | 'default' {
  switch (s) {
    case 'online':               return 'success'
    case 'pending': case 'drifted': case 'deploying': return 'warn'
    case 'error':                return 'danger'
    default:                     return 'default'
  }
}

// ─── Server card ─────────────────────────────────────────────────────────────

function ServerCard({ server, t }: { server: Server; t: (k: string, fb?: string) => string }) {
  const status   = server.status ?? (server.online ? 'online' : 'offline')
  const tone     = statusTone(status)
  const isOnline = server.online

  const stripeColor =
    tone === 'success' ? 'var(--success)'
    : tone === 'danger'  ? 'var(--danger)'
    : tone === 'warn'    ? 'var(--warning)'
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

// ─── Main dashboard ───────────────────────────────────────────────────────────

export default function Dashboard() {
  const user    = useAuth((s) => s.user)
  const { t }   = useI18n()
  const isStaff = user?.role === 'admin' || user?.role === 'operator'

  const [stats,   setStats]   = useState<Stats | null>(null)
  const [servers, setServers] = useState<Server[] | null>(null)
  const [error,   setError]   = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [lastTick, setLastTick] = useState<Date>(new Date())

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
      setLastTick(new Date())
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

  const totalTraffic = (stats?.bytes_rx_total ?? 0) + (stats?.bytes_tx_total ?? 0)
  const onlineCount  = stats?.servers_online ?? 0
  const totalCount   = stats?.servers ?? 0
  const nodesHealthy = totalCount > 0 && degraded === 0

  return (
    <div className="page">
      {/* ── Header ── */}
      <div className="page-header">
        <div>
          <div className="page-title">{t('dashboard_title')}</div>
          <div className="page-sub">{t('dashboard_sub')}</div>
        </div>
        <div className="row gap-3" style={{ alignItems: 'center' }}>
          {!loading && (
            <span className="live-badge">
              <span className="live-dot" />
              LIVE · {lastTick.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
            </span>
          )}
          <Button variant="ghost" onClick={load} disabled={loading}>{t('action_refresh')}</Button>
        </div>
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
          {/* ── Трафик ── */}
          <SectionTitle>{t('dashboard_section_traffic')}</SectionTitle>
          <div className="metrics-grid-3 mb-6">
            <MetricCard
              loading={loading}
              icon={IC.traffic}
              iconBg="linear-gradient(135deg,#7c3aed,#a855f7)"
              label={t('dashboard_traffic_total')}
              value={formatBytes(totalTraffic)}
              sub={`↓ ${formatBytes(stats?.bytes_rx_total ?? 0)}  ·  ↑ ${formatBytes(stats?.bytes_tx_total ?? 0)}`}
              subTone="mute"
            />
            <MetricCard
              loading={loading}
              icon={IC.download}
              iconBg="linear-gradient(135deg,#0d9488,#14b8a6)"
              label={t('dashboard_traffic_rx')}
              value={formatBytes(stats?.bytes_rx_total ?? 0)}
              sub={t('dashboard_traffic_rx_sub')}
              subTone="success"
            />
            <MetricCard
              loading={loading}
              icon={IC.upload}
              iconBg="linear-gradient(135deg,#0284c7,#38bdf8)"
              label={t('dashboard_traffic_tx')}
              value={formatBytes(stats?.bytes_tx_total ?? 0)}
              sub={t('dashboard_traffic_tx_sub')}
              subTone="mute"
            />
          </div>

          {/* ── Инфраструктура ── */}
          <SectionTitle>{t('dashboard_section_infra')}</SectionTitle>
          <div className="metrics-grid-4 mb-6">
            <MetricCard
              loading={loading}
              icon={IC.nodeOnline}
              iconBg={nodesHealthy ? 'linear-gradient(135deg,#16a34a,#22c55e)' : 'linear-gradient(135deg,#b45309,#f59e0b)'}
              label={t('dashboard_nodes_online')}
              value={`${onlineCount} / ${totalCount}`}
              sub={loading ? undefined : degraded > 0
                ? t('dashboard_degraded').replace('{n}', String(degraded))
                : t('dashboard_all_healthy')}
              subTone={degraded > 0 ? 'warn' : 'success'}
            />
            <MetricCard
              loading={loading}
              icon={IC.server}
              iconBg="linear-gradient(135deg,#475569,#64748b)"
              label={t('dashboard_servers')}
              value={formatNumber(stats?.servers ?? 0)}
              sub={t('dashboard_servers_sub')}
              subTone="mute"
            />
            <MetricCard
              loading={loading}
              icon={IC.peers}
              iconBg="linear-gradient(135deg,#7c3aed,#8b5cf6)"
              label={t('dashboard_clients')}
              value={formatNumber(stats?.peers ?? 0)}
              sub={t('dashboard_clients_sub')}
              subTone="mute"
            />
            <MetricCard
              loading={loading}
              icon={IC.users}
              iconBg="linear-gradient(135deg,#b45309,#d97706)"
              label={t('dashboard_users')}
              value={formatNumber(stats?.users ?? 0)}
              sub={t('dashboard_users_sub')}
              subTone="mute"
            />
          </div>

          {/* ── Серверы ── */}
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
            <SectionTitle>{t('dashboard_section_nodes')}</SectionTitle>
            <div className="row gap-2">
              <Link to="/settings"><Button size="sm">{t('servers_fleet_ops')}</Button></Link>
              <Link to="/servers"><Button size="sm" variant="primary">{t('servers_manage')}</Button></Link>
            </div>
          </div>

          {loading ? (
            <div className="servers-grid">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="server-card">
                  <div className="server-card-stripe" style={{ background: 'var(--border-strong)' }} />
                  <div className="server-card-body" style={{ gap: 10 }}>
                    <div className="metric-skeleton" style={{ height: 18 }} />
                    <div className="metric-skeleton" style={{ height: 14, width: '60%' }} />
                    <div className="metric-skeleton" style={{ height: 14, width: '40%' }} />
                  </div>
                </div>
              ))}
            </div>
          ) : !servers || servers.length === 0 ? (
            <div className="card">
              <Empty
                title={t('dashboard_no_servers')}
                sub={t('dashboard_no_servers_sub')}
                action={<Link to="/servers"><Button variant="primary">{t('dashboard_register')}</Button></Link>}
              />
            </div>
          ) : (
            <div className="servers-grid">
              {servers.map((s) => (
                <ServerCard key={s.id} server={s} t={t} />
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}
