import { Link } from 'react-router-dom'
import {
  Stethoscope, Building2, UserCog, BookOpen, Plug2, FlaskConical, FileText,
} from 'lucide-react'

const GROUPS = [
  {
    title: 'Организация',
    desc: 'Базовая информация о клинике и структуре',
    items: [
      {
        to: '/admin/settings/clinic',
        icon: Stethoscope,
        label: 'Клиника',
        desc: 'Название, контакты, логотип',
      },
      {
        to: '/admin/settings/branches',
        icon: Building2,
        label: 'Филиалы',
        desc: 'Адреса и кабинеты',
      },
      {
        to: '/admin/settings/directions',
        icon: BookOpen,
        label: 'Направления',
        desc: 'Специализации врачей',
      },
    ],
  },
  {
    title: 'Доступ',
    desc: 'Пользователи и права доступа',
    items: [
      {
        to: '/admin/settings/users',
        icon: UserCog,
        label: 'Пользователи и роли',
        desc: 'Администраторы, врачи',
      },
    ],
  },
  {
    title: 'Интеграции',
    desc: 'Внешние сервисы и каналы',
    items: [
      {
        to: '/admin/settings/integrations',
        icon: Plug2,
        label: 'Telegram bot',
        desc: 'Онлайн-запись через бот',
      },
      {
        to: '/admin/settings/lab',
        icon: FlaskConical,
        label: 'Лаборатория',
        desc: 'Интеграция с лабораторией',
      },
    ],
  },
  {
    title: 'Шаблоны',
    desc: 'Документы и уведомления',
    items: [
      {
        to: '#',
        icon: FileText,
        label: 'Документы',
        desc: 'Шаблоны договоров (v0.4)',
        disabled: true,
      },
    ],
  },
]

export default function SettingsDashboard() {
  return (
    <div className="p-6 lg:p-8 max-w-4xl mx-auto">
      <div className="mb-8">
        <h1 className="text-2xl font-semibold text-gray-900">Настройки</h1>
        <p className="text-sm text-gray-500 mt-0.5">Конфигурация клиники и системы</p>
      </div>

      <div className="space-y-8">
        {GROUPS.map(({ title, desc, items }) => (
          <section key={title}>
            <div className="mb-3">
              <h2 className="text-sm font-semibold text-gray-700">{title}</h2>
              <p className="text-xs text-gray-400 mt-0.5">{desc}</p>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
              {items.map(({ to, icon: Icon, label, desc: itemDesc, disabled }) => {
                const cls = `flex items-start gap-3 p-4 bg-white rounded-xl border border-gray-200 transition-all ${
                  disabled
                    ? 'opacity-50 cursor-not-allowed'
                    : 'hover:border-blue-300 hover:shadow-sm'
                }`
                const inner = (
                  <>
                    <div className="w-9 h-9 flex items-center justify-center rounded-lg bg-blue-50 text-blue-600 shrink-0">
                      <Icon size={16} />
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-900">{label}</p>
                      <p className="text-xs text-gray-400 mt-0.5">{itemDesc}</p>
                    </div>
                  </>
                )
                return disabled ? (
                  <div key={label} className={cls}>
                    {inner}
                  </div>
                ) : (
                  <Link key={label} to={to} className={cls}>
                    {inner}
                  </Link>
                )
              })}
            </div>
          </section>
        ))}
      </div>
    </div>
  )
}
