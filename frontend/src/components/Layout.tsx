import { useEffect, useRef, useState } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../store/auth'

interface NavDef {
  to: string
  label: string
  icon: string
  roles?: Array<'admin' | 'operator' | 'user'>
}

const NAV: NavDef[] = [
  { to: '/',         label: 'Dashboard', icon: '◧' },
  { to: '/peers',    label: 'Peers',     icon: '◍' },
  { to: '/servers',  label: 'Servers',   icon: '◈', roles: ['admin', 'operator'] },
  { to: '/users',    label: 'Users',     icon: '◉', roles: ['admin', 'operator'] },
  { to: '/profile',  label: 'Profile',   icon: '◎' },
]

const TITLES: Record<string, string> = {
  '/':        'Dashboard',
  '/peers':   'Peers',
  '/servers': 'Servers',
  '/users':   'Users',
  '/profile': 'Profile',
}

export default function Layout() {
  const user = useAuth((s) => s.user)
  const logout = useAuth((s) => s.logout)
  const nav = useNavigate()
  const loc = useLocation()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    window.addEventListener('mousedown', onClick)
    return () => window.removeEventListener('mousedown', onClick)
  }, [])

  const role = user?.role ?? 'user'
  const visible = NAV.filter((n) => !n.roles || n.roles.includes(role))

  let title = TITLES[loc.pathname] ?? ''
  if (!title && loc.pathname.startsWith('/servers/')) title = 'Server'

  const initial = (user?.email ?? '?').slice(0, 1)

  const handleLogout = () => {
    logout()
    nav('/login', { replace: true })
  }

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="sidebar-brand">
          <span className="brand-dot" />
          void-wg
        </div>
        <nav className="sidebar-nav">
          {visible.map((n) => (
            <NavLink
              key={n.to}
              to={n.to}
              end={n.to === '/'}
              className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}
            >
              <span className="nav-icon">{n.icon}</span>
              {n.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">v0.1.0 · {role}</div>
      </aside>

      <header className="topbar">
        <div className="topbar-title">{title}</div>
        <div className="topbar-right">
          <div className="user-menu" ref={ref}>
            <button className="user-button" onClick={() => setOpen((v) => !v)}>
              <span className="user-avatar">{initial}</span>
              <span className="truncate" style={{ maxWidth: 160 }}>{user?.email ?? '…'}</span>
              <span className="text-mute">▾</span>
            </button>
            {open && (
              <div className="user-dropdown">
                <div className="dropdown-item" style={{ cursor: 'default' }}>
                  <div className="stack-sm">
                    <div className="text-sm">{user?.email}</div>
                    <div className="text-xs text-mute">role: {role}</div>
                  </div>
                </div>
                <div className="dropdown-divider" />
                <button className="dropdown-item" onClick={() => { setOpen(false); nav('/profile') }}>
                  ◎ Profile
                </button>
                <button className="dropdown-item" onClick={handleLogout}>
                  ⏻ Sign out
                </button>
              </div>
            )}
          </div>
        </div>
      </header>

      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}
