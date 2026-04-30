import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import { useI18n } from '../i18n'
import type { CreateServerResponse, NodeCheckResult, Server } from '../types'
import {
  Badge, Button, ConfirmDialog, CopyField, Empty, Input, Modal, Skeleton, toast,
} from '../components/ui'
import { formatRelative } from '../lib/format'

interface FormState { name: string; endpoint: string }
const emptyForm: FormState = { name: '', endpoint: '' }

function statusTone(s: string | undefined): 'success' | 'warn' | 'danger' | 'default' {
  if (!s) return 'default'
  if (s === 'online') return 'success'
  if (s === 'pending' || s === 'deploying' || s === 'drifted') return 'warn'
  if (s === 'error') return 'danger'
  return 'default'
}

function ServerListCard({
  server, t, onCheck, onRemove, checkLoading,
}: {
  server: Server
  t: (k: string, fb?: string) => string
  onCheck: (id: string) => void
  onRemove: (s: Server) => void
  checkLoading: boolean
}) {
  const status   = server.status ?? (server.online ? 'online' : 'offline')
  const tone     = statusTone(status)
  const isOnline = server.online

  const stripeColor =
    tone === 'success' ? 'var(--success)'
    : tone === 'danger'  ? 'var(--danger)'
    : tone === 'warn'    ? 'var(--warning)'
    : 'var(--border-strong)'

  return (
    <div className="server-card">
      <div className="server-card-stripe" style={{ background: stripeColor }} />
      <div className="server-card-body">
        <div className="server-card-header">
          <div className="row gap-2">
            <span className="server-card-dot" style={{
              background: isOnline ? 'var(--success)' : 'var(--text-mute)',
              boxShadow: isOnline ? '0 0 8px var(--success)' : 'none',
            }} />
            <Link to={`/servers/${server.id}`} style={{ textDecoration: 'none' }}>
              <strong className="server-card-name">{server.name}</strong>
            </Link>
          </div>
          <Badge tone={tone}>{t('status_' + status, status)}</Badge>
        </div>

        <div className="server-card-meta">
          <span>{server.ip || server.endpoint}</span>
          {server.hostname && <span className="text-mute" style={{ marginLeft: 4 }}>· {server.hostname}</span>}
        </div>

        <div className="row gap-2" style={{ flexWrap: 'wrap' }}>
          {server.protocol && server.protocol !== 'none' && (
            <Badge tone="violet">{server.protocol}</Badge>
          )}
          {server.mode && <Badge>{server.mode}</Badge>}
          {server.agent_version && (
            <span className="text-xs text-mute" style={{ fontFamily: 'var(--font-mono)' }}>
              {server.agent_version}
            </span>
          )}
        </div>

        <div className="server-card-footer">
          <span className="text-xs text-mute">
            {t('servers_last_beat')}: {formatRelative(server.last_heartbeat)}
          </span>
        </div>

        <div className="row" style={{ gap: 8, paddingTop: 10, borderTop: '1px solid var(--border)', marginTop: 4, flexWrap: 'wrap' }}>
          <Link to={`/servers/${server.id}`} style={{ flex: 1 }}>
            <Button size="sm" style={{ width: '100%' }}>{t('servers_open')}</Button>
          </Link>
          <Button size="sm" onClick={() => onCheck(server.id)} loading={checkLoading}>
            {t('action_check')}
          </Button>
          <Button size="sm" variant="danger" onClick={() => onRemove(server)}>
            {t('servers_remove')}
          </Button>
        </div>
      </div>
    </div>
  )
}

// Pulsing dot animation for active polling
function PollDot() {
  return (
    <span style={{
      display: 'inline-block', width: 8, height: 8, borderRadius: '50%',
      background: 'var(--accent)', marginRight: 6,
      animation: 'pulse-dot 1.2s ease-in-out infinite',
    }} />
  )
}

export default function Servers() {
  const { t } = useI18n()
  const [list,         setList]         = useState<Server[] | null>(null)
  const [search,       setSearch]       = useState('')
  const [creating,     setCreating]     = useState(false)
  const [form,         setForm]         = useState<FormState>(emptyForm)
  const [busy,         setBusy]         = useState(false)
  const [err,          setErr]          = useState<string | null>(null)
  const [onboarding,   setOnboarding]   = useState<CreateServerResponse | null>(null)
  const [checkResult,  setCheckResult]  = useState<NodeCheckResult | null>(null)
  const [checkLoading, setCheckLoading] = useState(false)
  const [confirm,      setConfirm]      = useState<Server | null>(null)
  const [delBusy,      setDelBusy]      = useState(false)
  const [polling,      setPolling]      = useState(false)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const filtered = useMemo(() => {
    if (!list) return null
    const q = search.toLowerCase()
    if (!q) return list
    return list.filter((s) =>
      s.name.toLowerCase().includes(q) ||
      (s.ip || s.endpoint || '').toLowerCase().includes(q)
    )
  }, [list, search])

  async function reload() {
    try {
      setList(await api.servers.list())
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'load failed')
    }
  }

  useEffect(() => { reload() }, [])

  // Auto-poll every 5s while onboarding modal is open
  useEffect(() => {
    if (!onboarding) {
      stopPolling()
      return
    }
    startPolling(onboarding.id)
    return () => stopPolling()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [onboarding?.id])

  function startPolling(serverID: string) {
    stopPolling()
    setPolling(true)
    pollRef.current = setInterval(() => {
      silentCheck(serverID)
    }, 5000)
  }

  function stopPolling() {
    if (pollRef.current) {
      clearInterval(pollRef.current)
      pollRef.current = null
    }
    setPolling(false)
  }

  async function silentCheck(serverID: string) {
    try {
      const res = await api.servers.check(serverID)
      setCheckResult(res)
      if (res.status === 'online') {
        stopPolling()
        await reload()
        toast.success(t('servers_onboard_connected'))
      }
    } catch {
      // silently ignore errors during background polling
    }
  }

  async function submit(e?: React.FormEvent) {
    e?.preventDefault()
    if (!form.name.trim() || !form.endpoint.trim()) return
    setBusy(true)
    setErr(null)
    try {
      const resp = await api.servers.create({
        name:     form.name.trim(),
        endpoint: form.endpoint.trim(),
      })
      setOnboarding(resp)
      setCheckResult(null)
      setForm(emptyForm)
      setCreating(false)
      await reload()
      toast.success(t('action_create') + ' ✓')
    } catch (e) {
      setErr(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'create failed')
    } finally {
      setBusy(false)
    }
  }

  async function checkConnection(serverID: string) {
    setCheckLoading(true)
    try {
      const res = await api.servers.check(serverID)
      setCheckResult(res)
      if (res.status === 'online') {
        stopPolling()
        toast.success(t('status_online'))
      } else {
        toast.warn(t('status_offline'))
      }
      await reload()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'check failed')
    } finally {
      setCheckLoading(false)
    }
  }

  async function remove() {
    if (!confirm) return
    setDelBusy(true)
    try {
      await api.servers.delete(confirm.id)
      toast.success(t('action_delete') + ' ✓')
      setConfirm(null)
      await reload()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'delete failed')
    } finally {
      setDelBusy(false)
    }
  }

  const isOnline = checkResult?.status === 'online'

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">{t('servers_title')}</div>
          <div className="page-sub">{t('servers_sub')}</div>
        </div>
        <div className="row" style={{ flexWrap: 'wrap', gap: 10 }}>
          <input
            className="search-input"
            placeholder={t('servers_search')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Button variant="primary" onClick={() => setCreating(true)}>{t('servers_add')}</Button>
        </div>
      </div>

      {list === null ? (
        <div className="servers-grid">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="server-card">
              <div className="server-card-stripe" style={{ background: 'var(--border-strong)' }} />
              <div className="server-card-body" style={{ gap: 10 }}>
                <Skeleton height={18} />
                <Skeleton height={14} width="60%" />
                <Skeleton height={14} width="40%" />
              </div>
            </div>
          ))}
        </div>
      ) : filtered && filtered.length === 0 ? (
        <Empty
          title={search ? t('servers_no_match') : t('servers_empty')}
          sub={search ? t('servers_no_match_sub') : t('servers_empty_sub')}
          action={!search
            ? <Button variant="primary" onClick={() => setCreating(true)}>{t('servers_add')}</Button>
            : undefined
          }
        />
      ) : (
        <div className="servers-grid">
          {filtered!.map((s) => (
            <ServerListCard
              key={s.id}
              server={s}
              t={t}
              onCheck={checkConnection}
              onRemove={setConfirm}
              checkLoading={checkLoading}
            />
          ))}
        </div>
      )}

      <Modal
        open={creating}
        onClose={() => { setCreating(false); setErr(null) }}
        title={t('servers_create_title')}
        footer={(
          <>
            <Button variant="ghost" onClick={() => { setCreating(false); setErr(null) }}>
              {t('action_cancel')}
            </Button>
            <Button variant="primary" onClick={submit as never} loading={busy}>
              {t('action_create')}
            </Button>
          </>
        )}
      >
        <form className="stack" onSubmit={submit}>
          <Input
            label={t('servers_field_name')}
            placeholder={t('servers_field_name_ph')}
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            required
            autoFocus
          />
          <Input
            label={t('servers_field_host')}
            placeholder={t('servers_field_host_ph')}
            value={form.endpoint}
            onChange={(e) => setForm({ ...form, endpoint: e.target.value })}
            required
          />
          {err && <div className="text-danger text-sm">{err}</div>}
        </form>
      </Modal>

      {onboarding && (
        <Modal
          open
          onClose={() => { setOnboarding(null); stopPolling() }}
          title={`${t('servers_onboard_title')}: ${onboarding.name}`}
          width={560}
          footer={(
            <>
              {!isOnline && (
                <Button onClick={() => checkConnection(onboarding.id)} loading={checkLoading}>
                  {t('action_check')}
                </Button>
              )}
              {isOnline ? (
                <Button variant="primary" onClick={() => { setOnboarding(null); stopPolling() }}>
                  {t('servers_onboard_done')}
                </Button>
              ) : (
                <Button variant="ghost" onClick={() => { setOnboarding(null); stopPolling() }}>
                  {t('action_cancel')}
                </Button>
              )}
            </>
          )}
        >
          <div className="stack">
            <div className="state-panel">
              {/* Step 1 */}
              <div className="stack-sm">
                <div className="row gap-2">
                  <span className="onboard-step-num">1</span>
                  <strong>{t('servers_onboard_step1')}</strong>
                </div>
                <CopyField value={onboarding.install_command} />
              </div>

              {/* Step 2 */}
              <div className="stack-sm">
                <div className="row gap-2">
                  <span className="onboard-step-num">2</span>
                  <strong>{t('servers_onboard_step2')}</strong>
                </div>
                <pre style={{ margin: 0, fontSize: 12 }}>{onboarding.compose_snippet}</pre>
              </div>

              {/* Step 3 — connection status */}
              <div className="stack-sm">
                <div className="row gap-2">
                  <span className="onboard-step-num">3</span>
                  <strong>{t('servers_onboard_step3')}</strong>
                </div>
                {isOnline ? (
                  <div className="row gap-2" style={{ alignItems: 'center' }}>
                    <span style={{ color: 'var(--success)', fontSize: 18 }}>✓</span>
                    <Badge tone="success">{t('status_online')}</Badge>
                    <span className="text-sm text-mute">{t('servers_onboard_connected')}</span>
                  </div>
                ) : checkResult ? (
                  <div className="row gap-2" style={{ alignItems: 'center' }}>
                    <Badge tone="warn">{t('status_' + checkResult.status, checkResult.status)}</Badge>
                    <span className="text-sm text-mute">{t('servers_onboard_wait')}</span>
                  </div>
                ) : (
                  <div className="row gap-2" style={{ alignItems: 'center' }}>
                    {polling && <PollDot />}
                    <span className="text-sm text-mute">
                      {polling ? t('servers_onboard_polling') : t('servers_onboard_wait')}
                    </span>
                  </div>
                )}
              </div>
            </div>

            {/* Polling status bar */}
            {polling && !isOnline && (
              <div className="onboard-poll-bar">
                <div className="onboard-poll-fill" />
              </div>
            )}

            <div className="text-xs text-mute">
              {t('servers_onboard_hint')} → <Link to="/configs">{t('nav_configs')}</Link>
            </div>
          </div>
        </Modal>
      )}

      <ConfirmDialog
        open={!!confirm}
        title={t('servers_remove_title')}
        body={(
          <>
            {t('servers_remove_body')} <strong>{confirm?.name}</strong>.
          </>
        )}
        confirmText={t('servers_remove')}
        destructive
        loading={delBusy}
        onConfirm={remove}
        onClose={() => setConfirm(null)}
      />
    </div>
  )
}
