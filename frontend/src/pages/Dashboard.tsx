import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import { useAuth } from '../store/auth'
import { useI18n } from '../i18n'
import type { Server, Stats } from '../types'
import { Badge, Button } from '../components/ui'
import { formatBytes, formatNumber, formatRelative } from '../lib/format'

// ─── Animated counter ────────────────────────────────────────────────────────
function useCountUp(target: number, duration = 900): number {
  const [val, setVal] = useState(0)
  const ref = useRef<number>(0)
  useEffect(() => {
    if (target === ref.current) return
    ref.current = target
    if (target === 0) { setVal(0); return }
    const start = performance.now()
    const from = val
    const step = (now: number) => {
      const t = Math.min((now - start) / duration, 1)
      const ease = 1 - Math.pow(1 - t, 4)
      setVal(Math.round(from + (target - from) * ease))
      if (t < 1) requestAnimationFrame(step)
    }
    requestAnimationFrame(step)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target])
  return val
}

// ─── MetricCard ───────────────────────────────────────────────────────────────
function MetricCard({
  icon, iconBg, label, value, sub, subTone, loading, index = 0,
}: {
  icon: ReactNode; iconBg: string; label: string
  value: ReactNode; sub?: ReactNode
  subTone?: 'success' | 'warn' | 'danger' | 'mute'
  loading?: boolean; index?: number
}) {
  const subColor =
    subTone === 'success' ? 'var(--success)'
    : subTone === 'warn'    ? 'var(--warning)'
    : subTone === 'danger'  ? 'var(--danger)'
    : 'rgba(255,255,255,0.28)'

  return (
    <div className="metric-card" style={{ animationDelay: `${index * 60}ms` }}>
      <div className="metric-icon-wrap" style={{ background: iconBg }}>
        <div className="metric-icon-glow" style={{ background: iconBg }} />
        {icon}
      </div>
      <div className="metric-body">
        <div className="metric-label">{label}</div>
        {loading
          ? <div className="metric-skeleton" />
          : <div className="metric-value">{value}</div>
        }
        {sub && !loading && (
          <div className="metric-sub" style={{ color: subColor }}>{sub}</div>
        )}
      </div>
    </div>
  )
}

// ─── Section title ────────────────────────────────────────────────────────────
function SectionTitle({ children, right }: { children: ReactNode; right?: ReactNode }) {
  return (
    <div className="dash-section-row">
      <div className="dash-section-title">{children}</div>
      {right && <div>{right}</div>}
    </div>
  )
}

// ─── Icons ───────────────────────────────────────────────────────────────────
const IC = {
  download: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>,
  upload:   <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>,
  traffic:  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>,
  server:   <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/></svg>,
  users:    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>,
  peers:    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>,
  wifi:     <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><path d="M5 12.55a11 11 0 0 1 14.08 0"/><path d="M1.42 9a16 16 0 0 1 21.16 0"/><path d="M8.53 16.11a6 6 0 0 1 6.95 0"/><line x1="12" y1="20" x2="12.01" y2="20"/></svg>,
}

function statusTone(s: string): 'success' | 'warn' | 'danger' | 'default' {
  if (s === 'online') return 'success'
  if (s === 'pending' || s === 'drifted' || s === 'deploying') return 'warn'
  if (s === 'error') return 'danger'
  return 'default'
}

// ─── Server card ─────────────────────────────────────────────────────────────
function ServerCard({ server, t }: { server: Server; t: (k: string, fb?: string) => string }) {
  const status = server.status ?? (server.online ? 'online' : 'offline')
  const tone = statusTone(status)
  const isOnline = server.online
  const stripeColor =
    tone === 'success' ? 'var(--success)'
    : tone === 'danger' ? 'var(--danger)'
    : tone === 'warn'   ? 'var(--warning)'
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
            {server.hostname && <span className="text-xs text-mute" style={{ marginLeft: 6 }}>· {server.hostname}</span>}
          </div>
          <div className="server-card-footer">
            <div className="row gap-2">
              {server.protocol && server.protocol !== 'none' && <Badge tone="violet">{server.protocol}</Badge>}
              {server.mode && <Badge>{server.mode}</Badge>}
            </div>
            <span className="text-xs text-mute">{formatRelative(server.last_heartbeat)}</span>
          </div>
        </div>
      </div>
    </Link>
  )
}

// ─── Main ─────────────────────────────────────────────────────────────────────
export default function Dashboard() {
  const user    = useAuth((s) => s.user)
  const { t }   = useI18n()
  const isStaff = user?.role === 'admin' || user?.role === 'operator'

  const [stats,   setStats]   = useState<Stats | null>(null)
  const [servers, setServers] = useState<Server[] | null>(null)
  const [error,   setError]   = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [lastTick, setLastTick] = useState(new Date())

  const load = async () => {
    setError(null)
    setLoading(true)
    try {
      if (isStaff) {
        await Promise.all([
          api.stats.get().then(setStats),
          api.servers.list().then(setServers),
        ])
      }
      setLastTick(new Date())
    } catch (e) {
      setError(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'failed')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
    if (!isStaff) return
    const iv = setInterval(load, 15_000)
    return () => clearInterval(iv)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isStaff])

  const degraded  = useMemo(() => (servers ?? []).filter((s) => !s.online).length, [servers])
  const totalBytes = (stats?.bytes_rx_total ?? 0) + (stats?.bytes_tx_total ?? 0)
  const onlineN   = stats?.servers_online ?? 0
  const totalN    = stats?.servers ?? 0
  const healthy   = totalN > 0 && degraded === 0

  // Animated counters
  const cUsers   = useCountUp(stats?.users         ?? 0)
  const cPeers   = useCountUp(stats?.peers         ?? 0)
  const cOnline  = useCountUp(onlineN)
  const cTotal   = useCountUp(totalN)

  return (
    <div className="page">
      {/* Header */}
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
            <MetricCard index={0} loading={loading}
              icon={IC.traffic}
              iconBg="linear-gradient(135deg,#6d28d9,#a855f7)"
              label={t('dashboard_traffic_total')}
              value={formatBytes(totalBytes)}
              sub={`↓ ${formatBytes(stats?.bytes_rx_total ?? 0)}  ·  ↑ ${formatBytes(stats?.bytes_tx_total ?? 0)}`}
              subTone="mute"
            />
            <MetricCard index={1} loading={loading}
              icon={IC.download}
              iconBg="linear-gradient(135deg,#0d9488,#2dd4bf)"
              label={t('dashboard_traffic_rx')}
              value={formatBytes(stats?.bytes_rx_total ?? 0)}
              sub={t('dashboard_traffic_rx_sub')}
              subTone="success"
            />
            <MetricCard index={2} loading={loading}
              icon={IC.upload}
              iconBg="linear-gradient(135deg,#0369a1,#38bdf8)"
              label={t('dashboard_traffic_tx')}
              value={formatBytes(stats?.bytes_tx_total ?? 0)}
              sub={t('dashboard_traffic_tx_sub')}
              subTone="mute"
            />
          </div>

          {/* ── Инфраструктура ── */}
          <SectionTitle>{t('dashboard_section_infra')}</SectionTitle>
          <div className="metrics-grid-4 mb-6">
            <MetricCard index={0} loading={loading}
              icon={IC.wifi}
              iconBg={healthy
                ? 'linear-gradient(135deg,#15803d,#22c55e)'
                : 'linear-gradient(135deg,#92400e,#f59e0b)'}
              label={t('dashboard_nodes_online')}
              value={`${cOnline} / ${cTotal}`}
              sub={loading ? undefined : degraded > 0
                ? t('dashboard_degraded').replace('{n}', String(degraded))
                : t('dashboard_all_healthy')}
              subTone={degraded > 0 ? 'warn' : 'success'}
            />
            <MetricCard index={1} loading={loading}
              icon={IC.server}
              iconBg="linear-gradient(135deg,#374151,#6b7280)"
              label={t('dashboard_servers')}
              value={formatNumber(cTotal)}
              sub={t('dashboard_servers_sub')}
              subTone="mute"
            />
            <MetricCard index={2} loading={loading}
              icon={IC.peers}
              iconBg="linear-gradient(135deg,#5b21b6,#8b5cf6)"
              label={t('dashboard_clients')}
              value={formatNumber(cPeers)}
              sub={t('dashboard_clients_sub')}
              subTone="mute"
            />
            <MetricCard index={3} loading={loading}
              icon={IC.users}
              iconBg="linear-gradient(135deg,#92400e,#d97706)"
              label={t('dashboard_users')}
              value={formatNumber(cUsers)}
              sub={t('dashboard_users_sub')}
              subTone="mute"
            />
          </div>

          {/* ── Ноды ── */}
          <SectionTitle right={
            <div className="row gap-2">
              <Link to="/settings"><Button size="sm">{t('servers_fleet_ops')}</Button></Link>
              <Link to="/servers"><Button size="sm" variant="primary">{t('servers_manage')}</Button></Link>
            </div>
          }>
            {t('dashboard_section_nodes')}
          </SectionTitle>

          {loading ? (
            <div className="servers-grid">
              {[0,1,2].map((i) => (
                <div key={i} className="server-card">
                  <div className="server-card-stripe" style={{ background: 'var(--border-strong)' }} />
                  <div className="server-card-body" style={{ gap: 10 }}>
                    <div className="metric-skeleton" style={{ height: 16 }} />
                    <div className="metric-skeleton" style={{ height: 13, width: '55%' }} />
                    <div className="metric-skeleton" style={{ height: 13, width: '35%' }} />
                  </div>
                </div>
              ))}
            </div>
          ) : !servers?.length ? (
            <div className="dash-empty-nodes">
              <div className="dash-empty-icon">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <rect x="2" y="2" width="20" height="8" rx="2"/><rect x="2" y="14" width="20" height="8" rx="2"/>
                  <line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/>
                </svg>
              </div>
              <div className="dash-empty-title">{t('dashboard_no_servers')}</div>
              <div className="dash-empty-sub">{t('dashboard_no_servers_sub')}</div>
              <Link to="/servers">
                <Button variant="primary">{t('dashboard_register')}</Button>
              </Link>
            </div>
          ) : (
            <div className="servers-grid">
              {servers.map((s) => <ServerCard key={s.id} server={s} t={t} />)}
            </div>
          )}
        </>
      )}
    </div>
  )
}
