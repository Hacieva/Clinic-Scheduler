import { useNavigate } from 'react-router-dom'
import { format } from 'date-fns'
import { ru } from 'date-fns/locale'
import { UserPlus, CalendarCheck, Clock, TrendingUp, AlertCircle } from 'lucide-react'

export default function CashboxPage() {
  const navigate = useNavigate()
  const todayLabel = format(new Date(), 'EEEE, d MMMM', { locale: ru })

  return (
    <div className="p-6 lg:p-8 max-w-4xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Касса</h1>
        <p className="text-sm text-gray-500 mt-0.5 capitalize">{todayLabel}</p>
      </div>

      {/* v0.3 banner */}
      <div className="flex items-start gap-3 bg-amber-50 border border-amber-200 rounded-xl p-4 mb-6">
        <AlertCircle size={16} className="text-amber-500 shrink-0 mt-0.5" />
        <div>
          <p className="text-sm font-medium text-amber-800">Кассовый модуль — v0.3</p>
          <p className="text-xs text-amber-700 mt-0.5">
            Полноценная касса: оплата, чеки, смена-отчёт, выплаты врачам запустятся в v0.3.
            Сейчас доступен прием walk-in пациентов.
          </p>
        </div>
      </div>

      {/* Actions */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
        <button
          onClick={() => navigate('/admin/cashbox/walk-in')}
          className="flex items-start gap-4 bg-white border-2 border-blue-200 hover:border-blue-400 rounded-xl p-5 text-left transition-all group"
        >
          <div className="w-10 h-10 bg-blue-50 group-hover:bg-blue-100 rounded-lg flex items-center justify-center shrink-0 transition-colors">
            <UserPlus size={20} className="text-blue-600" />
          </div>
          <div>
            <p className="text-sm font-semibold text-gray-900">Новый walk-in визит</p>
            <p className="text-xs text-gray-500 mt-0.5">Пациент без предварительной записи</p>
          </div>
        </button>

        <div className="flex items-start gap-4 bg-white border border-dashed border-gray-200 rounded-xl p-5 opacity-60 cursor-not-allowed">
          <div className="w-10 h-10 bg-gray-50 rounded-lg flex items-center justify-center shrink-0">
            <CalendarCheck size={20} className="text-gray-400" />
          </div>
          <div>
            <p className="text-sm font-semibold text-gray-500">Запись → Визит</p>
            <p className="text-xs text-gray-400 mt-0.5">Открыть визит по существующей записи</p>
            <span className="inline-block mt-1.5 text-[10px] font-medium text-amber-600 bg-amber-50 px-1.5 py-0.5 rounded-full border border-amber-200">
              v0.3
            </span>
          </div>
        </div>
      </div>

      {/* Stats placeholder */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        {[
          { label: 'Визитов сегодня', Icon: Clock },
          { label: 'Чеков закрыто',   Icon: CalendarCheck },
          { label: 'Выручка (касса)', Icon: TrendingUp },
        ].map(({ label, Icon }) => (
          <div key={label} className="bg-white rounded-xl border border-dashed border-gray-200 p-4 flex items-center gap-3">
            <Icon size={18} className="text-gray-300 shrink-0" />
            <div>
              <p className="text-xs text-gray-400">{label}</p>
              <p className="text-lg font-bold text-gray-300 tabular-nums">—</p>
            </div>
          </div>
        ))}
      </div>

      {/* Visits list placeholder */}
      <div className="bg-white rounded-xl border border-dashed border-gray-200 p-10 flex flex-col items-center gap-2 text-center">
        <Clock size={36} strokeWidth={1.25} className="text-gray-200" />
        <p className="text-sm font-medium text-gray-400">Список визитов сегодня</p>
        <p className="text-xs text-gray-400">Появится после запуска кассового модуля (v0.3)</p>
      </div>
    </div>
  )
}
