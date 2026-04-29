import { useEffect, useMemo, useRef, useState } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../store/auth'
import { useI18n } from '../i18n'
import type { ReactNode } from 'react'

function IconDashboard() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="3" width="7" height="7" rx="1.5" />
      <rect x="14" y="3" width="7" height="7" rx="1.5" />
      <rect x="3" y="14" width="7" height="7" rx="1.5" />
      <rect x="14" y="14" width="7" height="7" rx="1.5" />
    </svg>
  )
}
function IconServers() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="2" y="3" width="20" height="5" rx="2" />
      <rect x="2" y="10" width="20" height="5" rx="2" />
      <rect x="2" y="17" width="20" height="5" rx="2" />
      <circle cx="6" cy="5.5" r="1" fill="currentColor" stroke="none" />
      <circle cx="6" cy="12.5" r="1" fill="currentColor" stroke="none" />
      <circle cx="6" cy="19.5" r="1" fill="currentColor" stroke="none" />
    </svg>
  )
}
function IconClients() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
      <path d="M16 3.13a4 4 0 0 1 0 7.75" />
    </svg>
  )
}
function IconConfigs() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 2L2 7l10 5 10-5-10-5z" />
      <path d="M2 17l10 5 10-5" />
      <path d="M2 12l10 5 10-5" />
    </svg>
  )
}
function IconLogs() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <polyline points="14 2 14 8 20 8" />
      <line x1="8" y1="13" x2="16" y2="13" />
      <line x1="8" y1="17" x2="13" y2="17" />
    </svg>
  )
}
function IconSettings() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
    </svg>
  )
}
function IconUsers() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
      <circle cx="12" cy="7" r="4" />
    </svg>
  )
}
function IconProfile() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
      <circle cx="12" cy="7" r="4" />
    </svg>
  )
}
function IconChevron() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="6 9 12 15 18 9" />
    </svg>
  )
}

const NAV_ICONS: Record<string, ReactNode> = {
  '/':         <IconDashboard />,
  '/servers':  <IconServers />,
  '/clients':  <IconClients />,
  '/configs':  <IconConfigs />,
  '/logs':     <IconLogs />,
  '/settings': <IconSettings />,
  '/users':    <IconUsers />,
  '/profile':  <IconProfile />,
}

interface NavDef {
  to: string
  label: string
  i18nKey: string
  section: 'overview' | 'operations' | 'security' | 'account'
  roles?: Array<'admin' | 'operator' | 'user'>
}

const NAV: NavDef[] = [
  { to: '/',         label: 'Dashboard', i18nKey: 'nav_dashboard', section: 'overview' },
  { to: '/servers',  label: 'Servers',   i18nKey: 'nav_servers',   section: 'operations', roles: ['admin', 'operator'] },
  { to: '/clients',  label: 'Clients',   i18nKey: 'nav_clients',   section: 'operations' },
  { to: '/configs',  label: 'Configs',   i18nKey: 'nav_configs',   section: 'operations' },
  { to: '/logs',     label: 'Logs',      i18nKey: 'nav_logs',      section: 'security',   roles: ['admin'] },
  { to: '/settings', label: 'Settings',  i18nKey: 'nav_settings',  section: 'security',   roles: ['admin', 'operator'] },
  { to: '/users',    label: 'Users',     i18nKey: 'nav_users',     section: 'account',    roles: ['admin', 'operator'] },
  { to: '/profile',  label: 'Profile',   i18nKey: 'nav_profile',   section: 'account' },
]

const SECTION_LABELS: Record<NavDef['section'], string> = {
  overview:   'Обзор',
  operations: 'Управление',
  security:   'Безопасность',
  account:    'Аккаунт',
}

const TITLE_KEYS: Record<string, string> = {
  '/':         'nav_dashboard',
  '/clients':  'nav_clients',
  '/servers':  'nav_servers',
  '/users':    'nav_users',
  '/profile':  'nav_profile',
  '/configs':  'nav_configs',
  '/logs':     'nav_logs',
  '/settings': 'nav_settings',
}

export default function Layout() {
  const user   = useAuth((s) => s.user)
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
  const visible = useMemo(() => NAV.filter((n) => !n.roles || n.roles.includes(role)), [role])

  const grouped = useMemo(() => {
    const buckets: Record<NavDef['section'], NavDef[]> = {
      overview: [], operations: [], security: [], account: [],
    }
    for (const item of visible) buckets[item.section].push(item)
    return buckets
  }, [visible])

  const titleKey = TITLE_KEYS[loc.pathname]
  const title = titleKey
    ? t(titleKey)
    : loc.pathname.startsWith('/servers/') ? t('nav_servers') : ''

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
            <span className="text-xs text-mute">{t('layout_subtitle')}</span>
          </div>
        </div>

        <nav className="sidebar-nav">
          {(Object.keys(grouped) as Array<NavDef['section']>).map((section) =>
            grouped[section].length > 0 ? (
              <div key={section} className="sidebar-section">
                <div className="sidebar-section-title">{SECTION_LABELS[section]}</div>
                {grouped[section].map((n) => (
                  <NavLink
                    key={n.to}
                    to={n.to}
                    end={n.to === '/'}
                    className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}
                  >
                    <span className="nav-icon">{NAV_ICONS[n.to]}</span>
                    <span>{t(n.i18nKey, n.label)}</span>
                  </NavLink>
                ))}
              </div>
            ) : null,
          )}
        </nav>

        <div className="sidebar-footer">
          <span className="badge badge-violet">{role}</span>
          <span>v0.2.0</span>
        </div>
      </aside>

      <header className="topbar">
        <div>
          <div className="topbar-title">{title}</div>
          <div className="topbar-sub">{t('layout_subtitle')}</div>
        </div>
        <div className="topbar-right">
          <div className="user-menu" ref={ref}>
            <button className="user-button" onClick={() => setOpen((v) => !v)}>
              <span className="user-avatar">{initial}</span>
              <span className="truncate" style={{ maxWidth: 180 }}>{user?.email ?? '...'}</span>
              <span className="text-mute" style={{ display: 'flex' }}><IconChevron /></span>
            </button>
            {open && (
              <div className="user-dropdown">
                <div className="dropdown-item" style={{ cursor: 'default', flexDirection: 'column', alignItems: 'flex-start' }}>
                  <div className="text-sm" style={{ fontWeight: 500 }}>{user?.email}</div>
                  <div className="text-xs text-mute">{t('common_role')}: {role}</div>
                </div>
                <div className="dropdown-divider" />
                <button className="dropdown-item" onClick={() => { setOpen(false); nav('/profile') }}>
                  {t('nav_profile')}
                </button>
                <div className="dropdown-item" style={{ cursor: 'default', flexDirection: 'column', alignItems: 'flex-start', gap: 6 }}>
                  <div className="text-xs text-mute">{t('settings_language')}</div>
                  <select
                    className="select"
                    style={{ padding: '4px 8px', fontSize: 12 }}
                    value={locale}
                    onChange={(e) => setLocale(e.target.value as 'en' | 'ru')}
                  >
                    <option value="ru">Русский</option>
                    <option value="en">English</option>
                  </select>
                </div>
                <div className="dropdown-divider" />
                <button className="dropdown-item" onClick={handleLogout} style={{ color: 'var(--danger)' }}>
                  {t('layout_signout')}
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
