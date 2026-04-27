import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import type { CreateServerResponse, Server } from '../types'
import {
  Badge, Button, ConfirmDialog, CopyField, Empty, Input, Modal, Skeleton, StatusDot,
  toast,
} from '../components/ui'
import { formatRelative, maskKey } from '../lib/format'

type Protocol = 'wireguard' | 'amneziawg' | 'xray'
type Mode = 'standalone' | 'cascade'

interface FormState {
  name: string
  protocol: Protocol
  mode: Mode
  endpoint: string
  listen_port: number
  subnet: string
  dns: string
  obfs_enabled: boolean

  xray_inbound_port: number
  xray_sni: string
  xray_dest: string
  xray_short_ids: number
  xray_fingerprint: string

  cascade_upstream_id: string
  rule_ru_direct: boolean
  rule_non_ru_proxy: boolean
}

const empty: FormState = {
  name: '',
  protocol: 'xray',
  mode: 'standalone',
  endpoint: '',
  listen_port: 51820,
  subnet: '10.10.0.0/24',
  dns: '1.1.1.1, 9.9.9.9',
  obfs_enabled: true,

  xray_inbound_port: 443,
  xray_sni: 'www.cloudflare.com',
  xray_dest: 'www.cloudflare.com:443',
  xray_short_ids: 3,
  xray_fingerprint: 'chrome',

  cascade_upstream_id: '',
  rule_ru_direct: true,
  rule_non_ru_proxy: true,
}

export default function Servers() {
  const [list, setList] = useState<Server[] | null>(null)
  const [search, setSearch] = useState('')
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState<FormState>(empty)
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const [token, setToken] = useState<{ name: string; token: string } | null>(null)
  const [confirm, setConfirm] = useState<Server | null>(null)
  const [delBusy, setDelBusy] = useState(false)

  async function reload() {
    try {
      setList(await api.servers.list())
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'load failed')
    }
  }

  useEffect(() => { reload() }, [])

  const filtered = useMemo(() => {
    if (!list) return null
    const q = search.trim().toLowerCase()
    if (!q) return list
    return list.filter((s) =>
      s.name.toLowerCase().includes(q) ||
      s.endpoint.toLowerCase().includes(q) ||
      (s.subnet ?? '').toLowerCase().includes(q),
    )
  }, [list, search])

  const xrayServers = useMemo(
    () => (list ?? []).filter((s) => s.protocol === 'xray'),
    [list],
  )

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setErr(null)
    setBusy(true)
    try {
      const dns = form.dns.split(',').map((s) => s.trim()).filter(Boolean)
      const body: any = {
        name: form.name.trim(),
        protocol: form.protocol,
        mode: form.protocol === 'xray' ? form.mode : 'standalone',
        endpoint: form.endpoint.trim(),
        obfs_enabled: form.obfs_enabled,
      }

      if (form.protocol === 'wireguard' || form.protocol === 'amneziawg') {
        body.listen_port = form.listen_port
        body.subnet = form.subnet.trim()
        body.dns = dns
      }

      if (form.protocol === 'xray') {
        body.xray_inbound_port = form.xray_inbound_port
        body.xray_sni = form.xray_sni.trim()
        body.xray_dest = form.xray_dest.trim()
        body.xray_short_ids = form.xray_short_ids
        body.xray_fingerprint = form.xray_fingerprint

        if (form.mode === 'cascade') {
          body.cascade_upstream_id = form.cascade_upstream_id
          const rules: Array<{ match: string; outbound: 'direct' | 'proxy' }> = []
          if (form.rule_ru_direct) rules.push({ match: 'geoip:ru', outbound: 'direct' })
          if (form.rule_non_ru_proxy) rules.push({ match: 'geoip:!ru', outbound: 'proxy' })
          body.cascade_rules = rules
        }
      }

      const resp: CreateServerResponse = await api.servers.create(body)
      if (resp.agent_token) setToken({ name: resp.name, token: resp.agent_token })
      setForm(empty)
      setCreating(false)
      await reload()
      toast.success('Server registered')
    } catch (e) {
      setErr(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'create failed')
    } finally {
      setBusy(false)
    }
  }

  async function remove() {
    if (!confirm) return
    setDelBusy(true)
    try {
      await api.servers.delete(confirm.id)
      toast.success('Server removed')
      setConfirm(null)
      await reload()
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'delete failed')
    } finally {
      setDelBusy(false)
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">Servers</div>
          <div className="page-sub">Register nodes, set routing mode, and monitor health.</div>
        </div>
        <div className="row" style={{ flexWrap: 'wrap' }}>
          <input
            className="search-input"
            placeholder="Search servers..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Button variant="primary" onClick={() => setCreating(true)}>+ Register server</Button>
        </div>
      </div>

      <div className="card">
        {list === null ? (
          <div className="stack">
            {Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} height={42} />)}
          </div>
        ) : filtered && filtered.length === 0 ? (
          <Empty
            title={search ? 'No servers match your search' : 'No servers yet'}
            sub={search ? 'Try a different query.' : 'Register your first VPN node so clients can be provisioned.'}
            action={!search ? <Button variant="primary" onClick={() => setCreating(true)}>+ Register server</Button> : undefined}
          />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Protocol</th>
                  <th>Mode</th>
                  <th>Endpoint</th>
                  <th>Subnet</th>
                  <th>Status</th>
                  <th>Last heartbeat</th>
                  <th>Public key</th>
                  <th className="actions">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered!.map((s) => (
                  <tr key={s.id}>
                    <td><Link to={`/servers/${s.id}`}><strong>{s.name}</strong></Link></td>
                    <td><Badge tone="info">{s.protocol}</Badge></td>
                    <td>{s.mode === 'cascade' ? <Badge tone="violet">cascade</Badge> : <Badge>standalone</Badge>}</td>
                    <td><code>{s.endpoint}</code></td>
                    <td><code>{s.subnet || '-'}</code></td>
                    <td><StatusDot online={s.online} /></td>
                    <td className="text-dim">{formatRelative(s.last_heartbeat)}</td>
                    <td><code>{maskKey(s.public_key ?? s.xray_public_key)}</code></td>
                    <td className="actions">
                      <div className="row-end">
                        <Link to={`/servers/${s.id}`}><Button size="sm">Open</Button></Link>
                        <Button size="sm" variant="danger" onClick={() => setConfirm(s)}>Remove</Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <Modal
        open={creating}
        onClose={() => setCreating(false)}
        title="Register VPN node"
        footer={
          <>
            <Button variant="ghost" onClick={() => setCreating(false)}>Cancel</Button>
            <Button variant="primary" onClick={submit as never} loading={busy}>Register</Button>
          </>
        }
      >
        <form className="stack" onSubmit={submit}>
          <Input
            label="Name"
            placeholder="eu-gateway-1"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            required
            autoFocus
          />

          <div className="row" style={{ alignItems: 'flex-start' }}>
            <div className="stack-sm" style={{ flex: 1 }}>
              <label className="label">Protocol</label>
              <select
                className="select"
                value={form.protocol}
                onChange={(e) => setForm({ ...form, protocol: e.target.value as Protocol, mode: 'standalone' })}
              >
                <option value="wireguard">WireGuard</option>
                <option value="amneziawg">AmneziaWG (UDP obfuscation)</option>
                <option value="xray">Xray (VLESS + Reality)</option>
              </select>
            </div>

            <div className="stack-sm" style={{ flex: 1 }}>
              <label className="label">Mode <span className="text-mute" title="Cascade is available only for Xray protocol">(?)</span></label>
              <select
                className="select"
                value={form.mode}
                disabled={form.protocol !== 'xray'}
                onChange={(e) => setForm({ ...form, mode: e.target.value as Mode })}
              >
                <option value="standalone">Standalone</option>
                <option value="cascade">Cascade (multi-hop)</option>
              </select>
            </div>
          </div>

          <Input
            label="Endpoint (host:port)"
            placeholder={form.protocol === 'xray' ? 'vpn.example.com:443' : 'vpn.example.com:51820'}
            value={form.endpoint}
            onChange={(e) => setForm({ ...form, endpoint: e.target.value })}
            required
          />

          {(form.protocol === 'wireguard' || form.protocol === 'amneziawg') && (
            <>
              <div className="row">
                <Input
                  label="Listen port (UDP)"
                  type="number"
                  min={1}
                  max={65535}
                  value={form.listen_port}
                  onChange={(e) => setForm({ ...form, listen_port: parseInt(e.target.value || '0', 10) })}
                  required
                />
                <Input
                  label="Subnet (CIDR)"
                  placeholder="10.10.0.0/24"
                  value={form.subnet}
                  onChange={(e) => setForm({ ...form, subnet: e.target.value })}
                  required
                />
              </div>
              <Input
                label="DNS (comma-separated)"
                placeholder="1.1.1.1, 9.9.9.9"
                value={form.dns}
                onChange={(e) => setForm({ ...form, dns: e.target.value })}
              />
              <label className="row gap-2" style={{ cursor: 'pointer' }}>
                <input
                  type="checkbox"
                  checked={form.obfs_enabled || form.protocol === 'amneziawg'}
                  disabled={form.protocol === 'amneziawg'}
                  onChange={(e) => setForm({ ...form, obfs_enabled: e.target.checked })}
                />
                <span>Enable UDP obfuscation</span>
              </label>
            </>
          )}

          {form.protocol === 'xray' && (
            <>
              <div className="row">
                <Input
                  label="Inbound port (TCP)"
                  type="number"
                  min={1}
                  max={65535}
                  value={form.xray_inbound_port}
                  onChange={(e) => setForm({ ...form, xray_inbound_port: parseInt(e.target.value || '443', 10) })}
                />
                <Input
                  label="ShortIDs pool size"
                  type="number"
                  min={1}
                  max={16}
                  value={form.xray_short_ids}
                  onChange={(e) => setForm({ ...form, xray_short_ids: parseInt(e.target.value || '3', 10) })}
                />
              </div>

              <Input
                label="Reality SNI"
                placeholder="www.cloudflare.com"
                value={form.xray_sni}
                onChange={(e) => setForm({ ...form, xray_sni: e.target.value })}
              />
              <Input
                label="Reality dest (host:port)"
                placeholder="www.cloudflare.com:443"
                value={form.xray_dest}
                onChange={(e) => setForm({ ...form, xray_dest: e.target.value })}
              />

              <div className="stack-sm">
                <label className="label">Fingerprint</label>
                <select
                  className="select"
                  value={form.xray_fingerprint}
                  onChange={(e) => setForm({ ...form, xray_fingerprint: e.target.value })}
                >
                  <option value="chrome">chrome</option>
                  <option value="firefox">firefox</option>
                  <option value="safari">safari</option>
                  <option value="random">random</option>
                </select>
              </div>

              {form.mode === 'cascade' && (
                <div className="state-panel" style={{ gap: 12 }}>
                  <div className="text-sm"><strong>Cascade routing</strong>: split traffic by geo rules and send non-local through upstream.</div>
                  <div className="stack-sm">
                    <label className="label">Upstream server</label>
                    <select
                      className="select"
                      value={form.cascade_upstream_id}
                      onChange={(e) => setForm({ ...form, cascade_upstream_id: e.target.value })}
                      required
                    >
                      <option value="">Select upstream...</option>
                      {xrayServers.filter((s) => s.id !== confirm?.id).map((s) => (
                        <option key={s.id} value={s.id}>{s.name} ({s.endpoint})</option>
                      ))}
                    </select>
                  </div>

                  <label className="row gap-2" style={{ cursor: 'pointer' }}>
                    <input
                      type="checkbox"
                      checked={form.rule_ru_direct}
                      onChange={(e) => setForm({ ...form, rule_ru_direct: e.target.checked })}
                    />
                    <span><code>geoip:ru</code> - direct</span>
                  </label>
                  <label className="row gap-2" style={{ cursor: 'pointer' }}>
                    <input
                      type="checkbox"
                      checked={form.rule_non_ru_proxy}
                      onChange={(e) => setForm({ ...form, rule_non_ru_proxy: e.target.checked })}
                    />
                    <span><code>geoip:!ru</code> - proxy via upstream</span>
                  </label>
                </div>
              )}
            </>
          )}

          {err && <div className="text-danger text-sm">{err}</div>}
        </form>
      </Modal>

      {token && (
        <Modal
          open
          onClose={() => setToken(null)}
          title={`Agent token for "${token.name}"`}
          footer={<Button variant="primary" onClick={() => setToken(null)}>Got it</Button>}
        >
          <div className="stack">
            <div className="text-warn text-sm">
              Save this token now - it will not be shown again.
            </div>
            <CopyField value={token.token} mask />
            <div className="text-mono text-xs text-mute">
              Pass it as <code>AGENT_TOKEN</code> to the node agent.
            </div>
          </div>
        </Modal>
      )}

      <ConfirmDialog
        open={!!confirm}
        title="Remove server?"
        body={
          <>
            This will remove server <strong>{confirm?.name}</strong> and associated clients.
          </>
        }
        confirmText="Remove"
        destructive
        loading={delBusy}
        onConfirm={remove}
        onClose={() => setConfirm(null)}
      />
    </div>
  )
}
