import { useMemo, useState } from 'react'
import { QRCodeSVG } from 'qrcode.react'
import { api, ApiError } from '../api/client'
import type { Peer } from '../types'
import { Button, Empty, Modal, Skeleton, copyToClipboard, downloadFile, toast } from '../components/ui'
import { formatRelative } from '../lib/format'

export default function Configs() {
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const [peers, setPeers] = useState<Peer[]>([])
  const [query, setQuery] = useState('')
  const [view, setView] = useState<{ peer: Peer; config: string } | null>(null)

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return peers
    return peers.filter((p) =>
      p.name.toLowerCase().includes(q) ||
      (p.assigned_ip ?? '').toLowerCase().includes(q),
    )
  }, [peers, query])

  const load = async () => {
    setLoading(true)
    try {
      const list = await api.peers.list()
      setPeers(list)
      setLoaded(true)
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'failed to load configs')
    } finally {
      setLoading(false)
    }
  }

  const openConfig = async (peer: Peer) => {
    try {
      const config = await api.peers.config(peer.id)
      setView({ peer, config })
    } catch (e) {
      toast.error(e instanceof ApiError ? (e.code ?? `HTTP ${e.status}`) : 'failed to load config')
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <div className="page-title">Configs / Access</div>
          <div className="page-sub">One-click access to client configs, QR codes and export.</div>
        </div>
        <div className="row" style={{ flexWrap: 'wrap' }}>
          <input
            className="search-input"
            placeholder="Search client..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            disabled={!loaded || loading}
          />
          <Button variant="primary" onClick={load} loading={loading}>Load configs</Button>
        </div>
      </div>

      <div className="card">
        {!loaded ? (
          <Empty title="Configs are not loaded yet" sub="Click Load configs to fetch all your client profiles." />
        ) : loading ? (
          <div className="stack">
            {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} height={40} />)}
          </div>
        ) : filtered.length === 0 ? (
          <Empty title="No configs found" sub="Create a client first from Clients section." />
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Client</th>
                  <th>Protocol</th>
                  <th>IP</th>
                  <th>Last handshake</th>
                  <th className="actions">Action</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((p) => (
                  <tr key={p.id}>
                    <td><strong>{p.name}</strong></td>
                    <td>{p.protocol}</td>
                    <td><code>{p.assigned_ip || '-'}</code></td>
                    <td className="text-dim">{formatRelative(p.last_handshake)}</td>
                    <td className="actions">
                      <Button size="sm" onClick={() => openConfig(p)}>Open config</Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {view && (
        <Modal
          open
          onClose={() => setView(null)}
          title={`${view.peer.name} - config`}
          footer={
            <>
              <Button onClick={() => copyToClipboard(view.config, 'Copied')}>Copy</Button>
              <Button variant="primary" onClick={() => downloadFile(`${view.peer.name}.conf`, view.config)}>Download</Button>
            </>
          }
        >
          <div className="stack">
            <div className="row" style={{ justifyContent: 'center' }}>
              <div className="qr-box">
                <QRCodeSVG value={view.config} size={220} level="M" />
              </div>
            </div>
            <pre>{view.config}</pre>
          </div>
        </Modal>
      )}
    </div>
  )
}
