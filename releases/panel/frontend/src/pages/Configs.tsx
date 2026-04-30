import { useEffect, useMemo, useState } from 'react'
import { api, ApiError } from '../api/client'
import { useI18n } from '../i18n'
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

function SectionHeader({ title, sub }: { title: string; sub?: string }) {
  return (
    <div style={{ borderBottom: '1px solid var(--border)', paddingBottom: 8, marginBottom: 4 }}>
      <div className="text-sm" style={{ fontWeight: 600, color: 'var(--text)' }}>{title}</div>
      {sub && <div className="text-xs text-mute" style={{ marginTop: 2 }}>{sub}</div>}
    </div>
  )
}

export default function Configs() {
  const { t } = useI18n()
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
    if (!serverID) { setConfigs([]); return }
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
    setErr(null)
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
      toast.success(t('toast_config_saved'))
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
      toast.success(t('toast_config_activated'))
      if (serverFilter) await loadConfigs(serverFilter)
      await loadServers()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : t('toast_activate_fail'))
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">{t('configs_title')}</div>
          <div className="page-sub">{t('configs_sub')}</div>
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
                {s.name} ({t('status_' + (s.status ?? 'offline'), s.status ?? 'offline')})
              </option>
            ))}
          </select>
          <Button variant="primary" onClick={openCreate}>{t('configs_create')}</Button>
        </div>
      </div>

      {!currentServer ? (
        <div className="card">
          <Empty title={t('configs_no_servers')} sub={t('configs_no_servers_sub')} />
        </div>
      ) : (
        <div className="card">
          <div className="card-header">
            <div>
              <div className="card-title">{currentServer.name}</div>
              <div className="card-sub">
                {t('configs_protocol_label')}: <strong>{currentServer.protocol || 'none'}</strong>
              </div>
            </div>
          </div>

          {configs === null ? (
            <div className="stack">
              {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} height={38} />)}
            </div>
          ) : configs.length === 0 ? (
            <Empty title={t('configs_no_configs')} sub={t('configs_no_configs_sub')} />
          ) : (
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>{t('configs_col_name')}</th>
                    <th>{t('configs_col_tpl')}</th>
                    <th>{t('configs_col_mode')}</th>
                    <th>{t('configs_col_routing')}</th>
                    <th>{t('configs_col_status')}</th>
                    <th>{t('configs_col_updated')}</th>
                    <th className="actions">{t('configs_col_actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {configs.map((cfg) => (
                    <tr key={cfg.id}>
                      <td><strong>{cfg.name}</strong></td>
                      <td><Badge>{cfg.template}</Badge></td>
                      <td>{cfg.setup_mode}</td>
                      <td>{cfg.routing_mode}</td>
                      <td>
                        {cfg.is_active
                          ? <Badge tone="success">{t('configs_status_active')}</Badge>
                          : <Badge tone="warn">{t('configs_status_inactive')}</Badge>
                        }
                      </td>
                      <td className="text-dim">{formatRelative(cfg.updated_at)}</td>
                      <td className="actions">
                        {!cfg.is_active && (
                          <Button size="sm" onClick={() => activate(cfg.id)}>
                            {t('configs_activate')}
                          </Button>
                        )}
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
        onClose={() => { setCreating(false); setErr(null) }}
        title={t('configs_modal_title')}
        width={560}
        footer={(
          <>
            <Button variant="ghost" onClick={() => { setCreating(false); setErr(null) }}>
              {t('action_cancel')}
            </Button>
            <Button variant="primary" onClick={submit as never} loading={busy}>
              {t('configs_save')}
            </Button>
          </>
        )}
      >
        <form className="stack" onSubmit={submit} style={{ gap: 16 }}>

          {/* Basic info */}
          <div className="stack" style={{ gap: 10 }}>
            <Input
              label={t('configs_field_name')}
              placeholder={t('configs_field_name_ph')}
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
              autoFocus
            />

            <div className="row" style={{ gap: 12 }}>
              <div className="stack-sm" style={{ flex: 1 }}>
                <label className="label">{t('configs_field_server')}</label>
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
                <label className="label">{t('configs_field_template')}</label>
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
                  <option value="vless_reality">{t('configs_tpl_vless')}</option>
                  <option value="grpc_reality">{t('configs_tpl_grpc')}</option>
                  <option value="cascade">{t('configs_tpl_cascade')}</option>
                  <option value="empty">{t('configs_tpl_empty')}</option>
                </select>
              </div>
            </div>

            <div className="row" style={{ gap: 12 }}>
              <div className="stack-sm" style={{ flex: 1 }}>
                <label className="label">{t('configs_field_mode')}</label>
                <select
                  className="select"
                  value={form.setup_mode}
                  onChange={(e) => setForm({ ...form, setup_mode: e.target.value as SetupMode })}
                >
                  <option value="simple">{t('configs_mode_simple')}</option>
                  <option value="advanced">{t('configs_mode_advanced')}</option>
                </select>
              </div>
              <div className="stack-sm" style={{ flex: 1 }}>
                <label className="label">{t('configs_field_routing')}</label>
                <select
                  className="select"
                  value={form.routing_mode}
                  onChange={(e) => setForm({ ...form, routing_mode: e.target.value as RoutingMode })}
                >
                  <option value="simple">{t('configs_routing_simple')}</option>
                  <option value="advanced">{t('configs_routing_advanced')}</option>
                  <option value="cascade">{t('configs_routing_cascade')}</option>
                </select>
              </div>
            </div>
          </div>

          {/* Inbound settings */}
          {form.setup_mode === 'simple' ? (
            <div className="stack" style={{ gap: 10 }}>
              <SectionHeader title={t('configs_section_inbound')} />
              <div className="row" style={{ gap: 12 }}>
                <Input
                  label={t('configs_field_port')}
                  type="number"
                  value={form.inbound_port}
                  onChange={(e) => setForm({ ...form, inbound_port: Number(e.target.value || 443) })}
                  style={{ width: 100 }}
                />
                <Input
                  label={t('configs_field_shortids')}
                  type="number"
                  value={form.short_ids_count}
                  onChange={(e) => setForm({ ...form, short_ids_count: Number(e.target.value || 3) })}
                  style={{ width: 80 }}
                />
              </div>
              <div className="row" style={{ gap: 12 }}>
                <Input
                  label={t('configs_field_sni')}
                  value={form.sni}
                  onChange={(e) => setForm({ ...form, sni: e.target.value })}
                  style={{ flex: 1 }}
                />
                <Input
                  label={t('configs_field_dest')}
                  value={form.dest}
                  onChange={(e) => setForm({ ...form, dest: e.target.value })}
                  style={{ flex: 1 }}
                />
              </div>
              <div className="row" style={{ gap: 12 }}>
                <Input
                  label={t('configs_field_fingerprint')}
                  value={form.fingerprint}
                  onChange={(e) => setForm({ ...form, fingerprint: e.target.value })}
                  style={{ flex: 1 }}
                />
                <Input
                  label={t('configs_field_flow')}
                  value={form.flow}
                  onChange={(e) => setForm({ ...form, flow: e.target.value })}
                  style={{ flex: 1 }}
                />
              </div>
            </div>
          ) : (
            <div className="stack" style={{ gap: 10 }}>
              <SectionHeader title={t('configs_field_raw')} />
              <textarea
                className="textarea"
                rows={12}
                spellCheck={false}
                value={form.raw_json}
                onChange={(e) => setForm({ ...form, raw_json: e.target.value })}
                style={{ fontFamily: 'var(--font-mono)', fontSize: 12, resize: 'vertical' }}
              />
            </div>
          )}

          {/* Cascade settings */}
          {form.routing_mode === 'cascade' && (
            <div className="stack" style={{ gap: 10 }}>
              <SectionHeader title={t('configs_section_cascade')} />
              <div className="state-panel" style={{ gap: 10 }}>
                <div className="row" style={{ gap: 12 }}>
                  <div className="stack-sm" style={{ flex: 1 }}>
                    <label className="label">{t('configs_cascade_upstream')}</label>
                    <select
                      className="select"
                      value={form.cascade_upstream_id}
                      onChange={(e) => setForm({ ...form, cascade_upstream_id: e.target.value })}
                    >
                      <option value="">{t('configs_cascade_upstream_ph')}</option>
                      {xrayServers.filter((s) => s.id !== form.server_id).map((s) => (
                        <option key={s.id} value={s.id}>{s.name}</option>
                      ))}
                    </select>
                  </div>
                  <div className="stack-sm" style={{ width: 160 }}>
                    <label className="label">{t('configs_cascade_strategy')}</label>
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

                <div className="stack-sm" style={{ gap: 6 }}>
                  <label className="row gap-2" style={{ cursor: 'pointer', alignItems: 'center' }}>
                    <input
                      type="checkbox"
                      checked={form.rule_ru_direct}
                      onChange={(e) => setForm({ ...form, rule_ru_direct: e.target.checked })}
                    />
                    <span className="text-sm">{t('configs_rule_ru_direct')}</span>
                  </label>
                  <label className="row gap-2" style={{ cursor: 'pointer', alignItems: 'center' }}>
                    <input
                      type="checkbox"
                      checked={form.rule_non_ru_proxy}
                      onChange={(e) => setForm({ ...form, rule_non_ru_proxy: e.target.checked })}
                    />
                    <span className="text-sm">{t('configs_rule_non_ru_proxy')}</span>
                  </label>
                </div>
              </div>
            </div>
          )}

          {/* Activate toggle */}
          <label className="row gap-2" style={{ cursor: 'pointer', alignItems: 'center', paddingTop: 4 }}>
            <input
              type="checkbox"
              checked={form.activate}
              onChange={(e) => setForm({ ...form, activate: e.target.checked })}
            />
            <span className="text-sm">{t('configs_activate_now')}</span>
          </label>

          {err && <div className="text-danger text-sm">{err}</div>}
        </form>
      </Modal>
    </div>
  )
}
