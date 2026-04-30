import { useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import { useI18n } from '../i18n'
import type { Config, Server } from '../types'
import { Badge, Button, CopyField, Empty, Skeleton, toast } from '../components/ui'
import { formatRelative } from '../lib/format'

function IcoCheck() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/>
    </svg>
  )
}
function IcoDeploy() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/>
      <path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/>
    </svg>
  )
}
function IcoRestart() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="23 4 23 10 17 10"/>
      <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
    </svg>
  )
}
function IcoKey() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
    </svg>
  )
}
function IcoRefresh() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="1 4 1 10 7 10"/><polyline points="23 20 23 14 17 14"/>
      <path d="M20.49 9A9 9 0 0 0 5.64 5.64L1 10m22 4-4.64 4.36A9 9 0 0 1 3.51 15"/>
    </svg>
  )
}

function statusTone(s: string | undefined): 'success' | 'warn' | 'danger' | 'default' {
  if (!s) return 'default'
  if (s === 'online') return 'success'
  if (s === 'pending' || s === 'deploying' || s === 'drifted') return 'warn'
  if (s === 'error') return 'danger'
  return 'default'
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="row" style={{ alignItems: 'flex-start', gap: 12 }}>
      <div className="text-mute text-sm" style={{ width: 180, flexShrink: 0, paddingTop: 2 }}>{label}</div>
      <div style={{ flex: 1 }}>{children}</div>
    </div>
  )
}

function MetaCard({ label, value, sub, tone }: {
  label: string; value: ReactNode; sub?: string; tone?: string
}) {
  return (
    <div className={`stat-card${tone ? ` ${tone}` : ''}`}>
      <div className="stat-label">{label}</div>
      <div className="stat-value" style={{ fontSize: 20 }}>{value}</div>
      {sub && <div className="stat-meta">{sub}</div>}
    </div>
  )
}

export default function ServerDetail() {
  const { id }    = useParams<{ id: string }>()
  const { t }     = useI18n()
  const [server,  setServer]   = useState<Server | null>(null)
  const [configs, setConfigs]  = useState<Config[] | null>(null)
  const [error,   setError]    = useState<string | null>(null)
  const [checking, setChecking] = useState(false)

  async function load() {
    if (!id) return
    try {
      setError(null)
      const list = await api.servers.list()
      const s = list.find((x) => x.id === id)
      if (!s) { setError(t('server_detail_notfound')); return }
      setServer(s)
      setConfigs(await api.configs.listByServer(s.id))
    } catch (e) {
      setError(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'load failed')
    }
  }

  useEffect(() => { load() }, [id]) // eslint-disable-line react-hooks/exhaustive-deps

  const check = async () => {
    if (!server) return
    setChecking(true)
    try {
      await api.servers.check(server.id)
      toast.success(t('toast_check_ok'))
      await load()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'check failed')
    } finally {
      setChecking(false)
    }
  }

  const deploy = async () => {
    if (!server) return
    try {
      await api.servers.deploy(server.id)
      toast.success(t('toast_deployed'))
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : t('toast_deploy_fail'))
    }
  }

  const restart = async () => {
    if (!server) return
    if (!window.confirm(t('confirm_restart'))) return
    try {
      await api.servers.restart(server.id)
      toast.success(t('toast_restart_ok'))
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'restart failed')
    }
  }

  const rotateSecret = async () => {
    if (!server) return
    if (!window.confirm(t('confirm_rotate'))) return
    try {
      const r = await api.servers.rotateSecret(server.id)
      toast.success(t('toast_secret_ok'))
      navigator.clipboard?.writeText(r.secret).catch(() => undefined)
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'rotate failed')
    }
  }

  if (error) {
    return (
      <div className="page">
        <Empty
          title={t('server_detail_notfound')}
          sub={error}
          action={<Link to="/servers"><Button>{t('server_detail_back_btn')}</Button></Link>}
        />
      </div>
    )
  }

  const status = server?.status ?? (server?.online ? 'online' : 'offline')

  return (
    <div className="page">
      <div className="detail-header">
        <div className="detail-header-left">
          <div className="text-sm">
            <Link to="/servers" className="text-dim">← {t('server_detail_back')}</Link>
          </div>
          <div className="detail-title-row">
            <div className="page-title">
              {server?.name ?? <Skeleton width={180} height={28} />}
            </div>
            {server && (
              <Badge tone={statusTone(status)}>
                {t('status_' + status, status)}
              </Badge>
            )}
          </div>
          {server?.endpoint && (
            <div className="page-sub" style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>
              {server.endpoint}
            </div>
          )}
        </div>

        <div className="detail-actions">
          <Button icon={<IcoCheck />} onClick={check} loading={checking}>
            {t('server_detail_check')}
          </Button>
          <Button icon={<IcoDeploy />} variant="primary" onClick={deploy}>
            {t('server_detail_deploy')}
          </Button>
          <Button icon={<IcoRestart />} onClick={restart}>
            {t('server_detail_restart')}
          </Button>
          <Button icon={<IcoKey />} variant="danger" onClick={rotateSecret}>
            {t('server_detail_rotate')}
          </Button>
          <Button icon={<IcoRefresh />} variant="ghost" onClick={load}>
            {t('server_detail_refresh')}
          </Button>
        </div>
      </div>

      <div className="stats-grid" style={{ marginBottom: 20 }}>
        <MetaCard
          label={t('server_detail_node')}
          value={
            server
              ? <span style={{ color: server.online ? 'var(--success)' : 'var(--text-mute)' }}>
                  {server.online ? t('status_online') : t('status_offline')}
                </span>
              : <Skeleton height={24} width={100} />
          }
          sub={server ? `${t('server_detail_beat')}: ${formatRelative(server.last_heartbeat)}` : undefined}
        />
        <MetaCard
          tone="violet"
          label={t('server_detail_endpoint')}
          value={<span style={{ fontFamily: 'var(--font-mono)', fontSize: 14 }}>{server?.ip || '—'}</span>}
          sub={server?.hostname || t('server_detail_hostname')}
        />
        <MetaCard
          tone="success"
          label={t('server_detail_protocol')}
          value={server?.protocol || 'none'}
          sub={server?.mode || '—'}
        />
        <MetaCard
          tone="warn"
          label={t('server_detail_version')}
          value={server?.agent_version || '—'}
          sub={server?.status || t('status_pending')}
        />
      </div>

      <div className="card mb-4">
        <div className="card-header">
          <div>
            <div className="card-title">{t('server_detail_identity')}</div>
            <div className="card-sub">{t('server_detail_identity_sub')}</div>
          </div>
        </div>
        {server ? (
          <div className="stack">
            <Field label={t('server_detail_id_node')}><CopyField value={server.node_id} /></Field>
            <Field label={t('server_detail_id_srv')}><CopyField value={server.id} /></Field>
            <Field label={t('server_detail_status')}>
              <Badge tone={statusTone(server.status)}>{t('status_' + status, status)}</Badge>
            </Field>
          </div>
        ) : (
          <div className="stack">
            {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} height={20} />)}
          </div>
        )}
      </div>

      <div className="card">
        <div className="card-header">
          <div>
            <div className="card-title">{t('server_detail_configs')}</div>
            <div className="card-sub">{t('server_detail_configs_sub')}</div>
          </div>
          <Link to="/configs"><Button>{t('server_detail_cfg_manage')}</Button></Link>
        </div>
        {!configs ? (
          <div className="stack">
            {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} height={36} />)}
          </div>
        ) : configs.length === 0 ? (
          <Empty title={t('server_detail_no_cfg')} sub={t('server_detail_no_cfg_sub')} />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>{t('server_detail_cfg_name')}</th>
                  <th>{t('server_detail_cfg_tpl')}</th>
                  <th>{t('server_detail_cfg_mode')}</th>
                  <th>{t('server_detail_cfg_routing')}</th>
                  <th>{t('server_detail_cfg_status')}</th>
                </tr>
              </thead>
              <tbody>
                {configs.map((cfg) => (
                  <tr key={cfg.id}>
                    <td><strong>{cfg.name}</strong></td>
                    <td><code>{cfg.template}</code></td>
                    <td>{cfg.setup_mode}</td>
                    <td>{cfg.routing_mode}</td>
                    <td>
                      {cfg.is_active
                        ? <Badge tone="success">{t('server_detail_cfg_active')}</Badge>
                        : <Badge tone="warn">{t('server_detail_cfg_inactive')}</Badge>
                      }
                    </td>
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
