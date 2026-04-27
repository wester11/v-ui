import { useEffect, useState } from 'react'
import { api, ApiError } from '../api/client'
import type { FleetHealthResult, FleetRedeployResult } from '../types'
import { Badge, Button, Empty, Skeleton, toast } from '../components/ui'
import { formatRelative } from '../lib/format'
import { useI18n } from '../i18n'

function healthTone(status: string): 'success' | 'danger' | 'warn' {
  if (status === 'online') return 'success'
  if (status === 'offline') return 'danger'
  return 'warn'
}

export default function Settings() {
  const { locale, setLocale, t } = useI18n()
  const [health, setHealth] = useState<FleetHealthResult[] | null>(null)
  const [redeployResult, setRedeployResult] = useState<FleetRedeployResult[] | null>(null)
  const [loading, setLoading] = useState(false)
  const [redeploying, setRedeploying] = useState(false)

  const loadHealth = async () => {
    setLoading(true)
    try {
      const data = await api.fleet.health()
      setHealth(data)
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'health check failed')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { loadHealth() }, [])

  const redeployAll = async () => {
    setRedeploying(true)
    setRedeployResult(null)
    try {
      const result = await api.fleet.redeployAll()
      setRedeployResult(result)
      const failed = result.filter((r) => r.status === 'error').length
      if (failed === 0) toast.success('Fleet redeploy finished successfully')
      else toast.warn(`Redeploy finished with ${failed} failed nodes`)
      await loadHealth()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'redeploy failed')
    } finally {
      setRedeploying(false)
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">Settings</div>
          <div className="page-sub">Fleet-wide operations, health monitoring and recovery controls.</div>
        </div>
        <div className="row">
          <Button onClick={loadHealth} loading={loading}>Refresh health</Button>
          <Button variant="primary" onClick={redeployAll} loading={redeploying}>Redeploy all Xray nodes</Button>
        </div>
      </div>

      <div className="card mb-4">
        <div className="card-header">
          <div>
            <div className="card-title">{t('settings_language', 'Language')}</div>
            <div className="card-sub">{t('settings_language_hint', 'Choose UI language')}</div>
          </div>
        </div>
        <div className="row">
          <select className="select" value={locale} onChange={(e) => setLocale(e.target.value as 'en' | 'ru')}>
            <option value="en">English</option>
            <option value="ru">Русский</option>
          </select>
        </div>
      </div>

      <div className="card mb-4">
        <div className="card-header">
          <div>
            <div className="card-title">Health monitoring</div>
            <div className="card-sub">Runtime probe + heartbeat state. Status can be online, offline or degraded.</div>
          </div>
        </div>

        {health === null ? (
          <div className="stack">
            {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} height={36} />)}
          </div>
        ) : health.length === 0 ? (
          <Empty title="No nodes yet" sub="Register servers first." />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Server</th>
                  <th>Protocol</th>
                  <th>Status</th>
                  <th>Last heartbeat</th>
                  <th>Error</th>
                </tr>
              </thead>
              <tbody>
                {health.map((h) => (
                  <tr key={h.server_id}>
                    <td><strong>{h.name}</strong></td>
                    <td>{h.protocol}</td>
                    <td><Badge tone={healthTone(h.status)}>{h.status}</Badge></td>
                    <td className="text-dim">{formatRelative(h.last_heartbeat)}</td>
                    <td className="text-danger text-sm">{h.error || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="card">
        <div className="card-header">
          <div>
            <div className="card-title">Last bulk operation result</div>
            <div className="card-sub">Redeploy status for each Xray node.</div>
          </div>
        </div>
        {!redeployResult ? (
          <Empty title="No operation executed" sub="Run Redeploy all Xray nodes to see a report." />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Server</th>
                  <th>Status</th>
                  <th>Retries</th>
                  <th>Error</th>
                </tr>
              </thead>
              <tbody>
                {redeployResult.map((r) => (
                  <tr key={r.server_id}>
                    <td>{r.name}</td>
                    <td><Badge tone={r.status === 'ok' ? 'success' : 'danger'}>{r.status}</Badge></td>
                    <td>{r.retries}</td>
                    <td className="text-danger text-sm">{r.error || '-'}</td>
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
