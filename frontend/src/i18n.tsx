import { createContext, useContext, useMemo, useState } from 'react'
import type { ReactNode } from 'react'

export type Locale = 'en' | 'ru'

type Dict = Record<string, string>

// === EN dictionary ===
const EN: Dict = {
  // navigation
  nav_dashboard: 'Dashboard',
  nav_servers:   'Servers',
  nav_clients:   'Clients',
  nav_configs:   'Configs',
  nav_logs:      'Logs / Audit',
  nav_settings:  'Settings',
  nav_users:     'Users',
  nav_profile:   'Profile',

  // common actions
  action_refresh:   'Refresh',
  action_create:    'Create',
  action_delete:    'Delete',
  action_save:      'Save',
  action_cancel:    'Cancel',
  action_back:      'Back',
  action_signout:   'Sign out',
  action_check:     'Check connection',
  action_deploy:    'Deploy config',
  action_restart:   'Restart node',
  action_rotate:    'Rotate secret',

  // statuses
  status_online:    'Online',
  status_offline:   'Offline',
  status_pending:   'Pending',
  status_deploying: 'Deploying',
  status_error:     'Error',
  status_degraded:  'Degraded',
  status_drifted:   'Drifted',

  // dashboard
  dashboard_title:       'Control center',
  dashboard_sub:         'Operational overview of your VPN fleet and clients.',
  dashboard_users:       'Users',
  dashboard_clients:     'Clients',
  dashboard_servers:     'Servers online',
  dashboard_traffic:     'Traffic',
  dashboard_unavailable: 'Dashboard unavailable',
  dashboard_no_servers:  'No servers yet',
  dashboard_no_servers_sub: 'Register your first VPN node to start provisioning clients.',
  dashboard_register:    'Register server',
  dashboard_all_healthy: 'all healthy',
  dashboard_degraded:    '{n} degraded/offline',
  dashboard_quick_title: 'Quick actions',
  dashboard_quick_sub:   'Most used controls.',
  dashboard_quick_clients: 'My clients',
  dashboard_quick_configs: 'Open configs',
  dashboard_quick_profile: 'Security profile',

  // servers list page
  servers_title:         'Servers',
  servers_sub:           'Add infrastructure nodes first. VPN configs are created separately in Configs.',
  servers_search:        'Search servers...',
  servers_add:           '+ Add server',
  servers_no_match:      'No servers match your search',
  servers_no_match_sub:  'Try a different query.',
  servers_empty:         'No servers yet',
  servers_empty_sub:     'Create your first node to start onboarding.',
  servers_open:          'Open',
  servers_remove:        'Remove',
  servers_last_beat:     'Last heartbeat',
  servers_create_title:  'Add infrastructure server',
  servers_field_name:    'Name',
  servers_field_name_ph: 'eu-1',
  servers_field_host:    'IP or hostname',
  servers_field_host_ph: '2.27.32.141 or eu-1.example.com',
  servers_onboard_title: 'Node onboarding',
  servers_onboard_step1: 'Step 1: copy command',
  servers_onboard_step2: 'Step 2: run on target VPS',
  servers_onboard_step3: 'Step 3: click Check connection',
  servers_onboard_wait:  'Waiting for first connection...',
  servers_onboard_hint:  'After node is online, go to Configs and create VPN logic.',
  servers_remove_title:  'Remove server?',
  servers_remove_body:   'This will remove server and associated configs.',

  // servers dashboard widget
  servers_status_card: 'Server status',
  servers_status_sub:  'Heartbeat, mode and protocol status for each node.',
  servers_col_name:    'Name',
  servers_col_protocol:'Protocol',
  servers_col_mode:    'Mode',
  servers_col_endpoint:'Endpoint',
  servers_col_status:  'Status',
  servers_col_lastbeat:'Last heartbeat',
  servers_col_node:    'Node ID',
  servers_manage:      'Manage servers',
  servers_fleet_ops:   'Fleet ops',

  // server detail
  server_detail_back:        'Back to servers',
  server_detail_back_btn:    'Back to servers',
  server_detail_node:        'Node status',
  server_detail_endpoint:    'Endpoint',
  server_detail_protocol:    'Active protocol',
  server_detail_version:     'Agent version',
  server_detail_identity:    'Node identity',
  server_detail_identity_sub:'Use Node ID for troubleshooting and onboarding.',
  server_detail_id_node:     'Node ID',
  server_detail_id_srv:      'Server ID',
  server_detail_configs:     'Configs on this server',
  server_detail_configs_sub: 'Separated VPN logic attached to this infrastructure node.',
  server_detail_status:      'Status',
  server_detail_hostname:    'hostname not reported',
  server_detail_check:       'Check connection',
  server_detail_deploy:      'Deploy config',
  server_detail_restart:     'Restart node',
  server_detail_rotate:      'Rotate secret',
  server_detail_refresh:     'Refresh',
  server_detail_no_cfg:      'No configs yet',
  server_detail_no_cfg_sub:  'Create config from Configs page.',
  server_detail_cfg_manage:  'Create or manage configs',
  server_detail_notfound:    'Cannot load server',
  server_detail_cfg_name:    'Name',
  server_detail_cfg_tpl:     'Template',
  server_detail_cfg_mode:    'Mode',
  server_detail_cfg_routing: 'Routing',
  server_detail_cfg_status:  'Status',
  server_detail_cfg_active:  'active',
  server_detail_cfg_inactive:'inactive',
  server_detail_beat:        'Last heartbeat',

  // confirms / toasts
  confirm_restart:   'Restart agent service on this node?',
  confirm_rotate:    'Rotate node secret? Old agent will stop authenticating until you re-run install-node.sh.',
  toast_deployed:    'Config deployed',
  toast_deploy_fail: 'Deploy failed',
  toast_restart_ok:  'Restart requested',
  toast_secret_ok:   'Secret rotated. Copied to clipboard - paste into node compose env.',
  toast_check_ok:    'Connection check completed',

  // common
  common_loading:    'Loading...',
  common_role:       'role',
  common_user_role:  'role',

  // layout
  layout_subtitle:   'VPN orchestration',
  layout_signout:    'Sign out',

  settings_language:      'Language',
  settings_language_hint: 'Choose UI language',

  // configs page
  configs_title:             'Configs',
  configs_sub:               'Create VPN logic separately from infrastructure servers.',
  configs_create:            '+ Create config',
  configs_no_servers:        'No servers found',
  configs_no_servers_sub:    'Create a server first in the Servers page.',
  configs_no_configs:        'No configs on this server',
  configs_no_configs_sub:    'Create your first Xray config to enable clients.',
  configs_col_name:          'Name',
  configs_col_tpl:           'Template',
  configs_col_mode:          'Mode',
  configs_col_routing:       'Routing',
  configs_col_status:        'Status',
  configs_col_updated:       'Updated',
  configs_col_actions:       'Actions',
  configs_status_active:     'active',
  configs_status_inactive:   'inactive',
  configs_activate:          'Activate',
  configs_protocol_label:    'Active protocol',
  configs_modal_title:       'Create Xray config',
  configs_field_name:        'Name',
  configs_field_name_ph:     'RU gateway profile',
  configs_field_server:      'Server',
  configs_field_template:    'Template',
  configs_tpl_vless:         'VLESS Reality (default)',
  configs_tpl_grpc:          'gRPC Reality',
  configs_tpl_cascade:       'Cascade config',
  configs_tpl_empty:         'Empty config',
  configs_field_mode:        'Mode',
  configs_mode_simple:       'Simple',
  configs_mode_advanced:     'Advanced (raw JSON)',
  configs_field_routing:     'Routing mode',
  configs_routing_simple:    'Simple',
  configs_routing_advanced:  'Advanced',
  configs_routing_cascade:   'Cascade',
  configs_section_inbound:   'Inbound settings',
  configs_section_routing:   'Routing settings',
  configs_section_cascade:   'Cascade settings',
  configs_field_port:        'Port',
  configs_field_shortids:    'Short IDs count',
  configs_field_sni:         'SNI host',
  configs_field_dest:        'Destination',
  configs_field_fingerprint: 'TLS fingerprint',
  configs_field_flow:        'Flow',
  configs_field_raw:         'Raw Xray JSON',
  configs_cascade_upstream:  'Upstream server',
  configs_cascade_upstream_ph: 'Select upstream...',
  configs_cascade_strategy:  'Strategy',
  configs_rule_ru_direct:    'geoip:ru - direct (RU traffic bypasses VPN)',
  configs_rule_non_ru_proxy: 'geoip:!ru - proxy (non-RU through VPN)',
  configs_activate_now:      'Activate immediately (auto-deploy)',
  configs_save:              'Save config',
  toast_config_saved:        'Config saved',
  toast_config_activated:    'Config activated and deployed',
  toast_activate_fail:       'Failed to activate config',
}

// === RU dictionary ===
const RU: Dict = {
  // navigation
  nav_dashboard: 'Панель',
  nav_servers:   'Серверы',
  nav_clients:   'Клиенты',
  nav_configs:   'Конфиги',
  nav_logs:      'Логи / аудит',
  nav_settings:  'Настройки',
  nav_users:     'Пользователи',
  nav_profile:   'Профиль',

  // common actions
  action_refresh:   'Обновить',
  action_create:    'Создать',
  action_delete:    'Удалить',
  action_save:      'Сохранить',
  action_cancel:    'Отмена',
  action_back:      'Назад',
  action_signout:   'Выйти',
  action_check:     'Проверить связь',
  action_deploy:    'Применить конфиг',
  action_restart:   'Перезапустить',
  action_rotate:    'Обновить ключ',

  // statuses
  status_online:    'Онлайн',
  status_offline:   'Оффлайн',
  status_pending:   'Ожидание',
  status_deploying: 'Деплой',
  status_error:     'Ошибка',
  status_degraded:  'Деградация',
  status_drifted:   'Дрейф',

  // dashboard
  dashboard_title:       'Командный центр',
  dashboard_sub:         'Обзор VPN-флота и клиентов.',
  dashboard_users:       'Пользователи',
  dashboard_clients:     'Клиенты',
  dashboard_servers:     'Серверы онлайн',
  dashboard_traffic:     'Трафик',
  dashboard_unavailable: 'Дашборд недоступен',
  dashboard_no_servers:  'Серверов пока нет',
  dashboard_no_servers_sub: 'Подключите первую VPN-ноду, чтобы выдавать клиентов.',
  dashboard_register:    'Добавить сервер',
  dashboard_all_healthy: 'всё здоровое',
  dashboard_degraded:    '{n} оффлайн / деградация',
  dashboard_quick_title: 'Быстрые действия',
  dashboard_quick_sub:   'Основные функции.',
  dashboard_quick_clients: 'Мои клиенты',
  dashboard_quick_configs: 'Открыть конфиги',
  dashboard_quick_profile: 'Профиль безопасности',

  // servers list page
  servers_title:         'Серверы',
  servers_sub:           'Сначала добавьте ноды. VPN-конфиги создаются отдельно в разделе Конфиги.',
  servers_search:        'Поиск серверов...',
  servers_add:           '+ Добавить сервер',
  servers_no_match:      'Ни один сервер не найден',
  servers_no_match_sub:  'Попробуйте другой запрос.',
  servers_empty:         'Серверов пока нет',
  servers_empty_sub:     'Создайте первую ноду, чтобы начать онбординг.',
  servers_open:          'Открыть',
  servers_remove:        'Удалить',
  servers_last_beat:     'Последний heartbeat',
  servers_create_title:  'Добавить сервер',
  servers_field_name:    'Название',
  servers_field_name_ph: 'eu-1',
  servers_field_host:    'IP или hostname',
  servers_field_host_ph: '2.27.32.141 или eu-1.example.com',
  servers_onboard_title: 'Подключение ноды',
  servers_onboard_step1: 'Шаг 1: скопируйте команду',
  servers_onboard_step2: 'Шаг 2: выполните на целевом VPS',
  servers_onboard_step3: 'Шаг 3: нажмите \xabПроверить связь\xbb',
  servers_onboard_wait:  'Ожидаем первое подключение...',
  servers_onboard_hint:  'После того как нода онлайн — создайте конфиг в разделе Конфиги.',
  servers_remove_title:  'Удалить сервер?',
  servers_remove_body:   'Сервер и связанные конфиги будут удалены.',

  // servers dashboard widget
  servers_status_card: 'Состояние серверов',
  servers_status_sub:  'Heartbeat, режим и протокол по каждой ноде.',
  servers_col_name:    'Название',
  servers_col_protocol:'Протокол',
  servers_col_mode:    'Режим',
  servers_col_endpoint:'Endpoint',
  servers_col_status:  'Статус',
  servers_col_lastbeat:'Последний heartbeat',
  servers_col_node:    'Node ID',
  servers_manage:      'Управление серверами',
  servers_fleet_ops:   'Fleet ops',

  // server detail
  server_detail_back:        'Назад к серверам',
  server_detail_back_btn:    'Назад к серверам',
  server_detail_node:        'Статус ноды',
  server_detail_endpoint:    'Endpoint',
  server_detail_protocol:    'Активный протокол',
  server_detail_version:     'Версия агента',
  server_detail_identity:    'Идентификация ноды',
  server_detail_identity_sub:'Node ID нужен для онбординга и поддержки.',
  server_detail_id_node:     'Node ID',
  server_detail_id_srv:      'Server ID',
  server_detail_configs:     'Конфиги на этом сервере',
  server_detail_configs_sub: 'VPN-логика, прикреплённая к этой ноде.',
  server_detail_status:      'Статус',
  server_detail_hostname:    'hostname не определён',
  server_detail_check:       'Проверить связь',
  server_detail_deploy:      'Применить конфиг',
  server_detail_restart:     'Перезапустить',
  server_detail_rotate:      'Обновить ключ',
  server_detail_refresh:     'Обновить',
  server_detail_no_cfg:      'Конфигов нет',
  server_detail_no_cfg_sub:  'Создайте конфиг в разделе Конфиги.',
  server_detail_cfg_manage:  'Создать или управлять конфигами',
  server_detail_notfound:    'Не удалось загрузить сервер',
  server_detail_cfg_name:    'Название',
  server_detail_cfg_tpl:     'Шаблон',
  server_detail_cfg_mode:    'Режим',
  server_detail_cfg_routing: 'Маршрутизация',
  server_detail_cfg_status:  'Статус',
  server_detail_cfg_active:  'активен',
  server_detail_cfg_inactive:'неактивен',
  server_detail_beat:        'Последний heartbeat',

  // confirms / toasts
  confirm_restart:   'Перезапустить агент на этой ноде?',
  confirm_rotate:    'Обновить секрет ноды? Старый агент перестанет авторизовываться — нужен повторный install-node.sh.',
  toast_deployed:    'Конфиг применён',
  toast_deploy_fail: 'Не удалось применить конфиг',
  toast_restart_ok:  'Запрос на перезапуск отправлен',
  toast_secret_ok:   'Секрет обновлён. Новое значение скопировано — вставьте в compose-env ноды.',
  toast_check_ok:    'Проверка связи завершена',

  // common
  common_loading:    'Загрузка...',
  common_role:       'роль',
  common_user_role:  'роль',

  // layout
  layout_subtitle:   'VPN-оркестрация',
  layout_signout:    'Выйти',

  settings_language:      'Язык',
  settings_language_hint: 'Выберите язык интерфейса',

  // configs page
  configs_title:             'Конфиги',
  configs_sub:               'Создавайте VPN-логику отдельно от инфраструктурных серверов.',
  configs_create:            '+ Создать конфиг',
  configs_no_servers:        'Серверов не найдено',
  configs_no_servers_sub:    'Сначала создайте сервер в разделе Серверы.',
  configs_no_configs:        'Конфигов на этом сервере нет',
  configs_no_configs_sub:    'Создайте первый Xray-конфиг, чтобы подключать клиентов.',
  configs_col_name:          'Название',
  configs_col_tpl:           'Шаблон',
  configs_col_mode:          'Режим',
  configs_col_routing:       'Маршрутизация',
  configs_col_status:        'Статус',
  configs_col_updated:       'Обновлён',
  configs_col_actions:       'Действия',
  configs_status_active:     'активен',
  configs_status_inactive:   'неактивен',
  configs_activate:          'Активировать',
  configs_protocol_label:    'Активный протокол',
  configs_modal_title:       'Создать Xray-конфиг',
  configs_field_name:        'Название',
  configs_field_name_ph:     'RU-шлюз',
  configs_field_server:      'Сервер',
  configs_field_template:    'Шаблон',
  configs_tpl_vless:         'VLESS Reality (по умолчанию)',
  configs_tpl_grpc:          'gRPC Reality',
  configs_tpl_cascade:       'Каскадный конфиг',
  configs_tpl_empty:         'Пустой конфиг',
  configs_field_mode:        'Режим настройки',
  configs_mode_simple:       'Простой',
  configs_mode_advanced:     'Расширенный (raw JSON)',
  configs_field_routing:     'Режим маршрутизации',
  configs_routing_simple:    'Простой',
  configs_routing_advanced:  'Расширенный',
  configs_routing_cascade:   'Каскадный',
  configs_section_inbound:   'Настройки входящего трафика',
  configs_section_routing:   'Настройки маршрутизации',
  configs_section_cascade:   'Параметры каскада',
  configs_field_port:        'Порт',
  configs_field_shortids:    'Количество Short IDs',
  configs_field_sni:         'SNI-хост',
  configs_field_dest:        'Назначение (dest)',
  configs_field_fingerprint: 'TLS fingerprint',
  configs_field_flow:        'Flow',
  configs_field_raw:         'Xray JSON (вручную)',
  configs_cascade_upstream:  'Вышестоящий сервер',
  configs_cascade_upstream_ph: 'Выберите upstream...',
  configs_cascade_strategy:  'Стратегия',
  configs_rule_ru_direct:    'geoip:ru — напрямую (RU трафик без VPN)',
  configs_rule_non_ru_proxy: 'geoip:!ru — через VPN (остальной трафик)',
  configs_activate_now:      'Активировать сразу (auto-deploy)',
  configs_save:              'Сохранить конфиг',
  toast_config_saved:        'Конфиг сохранён',
  toast_config_activated:    'Конфиг активирован и применён',
  toast_activate_fail:       'Не удалось активировать конфиг',
}

const dictionaries: Record<Locale, Dict> = { en: EN, ru: RU }

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
    return 'ru'
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
  if (!ctx) throw new Error('useI18n must be used inside I18nProvider')
  return ctx
}
