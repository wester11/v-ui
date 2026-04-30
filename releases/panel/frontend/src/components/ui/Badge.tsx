import { ReactNode } from 'react'

type Tone = 'default' | 'success' | 'warn' | 'danger' | 'info' | 'violet'

const cls: Record<Tone, string> = {
  default: 'badge',
  success: 'badge badge-success',
  warn:    'badge badge-warn',
  danger:  'badge badge-danger',
  info:    'badge badge-info',
  violet:  'badge badge-violet',
}

export function Badge({ tone = 'default', children }: { tone?: Tone; children: ReactNode }) {
  return <span className={cls[tone]}>{children}</span>
}

export function StatusDot({ online }: { online: boolean }) {
  return (
    <span className="row gap-2" style={{ display: 'inline-flex' }}>
      <span className={`dot ${online ? 'dot-success' : 'dot-mute'}`} />
      <span className="text-sm text-dim">{online ? 'online' : 'offline'}</span>
    </span>
  )
}
