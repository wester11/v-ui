import { createContext, useContext, useMemo, useState } from 'react'
import type { ReactNode } from 'react'

export type Locale = 'en' | 'ru'

type Dict = Record<string, string>

const dictionaries: Record<Locale, Dict> = {
  en: {
    nav_dashboard: 'Dashboard',
    nav_servers: 'Servers',
    nav_clients: 'Clients',
    nav_configs: 'Configs',
    nav_logs: 'Logs / Audit',
    nav_settings: 'Settings',
    nav_users: 'Users',
    nav_profile: 'Profile',
    settings_language: 'Language',
    settings_language_hint: 'Choose UI language',
  },
  ru: {
    nav_dashboard: 'Дашборд',
    nav_servers: 'Серверы',
    nav_clients: 'Клиенты',
    nav_configs: 'Конфиги',
    nav_logs: 'Логи / Аудит',
    nav_settings: 'Настройки',
    nav_users: 'Пользователи',
    nav_profile: 'Профиль',
    settings_language: 'Язык',
    settings_language_hint: 'Выберите язык интерфейса',
  },
}

type I18nValue = {
  locale: Locale
  setLocale: (locale: Locale) => void
  t: (key: string, fallback?: string) => string
}

const I18nContext = createContext<I18nValue | null>(null)

const STORAGE_KEY = 'voidwg.locale'

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (raw === 'ru' || raw === 'en') return raw
    return 'en'
  })

  const value = useMemo<I18nValue>(() => ({
    locale,
    setLocale: (next) => {
      setLocaleState(next)
      window.localStorage.setItem(STORAGE_KEY, next)
    },
    t: (key, fallback) => dictionaries[locale][key] || fallback || key,
  }), [locale])

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n() {
  const ctx = useContext(I18nContext)
  if (!ctx) {
    throw new Error('useI18n must be used inside I18nProvider')
  }
  return ctx
}

