import { useState } from 'react'
import {
  BarChart3, Users, FlaskConical, CreditCard, TrendingUp, CalendarX2,
  ShieldCheck, ExternalLink, DollarSign,
} from 'lucide-react'

const CHIPS = [
  { key: 'by_doctor',     label: 'По врачам',                  icon: Users,        color: 'blue' },
  { key: 'by_service',    label: 'По услугам',                  icon: BarChart3,    color: 'emerald' },
  { key: 'lab',           label: 'По лаборатории',              icon: FlaskConical, color: 'cyan' },
  { key: 'referrers',     label: 'По внешним направителям',     icon: ExternalLink, color: 'violet' },
  { key: 'payouts',       label: 'Кому сколько выплатить',      icon: DollarSign,   color: 'amber' },
  { key: 'by_admin',      label: 'По администраторам',          icon: ShieldCheck,  color: 'gray' },
  { key: 'cashbox',       label: 'Касса',                       icon: CreditCard,   color: 'emerald' },
  { key: 'avg_check',     label: 'Средний чек',                 icon: TrendingUp,   color: 'amber' },
  { key: 'visits',        label: 'Записи / отмены / неявки',    icon: CalendarX2,   color: 'red' },
]

const CHIP_STYLE = {
  blue:    'border-blue-200    bg-blue-50    text-blue-700    hover:bg-blue-100',
  violet:  'border-violet-200  bg-violet-50  text-violet-700  hover:bg-violet-100',
  emerald: 'border-emerald-200 bg-emerald-50 text-emerald-700 hover:bg-emerald-100',
  amber:   'border-amber-200   bg-amber-50   text-amber-700   hover:bg-amber-100',
  gray:    'border-gray-200    bg-gray-50    text-gray-700    hover:bg-gray-100',
  cyan:    'border-cyan-200    bg-cyan-50    text-cyan-700    hover:bg-cyan-100',
  red:     'border-rose-200    bg-rose-50    text-rose-700    hover:bg-rose-100',
}

export default function ReportsPage() {
  const [selected, setSelected] = useState(null)
  const chip = CHIPS.find((c) => c.key === selected)

  return (
    <div className="p-6 lg:p-8 max-w-5xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Отчёты</h1>
        <p className="text-sm text-gray-500 mt-0.5">Аналитика и статистика по клинике</p>
      </div>

      <div className="flex flex-wrap gap-2 mb-8">
        {CHIPS.map(({ key, label, icon: Icon, color }) => (
          <button
            key={key}
            onClick={() => setSelected(selected === key ? null : key)}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg border text-sm font-medium transition-all ${CHIP_STYLE[color]} ${
              selected === key ? 'ring-2 ring-offset-1 ring-current shadow-sm' : 'opacity-80'
            }`}
          >
            <Icon size={13} />
            {label}
          </button>
        ))}
      </div>

      <div className="bg-white rounded-xl border border-dashed border-gray-300 p-12 flex flex-col items-center justify-center text-center gap-4 min-h-80">
        <BarChart3 size={44} strokeWidth={1.25} className="text-gray-300" />
        <div>
          <p className="text-sm font-semibold text-gray-500">
            {chip ? `Отчёт «${chip.label}»` : 'Выберите тип отчёта'}
          </p>
          <p className="text-xs text-gray-400 mt-1 max-w-xs mx-auto">
            Детальная аналитика появится после запуска кассового и статистического модулей (v0.3)
          </p>
        </div>
        {chip && (
          <div className="px-4 py-2 bg-gray-50 rounded-lg border border-gray-200 text-xs text-gray-500">
            Endpoint: <span className="font-mono text-blue-600">/api/v1/reports/{chip.key}</span>
          </div>
        )}
      </div>
    </div>
  )
}
