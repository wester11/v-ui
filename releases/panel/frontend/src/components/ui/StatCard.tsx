import { ReactNode } from 'react'

type Tone = 'default' | 'success' | 'warn' | 'violet'

interface StatCardProps {
  label: string
  value: ReactNode
  meta?: ReactNode
  tone?: Tone
}

export function StatCard({ label, value, meta, tone = 'default' }: StatCardProps) {
  const cls = ['stat-card', tone === 'success' ? 'success' : '', tone === 'warn' ? 'warn' : '', tone === 'violet' ? 'violet' : '']
    .filter(Boolean)
    .join(' ')
  return (
    <div className={cls}>
      <div className="stat-label">{label}</div>
      <div className="stat-value">{value}</div>
      {meta && <div className="stat-meta">{meta}</div>}
    </div>
  )
}
