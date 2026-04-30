import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import { useI18n } from '../i18n'
import type { Server, Stats } from '../types'
import { Badge, Button, Skeleton } from '../components/ui'
import { formatBytes, formatRelative } from '../lib/format'

// ─── Animated counter ──────────────────────────────────────────────────────
function useCountUp(target: number, duration = 800): number {
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

// ─── MetricCard (Remnawave style) ──────────────────────────────────────────
function MetricCard({
  icon, iconColor, label, value, delta, deltaDir, sub, index = 0,
}: {
  icon: ReactNode
  iconColor: string
  label: string
  value: ReactNode
  delta?: string
  deltaDir?: 'up' | 'down' | 'mute'
  sub?: string
  index?: number
}) {
  return (
    <div className="metric-card" style={{ animationDelay: `${index * 55}ms` }}>
      <div
        className="metric-icon-wrap"
        style={{ background: iconColor + '22', color: iconColor }}
      >
        {icon}
      </div>
      <div className="metric-body">
        <div className="metric-label">{label}</div>
        <div className="metric-value">{value}</div>
        {delta && (
          <div className={`metric-delta ${deltaDir ?? 'mute'}`}>
            {deltaDir === 'up' ? '↑' : deltaDir === 'down' ? '↓' : ''}
            {' '}{delta}
          </div>
        )}
        {sub && !delta && <div className="metric-delta mute">{sub}</div>}
      </div>
    </div>
  )
}

// ─── Section header ─────────────────────────────────────────────────────────
function Section({ title, right, children, mb = true }: {
  title: string; right?: ReactNode; children: ReactNode; mb?: boolean
}) {
  return (
    <div style={{ marginBottom: mb ? 28 : 0 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
        <div className="section-title">{title}</div>
        {right && <div className="row">{right}</div>}
      </div>
      {children}
    </div>
  )
}

// ─── Dashboard ──────────────────────────────────────────────────────────────
export default function Dashboard() {
  const { t } = useI18n()
  const [stats, setStats] = useState<Stats | null>(null)
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(true)
  const [time, setTime] = useState(new Date())

  const load = async () => {
    try {
      const [s, srv] = await Promise.all([api.stats.get(), api.servers.list()])
      setStats(s)
      setServers(srv)
    } catch (e) {
      if (e instanceof ApiError) console.error(e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [])
  useEffect(() => {
    const t = setInterval(() => setTime(new Date()), 1000)
    return () => clearInterval(t)
  }, [])

  const totalBytes  = useCountUp((stats?.bytes_rx_total ?? 0) + (stats?.bytes_tx_total ?? 0))
  const rxBytes     = useCountUp(stats?.bytes_rx_total ?? 0)
  const txBytes     = useCountUp(stats?.bytes_tx_total ?? 0)
  const peersTotal  = useCountUp(stats?.peers ?? 0)
  const usersTotal  = useCountUp(stats?.users ?? 0)
  const serversOnline = useCountUp(servers.filter(s => s.online).length)
  const serversTotal  = useCountUp(servers.length)

  const timeStr = time.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit', second: '2-digit' })

  return (
    <div className="page">
      {/* ── Header ── */}
      <div className="page-header">
        <div>
          <div className="page-title">{t('dash_title')}</div>
          <div className="page-sub">{t('dash_sub')}</div>
        </div>
        <div className="row">
          <div className="live-badge">
            <span className="live-dot" />
            LIVE · {timeStr}
          </div>
          <Button onClick={load} loading={loading}>{t('dash_refresh')}</Button>
        </div>
      </div>

      {/* ── Трафик ── */}
      <Section title={t('dashboard_section_traffic')}>
        <div className="metric-grid-3">
          <MetricCard
            index={0}
            icon={<TrafficIcon />}
            iconColor="#06b6d4"
            label={t('dashboard_traffic_total')}
            value={loading ? <Skeleton height={26} width={80} /> : formatBytes(totalBytes)}
            sub={`↓ ${formatBytes(rxBytes)}  ↑ ${formatBytes(txBytes)}`}
          />
          <MetricCard
            index={1}
            icon={<DownloadIcon />}
            iconColor="#22c55e"
            label={t('dashboard_traffic_rx')}
            value={loading ? <Skeleton height={26} width={80} /> : formatBytes(rxBytes)}
            delta={t('dashboard_traffic_from_clients')}
            deltaDir="mute"
          />
          <MetricCard
            index={2}
            icon={<UploadIcon />}
            iconColor="#a78bfa"
            label={t('dashboard_traffic_tx')}
            value={loading ? <Skeleton height={26} width={80} /> : formatBytes(txBytes)}
            delta={t('dashboard_traffic_to_clients')}
            deltaDir="mute"
          />
        </div>
      </Section>

      {/* ── Инфраструктура ── */}
      <Section title={t('dashboard_section_infra')}>
        <div className="metric-grid-4">
          <MetricCard
            index={0}
            icon={<NodeIcon />}
            iconColor="#f59e0b"
            label={t('dashboard_nodes_online')}
            value={loading ? <Skeleton height={26} width={60} /> : `${serversOnline} / ${serversTotal}`}
            sub={serversTotal > 0 ? t('dash_all_healthy') : t('dash_no_nodes')}
          />
          <MetricCard
            index={1}
            icon={<ServerIcon />}
            iconColor="#64748b"
            label={t('dashboard_servers_online')}
            value={loading ? <Skeleton height={26} width={60} /> : serversOnline}
            delta={`${t('dashboard_of')} ${serversTotal} ${t('dashboard_registered')}`}
            deltaDir="mute"
          />
          <MetricCard
            index={2}
            icon={<PeersIcon />}
            iconColor="#8b5cf6"
            label={t('dashboard_clients')}
            value={loading ? <Skeleton height={26} width={60} /> : peersTotal}
            delta={t('dashboard_active_peers')}
            deltaDir="mute"
          />
          <MetricCard
            index={3}
            icon={<UsersIcon />}
            iconColor="#ec4899"
            label={t('dashboard_users')}
            value={loading ? <Skeleton height={26} width={60} /> : usersTotal}
            delta={t('dashboard_user_accounts')}
            deltaDir="mute"
          />
        </div>
      </Section>

      {/* ── Ноды ── */}
      <Section
        title={t('dashboard_section_nodes')}
        mb={false}
        right={
          <>
            <Link to="/servers/fleet">
              <Button size="sm">{t('dashboard_fleet_ops')}</Button>
            </Link>
            <Link to="/servers">
              <Button variant="primary" size="sm">{t('dashboard_manage')}</Button>
            </Link>
          </>
        }
      >
        {loading ? (
          <div className="stack-sm">
            {[0,1,2].map(i => <Skeleton key={i} height={64} />)}
          </div>
        ) : servers.length === 0 ? (
          <div className="dash-empty-nodes">
            <div className="dash-empty-icon"><NodeIcon /></div>
            <div className="dash-empty-title">{t('dash_no_servers')}</div>
            <div className="dash-empty-sub">{t('dash_no_servers_sub')}</div>
            <Link to="/servers">
              <Button variant="primary">{t('dashboard_add_server')}</Button>
            </Link>
          </div>
        ) : (
          <div className="stack-sm">
            {servers.map((srv, i) => (
              <NodeRow key={srv.id} srv={srv} index={i} t={t} />
            ))}
          </div>
        )}
      </Section>
    </div>
  )
}

// ─── NodeRow ────────────────────────────────────────────────────────────────
function NodeRow({ srv, index, t }: { srv: Server; index: number; t: (k: string, fb?: string) => string }) {
  const status = srv.online ? 'online' : srv.status === 'pending' ? 'pending' : 'offline'
  return (
    <Link to={`/servers/${srv.id}`} style={{ textDecoration: 'none' }}>
      <div
        className="node-card"
        style={{ animationDelay: `${index * 40}ms`, animation: 'card-in 300ms ease both' }}
      >
        <span className={`node-status-dot ${status}`} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontWeight: 600, fontSize: 14, color: 'var(--text)', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {srv.name}
          </div>
          <div style={{ fontSize: 12, color: 'var(--text-mute)', marginTop: 2 }}>
            {srv.hostname || srv.ip || '—'}
            {srv.endpoint ? ` · ${srv.endpoint}` : ''}
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexShrink: 0 }}>
          {srv.protocol && srv.protocol !== 'none' && (
            <span style={{ fontSize: 11, color: 'var(--text-mute)', fontFamily: 'var(--font-mono)' }}>
              {srv.protocol.toUpperCase()}
            </span>
          )}
          <Badge tone={status === 'online' ? 'success' : status === 'pending' ? 'warn' : 'danger'}>
            {t('status_' + status, status)}
          </Badge>
          {srv.last_heartbeat && (
            <span style={{ fontSize: 11, color: 'var(--text-mute)' }}>
              {formatRelative(srv.last_heartbeat)}
            </span>
          )}
        </div>
      </div>
    </Link>
  )
}

// ─── Icons ──────────────────────────────────────────────────────────────────
const ico = (d: string) => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d={d} />
  </svg>
)
const TrafficIcon  = () => ico("M22 12h-4l-3 9L9 3l-3 9H2")
const DownloadIcon = () => ico("M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3")
const UploadIcon   = () => ico("M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M17 8l-5-5-5 5M12 3v12")
const NodeIcon     = () => ico("M5 12.55a11 11 0 0 1 14.08 0M1.42 9a16 16 0 0 1 21.16 0M8.53 16.11a6 6 0 0 1 6.95 0M12 20h.01")
const ServerIcon   = () => ico("M2 8h20M2 16h20M6 8v8M18 8v8M2 4h20a0 0 0 0 1 0 4H2a0 0 0 0 1 0-4zM2 12h20a0 0 0 0 1 0 4H2a0 0 0 0 1 0-4z")
const PeersIcon    = () => ico("M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM23 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75")
const UsersIcon    = () => ico("M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8z")