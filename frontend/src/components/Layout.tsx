import { useEffect, useMemo, useRef, useState } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../store/auth'
import { useI18n } from '../i18n'

interface NavDef {
  to: string
  label: string
  i18nKey: string
  icon: string
  section: 'overview' | 'operations' | 'security' | 'account'
  roles?: Array<'admin' | 'operator' | 'user'>
}

const NAV: NavDef[] = [
  { to: '/', label: 'Dashboard', i18nKey: 'nav_dashboard', icon: 'DB', section: 'overview' },
  { to: '/servers', label: 'Servers', i18nKey: 'nav_servers', icon: 'SV', section: 'operations', roles: ['admin', 'operator'] },
  { to: '/clients', label: 'Clients', i18nKey: 'nav_clients', icon: 'CL', section: 'operations' },
  { to: '/configs', label: 'Configs', i18nKey: 'nav_configs', icon: 'CF', section: 'operations' },
  { to: '/logs', label: 'Logs / Audit', i18nKey: 'nav_logs', icon: 'LG', section: 'security', roles: ['admin'] },
  { to: '/settings', label: 'Settings', i18nKey: 'nav_settings', icon: 'ST', section: 'security', roles: ['admin', 'operator'] },
  { to: '/users', label: 'Users', i18nKey: 'nav_users', icon: 'US', section: 'account', roles: ['admin', 'operator'] },
  { to: '/profile', label: 'Profile', i18nKey: 'nav_profile', icon: 'ME', section: 'account' },
]

const SECTION_TITLES: Record<NavDef['section'], string> = {
  overview: 'Overview',
  operations: 'Operations',
  security: 'Security',
  account: 'Account',
}

const TITLES: Record<string, string> = {
  '/': 'Fleet Dashboard',
  '/clients': 'Clients',
  '/servers': 'Servers',
  '/users': 'Users',
  '/profile': 'Profile',
  '/configs': 'Configs / Access',
  '/logs': 'Logs / Audit',
  '/settings': 'Settings',
}

export default function Layout() {
  const user = useAuth((s) => s.user)
  const logout = useAuth((s) => s.logout)
  const { t, locale, setLocale } = useI18n()
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
  const visible = useMemo(
    () => NAV.filter((n) => !n.roles || n.roles.includes(role)),
    [role],
  )

  const grouped = useMemo(() => {
    const buckets: Record<NavDef['section'], NavDef[]> = {
      overview: [],
      operations: [],
      security: [],
      account: [],
    }
    for (const item of visible) buckets[item.section].push(item)
    return buckets
  }, [visible])

  let title = TITLES[loc.pathname] ?? ''
  if (!title && loc.pathname.startsWith('/servers/')) title = 'Server Detail'

  const initial = (user?.email ?? '?').slice(0, 1).toUpperCase()

  const handleLogout = () => {
    logout()
    nav('/login', { replace: true })
  }

  return (
    <div className="app shell-bg">
      <aside className="sidebar">
        <div className="sidebar-brand">
          <span className="brand-dot" />
          <div className="stack-sm" style={{ gap: 2 }}>
            <strong>void-wg</strong>
            <span className="text-xs text-mute">Control plane</span>
          </div>
        </div>

        <nav className="sidebar-nav">
          {(Object.keys(grouped) as Array<NavDef['section']>).map((section) => (
            grouped[section].length > 0 ? (
              <div key={section} className="sidebar-section">
                <div className="sidebar-section-title">{SECTION_TITLES[section]}</div>
                {grouped[section].map((n) => (
                  <NavLink
                    key={n.to}
                    to={n.to}
                    end={n.to === '/'}
                    className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}
                  >
                    <span className="nav-icon">{n.icon}</span>
                    <span>{t(n.i18nKey, n.label)}</span>
                  </NavLink>
                ))}
              </div>
            ) : null
          ))}
        </nav>

        <div className="sidebar-footer">
          <span className="badge badge-info">{role}</span>
          <span>v0.2.0</span>
        </div>
      </aside>

      <header className="topbar">
        <div>
          <div className="topbar-title">{title}</div>
          <div className="topbar-sub">Production-oriented VPN orchestration</div>
        </div>
        <div className="topbar-right">
          <div className="user-menu" ref={ref}>
            <button className="user-button" onClick={() => setOpen((v) => !v)}>
              <span className="user-avatar">{initial}</span>
              <span className="truncate" style={{ maxWidth: 200 }}>{user?.email ?? '...'}</span>
              <span className="text-mute">v</span>
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
                  {t('nav_profile', 'Profile')}
                </button>
                <div className="dropdown-item" style={{ cursor: 'default' }}>
                  <div className="stack-sm">
                    <div className="text-xs text-mute">{t('settings_language', 'Language')}</div>
                    <select
                      className="select"
                      value={locale}
                      onChange={(e) => setLocale(e.target.value as 'en' | 'ru')}
                    >
                      <option value="en">English</option>
                      <option value="ru">Русский</option>
                    </select>
                  </div>
                </div>
                <button className="dropdown-item" onClick={handleLogout}>
                  Sign out
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
