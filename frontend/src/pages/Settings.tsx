import { useEffect, useRef, useState } from 'react'
import { api, ApiError } from '../api/client'
import type { FleetHealthResult, FleetRedeployResult, SystemVersionInfo } from '../types'
import { Badge, Button, Empty, Skeleton, toast } from '../components/ui'
import { useI18n } from '../i18n'
import { formatRelative } from '../lib/format'

function healthTone(s: string): 'success' | 'danger' | 'warn' {
  if (s === 'online') return 'success'
  if (s === 'offline') return 'danger'
  return 'warn'
}

function uptimeStr(s: number): string {
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60)
  if (d > 0) return `${d}д ${h}ч`
  if (h > 0) return `${h}ч ${m}м`
  return `${m}м`
}

// Strip ANSI color codes from terminal output
function stripAnsi(s: string): string {
  // eslint-disable-next-line no-control-regex
  return s.replace(/\x1B\[[0-9;]*[mGKHF]/g, '')
}

export default function Settings() {
  const { t, locale, setLocale } = useI18n()
  const [health,         setHealth]         = useState<FleetHealthResult[] | null>(null)
  const [redeployResult, setRedeployResult] = useState<FleetRedeployResult[] | null>(null)
  const [loading,        setLoading]        = useState(false)
  const [redeploying,    setRedeploying]    = useState(false)
  const [version,        setVersion]        = useState<SystemVersionInfo | null>(null)
  const [versionLoading, setVersionLoading] = useState(false)
  const [updating,       setUpdating]       = useState(false)
  const [updateLog,      setUpdateLog]      = useState<string[]>([])
  const [updateDone,     setUpdateDone]     = useState(false)
  const logRef = useRef<HTMLDivElement>(null)
  const esRef  = useRef<EventSource | null>(null)

  const loadHealth = async () => {
    setLoading(true)
    try { setHealth(await api.fleet.health()) }
    catch (e) { toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'failed') }
    finally { setLoading(false) }
  }

  const loadVersion = async () => {
    setVersionLoading(true)
    try { setVersion(await api.system.version()) }
    catch { /* silent */ }
    finally { setVersionLoading(false) }
  }

  useEffect(() => { loadHealth(); loadVersion() }, [])

  // Auto-scroll log to bottom
  useEffect(() => {
    if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight
  }, [updateLog])

  // Cleanup SSE on unmount
  useEffect(() => () => esRef.current?.close(), [])

  const startUpdateStream = () => {
    if (updating) { esRef.current?.close(); return }
    setUpdating(true)
    setUpdateDone(false)
    setUpdateLog([])

    const es = new EventSource('/api/v1/admin/system/update/stream')
    esRef.current = es

    es.onmessage = (e) => {
      const line = stripAnsi(e.data)
      if (line === '__DONE__') {
        setUpdating(false)
        setUpdateDone(true)
        es.close()
        toast.success(t('settings_update_started'))
        setTimeout(() => loadVersion(), 3000)
        return
      }
      if (line.startsWith('__ERROR__:')) {
        setUpdating(false)
        es.close()
        toast.error(line.replace('__ERROR__:', ''))
        return
      }
      setUpdateLog((prev) => [...prev, line])
    }
    es.onerror = () => {
      setUpdating(false)
      es.close()
    }
  }

  const redeployAll = async () => {
    setRedeploying(true); setRedeployResult(null)
    try {
      const result = await api.fleet.redeployAll()
      setRedeployResult(result)
      const failed = result.filter((r) => r.status === 'error').length
      if (failed === 0) toast.success(t('toast_deployed'))
      else toast.warn(`${failed} нод с ошибкой`)
      await loadHealth()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : t('toast_deploy_fail'))
    } finally { setRedeploying(false) }
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">{t('settings_title')}</div>
          <div className="page-sub">{t('settings_sub')}</div>
        </div>
        <div className="row">
          <Button onClick={loadHealth} loading={loading}>{t('settings_refresh_health')}</Button>
          <Button variant="primary" onClick={redeployAll} loading={redeploying}>
            {t('settings_redeploy_all')}
          </Button>
        </div>
      </div>

      {/* Language */}
      <div className="card mb-4">
        <div className="card-header">
          <div>
            <div className="card-title">{t('settings_language')}</div>
            <div className="card-sub">{t('settings_language_hint')}</div>
          </div>
        </div>
        <div className="row" style={{ gap: 8 }}>
          {(['ru', 'en'] as const).map((loc) => (
            <button key={loc} onClick={() => setLocale(loc)}
              className={`btn${locale === loc ? ' btn-primary' : ''}`}>
              {loc === 'ru' ? 'Русский' : 'English'}
            </button>
          ))}
        </div>
      </div>

      {/* System update */}
      <div className="card mb-4">
        <div className="card-header">
          <div>
            <div className="card-title">{t('settings_update_title')}</div>
            <div className="card-sub">{t('settings_update_sub')}</div>
          </div>
          <Button variant="primary" size="sm" onClick={startUpdateStream} loading={updating}>
            {updating ? t('settings_update_running') : t('settings_update_btn')}
          </Button>
        </div>

        {/* Version info */}
        {versionLoading ? (
          <div className="stack-sm">
            <Skeleton height={14} width="40%" />
            <Skeleton height={14} width="30%" />
          </div>
        ) : version ? (
          <div className="version-grid">
            <div className="version-item">
              <span className="text-xs text-mute">{t('settings_ver_commit')}</span>
              <code className="text-xs" style={{ fontFamily: 'var(--font-mono)', color: 'var(--accent)' }}>
                {version.commit.slice(0, 12)}
              </code>
            </div>
            <div className="version-item">
              <span className="text-xs text-mute">{t('settings_ver_branch')}</span>
              <code className="text-xs" style={{ fontFamily: 'var(--font-mono)' }}>{version.branch}</code>
            </div>
            <div className="version-item">
              <span className="text-xs text-mute">{t('settings_ver_built')}</span>
              <span className="text-xs">{version.built_at || '—'}</span>
            </div>
            <div className="version-item">
              <span className="text-xs text-mute">{t('settings_ver_uptime')}</span>
              <span className="text-xs">{uptimeStr(version.uptime_seconds)}</span>
            </div>
          </div>
        ) : (
          <div className="text-sm text-mute">{t('settings_ver_unavailable')}</div>
        )}

        {/* SSE log terminal */}
        {(updating || updateLog.length > 0) && (
          <div className="update-terminal" ref={logRef}>
            <div className="update-terminal-header">
              <span className="update-terminal-dot" style={{ background: updating ? 'var(--warning)' : updateDone ? 'var(--success)' : 'var(--danger)' }} />
              <span className="text-xs" style={{ fontFamily: 'var(--font-mono)', color: 'rgba(255,255,255,0.4)' }}>
                {updating ? t('settings_update_running') : updateDone ? t('settings_update_started') : t('settings_update_fail')}
              </span>
            </div>
            {updateLog.map((line, i) => (
              <div key={i} className="update-terminal-line">{line || ' '}</div>
            ))}
            {updating && <div className="update-terminal-cursor" />}
          </div>
        )}

        <div className="text-xs text-mute" style={{ marginTop: 12, paddingTop: 12, borderTop: '1px solid var(--border)' }}>
          {t('settings_update_note')}
        </div>
      </div>

      {/* Fleet health */}
      <div className="card mb-4">
        <div className="card-header">
          <div>
            <div className="card-title">{t('settings_fleet_title')}</div>
            <div className="card-sub">{t('settings_fleet_sub')}</div>
          </div>
        </div>
        {health === null ? (
          <div className="stack">
            {[0,1,2].map((i) => <Skeleton key={i} height={38} />)}
          </div>
        ) : health.length === 0 ? (
          <Empty title={t('settings_no_nodes')} sub={t('settings_no_nodes_sub')} />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead><tr>
                <th>{t('settings_col_server')}</th>
                <th>{t('settings_col_protocol')}</th>
                <th>{t('settings_col_status')}</th>
                <th>{t('settings_col_beat')}</th>
                <th>{t('settings_col_error')}</th>
              </tr></thead>
              <tbody>
                {health.map((h) => (
                  <tr key={h.server_id}>
                    <td><strong>{h.name}</strong></td>
                    <td>{h.protocol}</td>
                    <td><Badge tone={healthTone(h.status)}>{t('status_' + h.status, h.status)}</Badge></td>
                    <td className="text-dim">{formatRelative(h.last_heartbeat)}</td>
                    <td className="text-danger text-sm">{h.error || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Redeploy result */}
      <div className="card">
        <div className="card-header">
          <div>
            <div className="card-title">{t('settings_last_op_title')}</div>
            <div className="card-sub">{t('settings_last_op_sub')}</div>
          </div>
        </div>
        {!redeployResult ? (
          <Empty title={t('settings_no_op')} sub={t('settings_no_op_sub')} />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead><tr>
                <th>{t('settings_col_server')}</th>
                <th>{t('settings_col_status')}</th>
                <th>{t('settings_col_retries')}</th>
                <th>{t('settings_col_error')}</th>
              </tr></thead>
              <tbody>
                {redeployResult.map((r) => (
                  <tr key={r.server_id}>
                    <td>{r.name}</td>
                    <td><Badge tone={r.status === 'ok' ? 'success' : 'danger'}>{r.status}</Badge></td>
                    <td>{r.retries}</td>
                    <td className="text-danger text-sm">{r.error || '—'}</td>
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
