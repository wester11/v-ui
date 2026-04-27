import { useEffect, useMemo, useState } from 'react'
import { api, ApiError } from '../api/client'
import type { Config, CreateConfigRequest, Server } from '../types'
import { Badge, Button, Empty, Input, Modal, Skeleton, toast } from '../components/ui'
import { formatRelative } from '../lib/format'

type SetupMode = 'simple' | 'advanced'
type RoutingMode = 'simple' | 'advanced' | 'cascade'
type Template = 'vless_reality' | 'grpc_reality' | 'cascade' | 'empty'

interface FormState {
  server_id: string
  name: string
  template: Template
  setup_mode: SetupMode
  routing_mode: RoutingMode
  activate: boolean
  inbound_port: number
  sni: string
  dest: string
  fingerprint: string
  flow: string
  short_ids_count: number
  raw_json: string
  cascade_upstream_id: string
  cascade_strategy: 'leastPing' | 'random'
  rule_ru_direct: boolean
  rule_non_ru_proxy: boolean
}

const empty: FormState = {
  server_id: '',
  name: '',
  template: 'vless_reality',
  setup_mode: 'simple',
  routing_mode: 'simple',
  activate: true,
  inbound_port: 443,
  sni: 'www.cloudflare.com',
  dest: 'www.cloudflare.com:443',
  fingerprint: 'chrome',
  flow: 'xtls-rprx-vision',
  short_ids_count: 3,
  raw_json: '{\n  "inbounds": [],\n  "outbounds": [],\n  "routing": {"rules": []}\n}',
  cascade_upstream_id: '',
  cascade_strategy: 'leastPing',
  rule_ru_direct: true,
  rule_non_ru_proxy: true,
}

export default function Configs() {
  const [servers, setServers] = useState<Server[] | null>(null)
  const [configs, setConfigs] = useState<Config[] | null>(null)
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState<FormState>(empty)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const [serverFilter, setServerFilter] = useState('')

  const currentServer = useMemo(
    () => servers?.find((s) => s.id === serverFilter) ?? null,
    [servers, serverFilter],
  )

  const xrayServers = useMemo(
    () => (servers ?? []).filter((s) => s.protocol === 'xray'),
    [servers],
  )

  async function loadServers() {
    try {
      const list = await api.servers.list()
      setServers(list)
      if (!serverFilter && list.length > 0) setServerFilter(list[0].id)
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'failed to load servers')
    }
  }

  async function loadConfigs(serverID: string) {
    if (!serverID) {
      setConfigs([])
      return
    }
    try {
      setConfigs(null)
      setConfigs(await api.configs.listByServer(serverID))
    } catch (e) {
      setConfigs([])
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'failed to load configs')
    }
  }

  useEffect(() => { loadServers() }, [])
  useEffect(() => { if (serverFilter) loadConfigs(serverFilter) }, [serverFilter])

  const openCreate = () => {
    const first = servers?.[0]?.id ?? ''
    setForm({ ...empty, server_id: first, cascade_upstream_id: first })
    setCreating(true)
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErr(null)
    setBusy(true)
    try {
      const body: CreateConfigRequest = {
        server_id: form.server_id,
        name: form.name.trim(),
        protocol: 'xray',
        template: form.template,
        setup_mode: form.setup_mode,
        routing_mode: form.routing_mode,
        activate: form.activate,
      }
      if (form.setup_mode === 'simple') {
        body.inbound_port = form.inbound_port
        body.sni = form.sni.trim()
        body.dest = form.dest.trim()
        body.fingerprint = form.fingerprint.trim()
        body.flow = form.flow.trim()
        body.short_ids_count = form.short_ids_count
      } else {
        body.raw_json = form.raw_json
      }
      if (form.routing_mode === 'cascade') {
        body.cascade_upstream_id = form.cascade_upstream_id
        body.cascade_strategy = form.cascade_strategy
        const rules: Array<{ match: string; outbound: 'direct' | 'proxy' }> = []
        if (form.rule_ru_direct) rules.push({ match: 'geoip:ru', outbound: 'direct' })
        if (form.rule_non_ru_proxy) rules.push({ match: 'geoip:!ru', outbound: 'proxy' })
        body.cascade_rules = rules
      }

      await api.configs.create(body)
      toast.success('Config saved')
      setCreating(false)
      if (serverFilter) await loadConfigs(serverFilter)
      await loadServers()
    } catch (e) {
      setErr(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'create failed')
    } finally {
      setBusy(false)
    }
  }

  const activate = async (id: string) => {
    try {
      await api.configs.activate(id)
      toast.success('Config activated and deployed')
      if (serverFilter) await loadConfigs(serverFilter)
      await loadServers()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'activate failed')
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">Configs</div>
          <div className="page-sub">Create VPN logic separately from infrastructure servers.</div>
        </div>
        <div className="row" style={{ flexWrap: 'wrap' }}>
          <select
            className="select"
            value={serverFilter}
            onChange={(e) => setServerFilter(e.target.value)}
            style={{ minWidth: 260 }}
          >
            {(servers ?? []).map((s) => (
              <option key={s.id} value={s.id}>
                {s.name} ({s.status})
              </option>
            ))}
          </select>
          <Button variant="primary" onClick={openCreate}>+ Create config</Button>
        </div>
      </div>

      {!currentServer ? (
        <div className="card">
          <Empty title="No servers found" sub="Create server first in Servers page." />
        </div>
      ) : (
        <div className="card">
          <div className="card-header">
            <div>
              <div className="card-title">{currentServer.name}</div>
              <div className="card-sub">Active protocol: {currentServer.protocol || 'none'}</div>
            </div>
          </div>

          {configs === null ? (
            <div className="stack">
              {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} height={38} />)}
            </div>
          ) : configs.length === 0 ? (
            <Empty title="No configs on this server" sub="Create your first Xray config to enable clients." />
          ) : (
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Template</th>
                    <th>Mode</th>
                    <th>Routing</th>
                    <th>Status</th>
                    <th>Updated</th>
                    <th className="actions">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {configs.map((cfg) => (
                    <tr key={cfg.id}>
                      <td><strong>{cfg.name}</strong></td>
                      <td><Badge>{cfg.template}</Badge></td>
                      <td>{cfg.setup_mode}</td>
                      <td>{cfg.routing_mode}</td>
                      <td>{cfg.is_active ? <Badge tone="success">active</Badge> : <Badge tone="warn">inactive</Badge>}</td>
                      <td className="text-dim">{formatRelative(cfg.updated_at)}</td>
                      <td className="actions">
                        {!cfg.is_active && <Button size="sm" onClick={() => activate(cfg.id)}>Activate</Button>}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      <Modal
        open={creating}
        onClose={() => setCreating(false)}
        title="Create Xray config"
        footer={(
          <>
            <Button variant="ghost" onClick={() => setCreating(false)}>Cancel</Button>
            <Button variant="primary" onClick={submit as never} loading={busy}>Save config</Button>
          </>
        )}
      >
        <form className="stack" onSubmit={submit}>
          <Input
            label="Name"
            placeholder="RU gateway profile"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            required
          />
          <div className="row">
            <div className="stack-sm" style={{ flex: 1 }}>
              <label className="label">Server</label>
              <select
                className="select"
                value={form.server_id}
                onChange={(e) => setForm({ ...form, server_id: e.target.value })}
              >
                {(servers ?? []).map((s) => (
                  <option key={s.id} value={s.id}>{s.name}</option>
                ))}
              </select>
            </div>
            <div className="stack-sm" style={{ flex: 1 }}>
              <label className="label">Template</label>
              <select
                className="select"
                value={form.template}
                onChange={(e) => {
                  const template = e.target.value as Template
                  setForm({
                    ...form,
                    template,
                    routing_mode: template === 'cascade' ? 'cascade' : form.routing_mode,
                  })
                }}
              >
                <option value="vless_reality">VLESS Reality (default)</option>
                <option value="grpc_reality">gRPC Reality</option>
                <option value="cascade">Cascade config</option>
                <option value="empty">Empty config</option>
              </select>
            </div>
          </div>

          <div className="row">
            <div className="stack-sm" style={{ flex: 1 }}>
              <label className="label">Mode</label>
              <select
                className="select"
                value={form.setup_mode}
                onChange={(e) => setForm({ ...form, setup_mode: e.target.value as SetupMode })}
              >
                <option value="simple">Simple</option>
                <option value="advanced">Advanced (raw JSON)</option>
              </select>
            </div>
            <div className="stack-sm" style={{ flex: 1 }}>
              <label className="label">Routing mode</label>
              <select
                className="select"
                value={form.routing_mode}
                onChange={(e) => setForm({ ...form, routing_mode: e.target.value as RoutingMode })}
              >
                <option value="simple">Simple</option>
                <option value="advanced">Advanced</option>
                <option value="cascade">Cascade</option>
              </select>
            </div>
          </div>

          {form.setup_mode === 'simple' ? (
            <>
              <div className="row">
                <Input
                  label="Port"
                  type="number"
                  value={form.inbound_port}
                  onChange={(e) => setForm({ ...form, inbound_port: Number(e.target.value || 443) })}
                />
                <Input
                  label="Short IDs count"
                  type="number"
                  value={form.short_ids_count}
                  onChange={(e) => setForm({ ...form, short_ids_count: Number(e.target.value || 3) })}
                />
              </div>
              <Input label="SNI" value={form.sni} onChange={(e) => setForm({ ...form, sni: e.target.value })} />
              <Input label="Dest" value={form.dest} onChange={(e) => setForm({ ...form, dest: e.target.value })} />
              <div className="row">
                <Input label="Fingerprint" value={form.fingerprint} onChange={(e) => setForm({ ...form, fingerprint: e.target.value })} />
                <Input label="Flow" value={form.flow} onChange={(e) => setForm({ ...form, flow: e.target.value })} />
              </div>
            </>
          ) : (
            <div className="stack-sm">
              <label className="label">Raw Xray JSON</label>
              <textarea
                className="textarea"
                rows={12}
                value={form.raw_json}
                onChange={(e) => setForm({ ...form, raw_json: e.target.value })}
              />
            </div>
          )}

          {form.routing_mode === 'cascade' && (
            <div className="state-panel">
              <div className="row">
                <div className="stack-sm" style={{ flex: 1 }}>
                  <label className="label">Upstream server</label>
                  <select
                    className="select"
                    value={form.cascade_upstream_id}
                    onChange={(e) => setForm({ ...form, cascade_upstream_id: e.target.value })}
                  >
                    <option value="">Select upstream...</option>
                    {xrayServers.filter((s) => s.id !== form.server_id).map((s) => (
                      <option key={s.id} value={s.id}>{s.name}</option>
                    ))}
                  </select>
                </div>
                <div className="stack-sm" style={{ width: 180 }}>
                  <label className="label">Strategy</label>
                  <select
                    className="select"
                    value={form.cascade_strategy}
                    onChange={(e) => setForm({ ...form, cascade_strategy: e.target.value as 'leastPing' | 'random' })}
                  >
                    <option value="leastPing">leastPing</option>
                    <option value="random">random</option>
                  </select>
                </div>
              </div>
              <label className="row gap-2" style={{ cursor: 'pointer' }}>
                <input
                  type="checkbox"
                  checked={form.rule_ru_direct}
                  onChange={(e) => setForm({ ...form, rule_ru_direct: e.target.checked })}
                />
                <span>geoip:ru - direct</span>
              </label>
              <label className="row gap-2" style={{ cursor: 'pointer' }}>
                <input
                  type="checkbox"
                  checked={form.rule_non_ru_proxy}
                  onChange={(e) => setForm({ ...form, rule_non_ru_proxy: e.target.checked })}
                />
                <span>geoip:!ru - proxy</span>
              </label>
            </div>
          )}

          <label className="row gap-2" style={{ cursor: 'pointer' }}>
            <input
              type="checkbox"
              checked={form.activate}
              onChange={(e) => setForm({ ...form, activate: e.target.checked })}
            />
            <span>Activate immediately (auto-deploy)</span>
          </label>

          {err && <div className="text-danger text-sm">{err}</div>}
        </form>
      </Modal>
    </div>
  )
}

