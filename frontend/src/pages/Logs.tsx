import { useEffect, useState } from 'react'
import { api, ApiError } from '../api/client'
import type { AuditEntry } from '../types'
import { Badge, Button, Empty, Skeleton, toast } from '../components/ui'
import { formatDate } from '../lib/format'

function resultTone(result: string): 'success' | 'danger' | 'warn' {
  if (result === 'ok') return 'success'
  if (result === 'error') return 'danger'
  return 'warn'
}

export default function Logs() {
  const [items, setItems] = useState<AuditEntry[] | null>(null)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const load = async () => {
    setBusy(true)
    setErr(null)
    try {
      const data = await api.audit.list(100)
      setItems(data)
    } catch (e) {
      const msg = e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'failed to load logs'
      setErr(msg)
      toast.error(msg)
    } finally {
      setBusy(false)
    }
  }

  useEffect(() => { load() }, [])

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">Logs / Audit</div>
          <div className="page-sub">Security and operations timeline for control-plane actions.</div>
        </div>
        <Button onClick={load} loading={busy}>Refresh</Button>
      </div>

      {err && <div className="state-panel state-error mb-4">{err}</div>}

      <div className="card">
        {items === null ? (
          <div className="stack">
            {Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} height={36} />)}
          </div>
        ) : items.length === 0 ? (
          <Empty title="No audit entries" sub="Actions will appear here after first operations." />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Actor</th>
                  <th>Action</th>
                  <th>Target</th>
                  <th>Result</th>
                  <th>IP</th>
                </tr>
              </thead>
              <tbody>
                {items.map((it) => (
                  <tr key={it.id}>
                    <td className="text-dim">{formatDate(it.ts)}</td>
                    <td>{it.actor_email || '-'}</td>
                    <td><code>{it.action}</code></td>
                    <td>{it.target_type ? `${it.target_type}:${it.target_id || '-'}` : '-'}</td>
                    <td><Badge tone={resultTone(it.result)}>{it.result}</Badge></td>
                    <td className="text-mono text-dim">{it.ip || '-'}</td>
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
