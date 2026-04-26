import { create } from 'zustand'

type Kind = 'info' | 'success' | 'error' | 'warn'
interface ToastItem { id: number; text: string; kind: Kind }

interface ToastState {
  items: ToastItem[]
  push: (text: string, kind?: Kind) => void
  remove: (id: number) => void
}

export const useToasts = create<ToastState>((set, get) => ({
  items: [],
  push: (text, kind = 'info') => {
    const id = Date.now() + Math.random()
    set({ items: [...get().items, { id, text, kind }] })
    setTimeout(() => get().remove(id), 4000)
  },
  remove: (id) => set({ items: get().items.filter((t) => t.id !== id) }),
}))

export const toast = {
  info:    (t: string) => useToasts.getState().push(t, 'info'),
  success: (t: string) => useToasts.getState().push(t, 'success'),
  error:   (t: string) => useToasts.getState().push(t, 'error'),
  warn:    (t: string) => useToasts.getState().push(t, 'warn'),
}

export function ToastHost() {
  const items = useToasts((s) => s.items)
  const remove = useToasts((s) => s.remove)
  return (
    <div className="toast-host" aria-live="polite">
      {items.map((t) => (
        <div key={t.id} className={`toast ${t.kind}`} onClick={() => remove(t.id)}>
          {t.text}
        </div>
      ))}
    </div>
  )
}
