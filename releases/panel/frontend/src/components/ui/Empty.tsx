import { ReactNode } from 'react'

interface EmptyProps {
  title: string
  sub?: string
  icon?: ReactNode
  action?: ReactNode
}

export function Empty({ title, sub, icon = '∅', action }: EmptyProps) {
  return (
    <div className="empty">
      <div className="empty-icon">{icon}</div>
      <div className="empty-title">{title}</div>
      {sub && <div className="empty-sub">{sub}</div>}
      {action && <div style={{ marginTop: 16 }}>{action}</div>}
    </div>
  )
}

export function Skeleton({ height = 16, width }: { height?: number; width?: number | string }) {
  return <div className="skel" style={{ height, width: width ?? '100%' }} />
}

export function SkeletonRows({ rows = 4 }: { rows?: number }) {
  return (
    <div className="stack">
      {Array.from({ length: rows }).map((_, i) => (
        <Skeleton key={i} height={20} />
      ))}
    </div>
  )
}
