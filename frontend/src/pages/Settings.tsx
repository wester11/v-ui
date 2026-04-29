import { useEffect, useState } from 'react'
import { api, ApiError } from '../api/client'
import type { FleetHealthResult, FleetRedeployResult, SystemVersionInfo } from '../types'
import { Badge, Button, Empty, Skeleton, toast } from '../components/ui'
import { useI18n } from '../i18n'
import { formatRelative } from '../lib/format'

function healthTone(status: string): 'success' | 'danger' | 'warn' {
  if (status === 'online') return 'success'
  if (status === 'offline') return 'danger'
  return 'warn'
}

function uptimeStr(seconds: number): string {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}д ${h}ч`
  if (h > 0) return `${h}ч ${m}м`
  return `${m}м`
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
  const [updateMsg,      setUpdateMsg]      = useState<string | null>(null)

  const loadHealth = async () => {
    setLoading(true)
    try {
      setHealth(await api.fleet.health())
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'health check failed')
    } finally {
      setLoading(false)
    }
  }

  const loadVersion = async () => {
    setVersionLoading(true)
    try {
      setVersion(await api.system.version())
    } catch {
      // version endpoint may not be deployed yet — fail silently
    } finally {
      setVersionLoading(false)
    }
  }

  useEffect(() => {
    loadHealth()
    loadVersion()
  }, [])

  const redeployAll = async () => {
    setRedeploying(true)
    setRedeployResult(null)
    try {
      const result = await api.fleet.redeployAll()
      setRedeployResult(result)
      const failed = result.filter((r) => r.status === 'error').length
      if (failed === 0) toast.success(t('toast_deployed'))
      else toast.warn(`${failed} нод с ошибкой`)
      await loadHealth()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : t('toast_deploy_fail'))
    } finally {
      setRedeploying(false)
    }
  }

  const runUpdate = async () => {
    setUpdating(true)
    setUpdateMsg(null)
    try {
      const res = await api.system.update()
      setUpdateMsg(res.message)
      if (res.status === 'error') {
        toast.error(res.message || t('settings_update_fail'))
      } else {
        toast.success(t('settings_update_started'))
      }
      // Reload version after a short delay
      setTimeout(() => loadVersion(), 3000)
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : t('settings_update_fail'))
    } finally {
      setUpdating(false)
    }
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
            <button
              key={loc}
              onClick={() => setLocale(loc)}
              className={`btn${locale === loc ? ' btn-primary' : ''}`}
            >
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
          <Button
            variant="primary"
            size="sm"
            onClick={runUpdate}
            loading={updating}
          >
            {t('settings_update_btn')}
          </Button>
        </div>

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

        {updateMsg && (
          <div className="update-msg-box">
            <span className="text-xs" style={{ fontFamily: 'var(--font-mono)' }}>{updateMsg}</span>
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
            {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} height={38} />)}
          </div>
        ) : health.length === 0 ? (
          <Empty title={t('settings_no_nodes')} sub={t('settings_no_nodes_sub')} />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>{t('settings_col_server')}</th>
                  <th>{t('settings_col_protocol')}</th>
                  <th>{t('settings_col_status')}</th>
                  <th>{t('settings_col_beat')}</th>
                  <th>{t('settings_col_error')}</th>
                </tr>
              </thead>
              <tbody>
                {health.map((h) => (
                  <tr key={h.server_id}>
                    <td><strong>{h.name}</strong></td>
                    <td>{h.protocol}</td>
                    <td>
                      <Badge tone={healthTone(h.status)}>
                        {t('status_' + h.status, h.status)}
                      </Badge>
                    </td>
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
              <thead>
                <tr>
                  <th>{t('settings_col_server')}</th>
                  <th>{t('settings_col_status')}</th>
                  <th>{t('settings_col_retries')}</th>
                  <th>{t('settings_col_error')}</th>
                </tr>
              </thead>
              <tbody>
                {redeployResult.map((r) => (
                  <tr key={r.server_id}>
                    <td>{r.name}</td>
                    <td>
                      <Badge tone={r.status === 'ok' ? 'success' : 'danger'}>
                        {r.status}
                      </Badge>
                    </td>
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
