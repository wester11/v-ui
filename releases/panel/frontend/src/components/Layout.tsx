import { useEffect, useMemo, useRef, useState } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../store/auth'
import { useI18n } from '../i18n'
import type { ReactNode } from 'react'

/* ── Brand icon ─────────────────────────────────────────── */
function BrandIcon() {
  return (
    <svg width="28" height="28" viewBox="0 0 28 28" fill="none">
      <rect width="28" height="28" rx="8" fill="url(#bg)" />
      <g stroke="#fff" strokeWidth="2" strokeLinecap="round">
        <line x1="4"  y1="14" x2="4"  y2="14" />
        <line x1="8"  y1="10" x2="8"  y2="18" />
        <line x1="12" y1="7"  x2="12" y2="21" />
        <line x1="16" y1="11" x2="16" y2="17" />
        <line x1="20" y1="9"  x2="20" y2="19" />
        <line x1="24" y1="13" x2="24" y2="15" />
      </g>
      <defs>
        <linearGradient id="bg" x1="0" y1="0" x2="28" y2="28" gradientUnits="userSpaceOnUse">
          <stop stopColor="#7c3aed" />
          <stop offset="1" stopColor="#a78bfa" />
        </linearGradient>
      </defs>
    </svg>
  )
}

/* ── Nav icons ───────────────────────────────────────────── */
function IconDashboard() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="3" width="7" height="7" rx="1.5" /><rect x="14" y="3" width="7" height="7" rx="1.5" />
      <rect x="3" y="14" width="7" height="7" rx="1.5" /><rect x="14" y="14" width="7" height="7" rx="1.5" />
    </svg>
  )
}
function IconServers() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="2" y="3" width="20" height="5" rx="2" /><rect x="2" y="10" width="20" height="5" rx="2" />
      <rect x="2" y="17" width="20" height="5" rx="2" />
      <circle cx="6" cy="5.5" r="1" fill="currentColor" stroke="none" />
      <circle cx="6" cy="12.5" r="1" fill="currentColor" stroke="none" />
      <circle cx="6" cy="19.5" r="1" fill="currentColor" stroke="none" />
    </svg>
  )
}
function IconClients() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M23 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" />
    </svg>
  )
}
function IconConfigs() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 2L2 7l10 5 10-5-10-5z" /><path d="M2 17l10 5 10-5" /><path d="M2 12l10 5 10-5" />
    </svg>
  )
}
function IconLogs() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <polyline points="14 2 14 8 20 8" /><line x1="8" y1="13" x2="16" y2="13" /><line x1="8" y1="17" x2="13" y2="17" />
    </svg>
  )
}
function IconUsers() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" />
    </svg>
  )
}
function IconProfile() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" />
    </svg>
  )
}
function IconList() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <line x1="8" y1="6" x2="21" y2="6" /><line x1="8" y1="12" x2="21" y2="12" /><line x1="8" y1="18" x2="21" y2="18" />
      <line x1="3" y1="6" x2="3.01" y2="6" /><line x1="3" y1="12" x2="3.01" y2="12" /><line x1="3" y1="18" x2="3.01" y2="18" />
    </svg>
  )
}
function IconFleet() {
  return (
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
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

/* ── Types ───────────────────────────────────────────────── */
interface SubNavItem {
  to: string
  labelKey: string
  label: string
  icon?: ReactNode
}

interface NavItem {
  to?: string
  label: string
  labelKey: string
  icon: ReactNode
  section: 'overview' | 'operations' | 'security' | 'account'
  roles?: Array<'admin' | 'operator' | 'user'>
  end?: boolean
  expandable?: boolean
  subItems?: SubNavItem[]
  beta?: boolean
}

const NAV: NavItem[] = [
  { to: '/', label: 'Панель', labelKey: 'nav_dashboard', icon: <IconDashboard />, section: 'overview', end: true },
  { to: '/clients', label: 'Клиенты', labelKey: 'nav_clients', icon: <IconClients />, section: 'operations' },
  { to: '/users', label: 'Пользователи', labelKey: 'nav_users', icon: <IconUsers />, section: 'operations', roles: ['admin', 'operator'] },
  {
    label: 'Серверы', labelKey: 'nav_servers', icon: <IconServers />, section: 'operations',
    roles: ['admin', 'operator'],
    expandable: true,
    subItems: [
      { to: '/servers', label: 'Список нод', labelKey: 'nav_servers_list', icon: <IconList /> },
      { to: '/settings', label: 'Fleet ops', labelKey: 'nav_fleet', icon: <IconFleet /> },
    ],
  },
  { to: '/configs', label: 'Конфиги', labelKey: 'nav_configs', icon: <IconConfigs />, section: 'operations' },
  { to: '/logs', label: 'Логи', labelKey: 'nav_logs', icon: <IconLogs />, section: 'security', roles: ['admin'] },
  { to: '/profile', label: 'Профиль', labelKey: 'nav_profile', icon: <IconProfile />, section: 'account' },
]

const SECTION_LABELS: Record<NavItem['section'], string> = {
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

/* ── Expandable nav group ────────────────────────────────── */
function ExpandableNavItem({
  item, t, defaultOpen,
}: {
  item: NavItem
  t: (k: string, fb?: string) => string
  defaultOpen: boolean
}) {
  const [open, setOpen] = useState(defaultOpen)

  return (
    <div>
      <button
        className="nav-item nav-expandable"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
      >
        <span className="nav-icon">{item.icon}</span>
        <span className="nav-label">{t(item.labelKey, item.label)}</span>
        {item.beta && <span className="badge-beta">β</span>}
        <span className={`nav-chevron${open ? ' open' : ''}`}><IconChevron /></span>
      </button>
      {open && (
        <div className="nav-sub">
          {item.subItems?.map((sub) => (
            <NavLink
              key={sub.to}
              to={sub.to}
              className={({ isActive }) => `nav-sub-item${isActive ? ' active' : ''}`}
            >
              {sub.icon && <span className="nav-sub-icon">{sub.icon}</span>}
              <span>{t(sub.labelKey, sub.label)}</span>
            </NavLink>
          ))}
        </div>
      )}
    </div>
  )
}

/* ── Layout ──────────────────────────────────────────────── */
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

  const visible = useMemo(() =>
    NAV.filter((n) => !n.roles || n.roles.includes(role)),
  [role])

  const grouped = useMemo(() => {
    const buckets: Record<NavItem['section'], NavItem[]> = {
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

  const handleLogout = () => { logout(); nav('/login', { replace: true }) }

  // Auto-expand servers group if on /servers or /settings
  const serversAutoOpen = loc.pathname.startsWith('/servers') || loc.pathname === '/settings'

  return (
    <div className="app shell-bg">
      <aside className="sidebar">
        {/* Brand */}
        <div className="sidebar-brand">
          <BrandIcon />
          <div className="brand-text">
            <div className="brand-name">VOIDVPN</div>
            <div className="brand-sub">{t('layout_subtitle')}</div>
          </div>
        </div>

        {/* Nav */}
        <nav className="sidebar-nav">
          {(Object.keys(grouped) as Array<NavItem['section']>).map((section) =>
            grouped[section].length > 0 ? (
              <div key={section} className="sidebar-section">
                <div className="sidebar-section-title">
                  <span>{SECTION_LABELS[section]}</span>
                </div>
                {grouped[section].map((n) =>
                  n.expandable ? (
                    <ExpandableNavItem
                      key={n.labelKey}
                      item={n}
                      t={t}
                      defaultOpen={serversAutoOpen}
                    />
                  ) : (
                    <NavLink
                      key={n.to}
                      to={n.to!}
                      end={n.end}
                      className={({ isActive }) => `nav-item${isActive ? ' active' : ''}`}
                    >
                      <span className="nav-icon">{n.icon}</span>
                      <span className="nav-label">{t(n.labelKey, n.label)}</span>
                      {n.beta && <span className="badge-beta">β</span>}
                    </NavLink>
                  ),
                )}
              </div>
            ) : null,
          )}
        </nav>

        {/* Footer */}
        <div className="sidebar-footer">
          <div className="row gap-2">
            <span className="badge badge-violet text-xs">{role}</span>
            <span className="text-xs text-mute">v0.2.0</span>
          </div>
        </div>
      </aside>

      {/* Topbar */}
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
