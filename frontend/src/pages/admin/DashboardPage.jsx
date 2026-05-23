import { useQuery } from '@tanstack/react-query'
import { format } from 'date-fns'
import { ru } from 'date-fns/locale'
import {
  CalendarCheck, Users, CheckCircle2, XCircle,
  BarChart3, UserRound, TrendingUp, Clock,
} from 'lucide-react'
import { getAppointments } from '../../api/appointments'
import { getDoctors } from '../../api/doctors'
import useBranchStore from '../../stores/branch'

const today = format(new Date(), 'yyyy-MM-dd')

const AVATAR_COLORS = [
  'bg-blue-500', 'bg-emerald-500', 'bg-violet-500',
  'bg-amber-500', 'bg-rose-500', 'bg-cyan-500',
]
function avatarBg(id) { return AVATAR_COLORS[id % AVATAR_COLORS.length] }

// ─── StatCard ─────────────────────────────────────────────────────────────────

const COLOR_MAP = {
  blue:   'bg-blue-50 text-blue-600',
  green:  'bg-emerald-50 text-emerald-600',
  amber:  'bg-amber-50 text-amber-600',
  red:    'bg-rose-50 text-rose-600',
  violet: 'bg-violet-50 text-violet-600',
  gray:   'bg-gray-100 text-gray-500',
}

function StatCard({ icon: Icon, label, value, sub, color = 'blue', loading = false }) {
  return (
    <div className="bg-white rounded-xl border border-gray-200 p-5 flex items-start gap-4">
      <div className={`flex items-center justify-center w-10 h-10 rounded-lg ${COLOR_MAP[color]} shrink-0`}>
        <Icon size={18} />
      </div>
      <div className="min-w-0">
        <p className="text-xs text-gray-500 font-medium">{label}</p>
        {loading ? (
          <div className="h-7 w-12 bg-gray-200 rounded animate-pulse mt-0.5" />
        ) : (
          <p className="text-2xl font-bold text-gray-900 leading-tight mt-0.5">{value}</p>
        )}
        {sub && !loading && <p className="text-xs text-gray-400 mt-0.5">{sub}</p>}
      </div>
    </div>
  )
}

// ─── StatusRow ────────────────────────────────────────────────────────────────

const STATUS_COLORS = {
  blue:  'bg-blue-50 text-blue-700 border-blue-200',
  green: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  amber: 'bg-amber-50 text-amber-700 border-amber-200',
  red:   'bg-rose-50 text-rose-700 border-rose-200',
  gray:  'bg-gray-50 text-gray-600 border-gray-200',
}

function StatusRow({ count, label, color }) {
  if (!count) return null
  return (
    <div className={`flex items-center justify-between px-3 py-2.5 rounded-lg border ${STATUS_COLORS[color]}`}>
      <span className="text-sm font-medium">{label}</span>
      <span className="text-sm font-bold tabular-nums">{count}</span>
    </div>
  )
}

// ─── DoctorLoadRow ────────────────────────────────────────────────────────────

function DoctorLoadRow({ doctor, appointments }) {
  const docAppts = appointments.filter((a) => a.doctor_id === doctor.id)
  const active = docAppts.filter((a) => !['cancelled_by_admin', 'cancelled_by_patient'].includes(a.status))
  const completed = docAppts.filter((a) => a.status === 'completed').length
  const total = active.length
  const MAX = 12
  const pct = Math.min(total / MAX, 1)
  const name = [doctor.last_name, doctor.first_name].filter(Boolean).join(' ')
  const barColor = pct >= 0.75 ? 'bg-rose-400' : pct >= 0.5 ? 'bg-amber-400' : 'bg-emerald-400'

  return (
    <div className="flex items-center gap-3 py-2.5 border-b border-gray-100 last:border-0">
      <div className={`w-8 h-8 rounded-full ${avatarBg(doctor.id)} flex items-center justify-center text-white text-xs font-bold shrink-0`}>
        {(doctor.last_name ?? doctor.first_name ?? '?')[0]}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between mb-1">
          <span className="text-sm font-medium text-gray-800 truncate">{name}</span>
          <span className="text-xs text-gray-500 shrink-0 ml-2 tabular-nums">{total} зап.</span>
        </div>
        <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all duration-500 ${barColor}`}
            style={{ width: `${pct * 100}%` }}
          />
        </div>
      </div>
      {completed > 0 && (
        <span className="text-xs text-gray-400 shrink-0 tabular-nums">{completed} ✓</span>
      )}
    </div>
  )
}

// ─── AppointmentTimeRow ───────────────────────────────────────────────────────

const STATUS_DOT = {
  created:              'bg-blue-400',
  confirmed:            'bg-emerald-400',
  completed:            'bg-gray-300',
  cancelled_by_admin:   'bg-rose-400',
  cancelled_by_patient: 'bg-rose-400',
  no_show:              'bg-amber-400',
}
const STATUS_LABEL = {
  created: 'Ожидает',
  confirmed: 'Подтверждён',
  completed: 'Завершён',
  cancelled_by_admin: 'Отменён',
  cancelled_by_patient: 'Отменён пациентом',
  no_show: 'Не пришёл',
}

function AppointmentRow({ appt }) {
  const time = format(new Date(appt.start_at), 'HH:mm')
  return (
    <div className="flex items-center gap-3 py-2 border-b border-gray-100 last:border-0">
      <span className="text-xs text-gray-400 font-medium tabular-nums w-10 shrink-0">{time}</span>
      <span className={`w-2 h-2 rounded-full shrink-0 ${STATUS_DOT[appt.status] ?? 'bg-gray-300'}`} />
      <span className="text-sm text-gray-800 truncate flex-1">{appt.patient_name}</span>
      <span className="text-xs text-gray-400 truncate max-w-[120px] hidden sm:block">{appt.doctor_full_name}</span>
    </div>
  )
}

// ─── DashboardPage ────────────────────────────────────────────────────────────

export default function DashboardPage() {
  const activeBranchId = useBranchStore((s) => s.activeBranchId)
  const todayLabel = format(new Date(), 'EEEE, d MMMM', { locale: ru })

  const { data: todayAppts = [], isLoading: loadingAppts } = useQuery({
    queryKey: ['dashboard-today', today, activeBranchId ?? null],
    queryFn: () =>
      getAppointments({
        date_from: today,
        date_to: today,
        limit: 200,
        ...(activeBranchId ? { branch_id: activeBranchId } : {}),
      }),
  })

  const { data: doctors = [], isLoading: loadingDoctors } = useQuery({
    queryKey: ['dashboard-doctors', activeBranchId ?? null],
    queryFn: () => getDoctors(activeBranchId ? { branch_id: activeBranchId } : undefined),
  })

  const activeDoctors = doctors.filter((d) => d.is_active)
  const total = todayAppts.length
  const created = todayAppts.filter((a) => a.status === 'created').length
  const confirmed = todayAppts.filter((a) => a.status === 'confirmed').length
  const completed = todayAppts.filter((a) => a.status === 'completed').length
  const cancelled = todayAppts.filter((a) => ['cancelled_by_admin', 'cancelled_by_patient'].includes(a.status)).length
  const noShow = todayAppts.filter((a) => a.status === 'no_show').length

  const upcoming = todayAppts
    .filter((a) => ['created', 'confirmed'].includes(a.status))
    .sort((a, b) => new Date(a.start_at) - new Date(b.start_at))
    .slice(0, 6)

  const serviceCounts = {}
  todayAppts.forEach((a) => {
    if (a.service_name) serviceCounts[a.service_name] = (serviceCounts[a.service_name] || 0) + 1
  })
  const popularServices = Object.entries(serviceCounts).sort((a, b) => b[1] - a[1]).slice(0, 6)

  const loading = loadingAppts || loadingDoctors

  return (
    <div className="p-6 lg:p-8 max-w-6xl mx-auto">

      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Сводка</h1>
        <p className="text-sm text-gray-500 mt-0.5 capitalize">{todayLabel}</p>
      </div>

      {/* KPI row */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <StatCard
          icon={CalendarCheck}
          label="Записей сегодня"
          value={total}
          sub={confirmed > 0 ? `${confirmed} подтверждено` : 'нет подтверждённых'}
          color="blue"
          loading={loadingAppts}
        />
        <StatCard
          icon={CheckCircle2}
          label="Завершено"
          value={completed}
          sub={total > 0 ? `${Math.round((completed / total) * 100)}% от всех` : undefined}
          color="green"
          loading={loadingAppts}
        />
        <StatCard
          icon={Users}
          label="Активных врачей"
          value={activeDoctors.length}
          sub="в системе"
          color="violet"
          loading={loadingDoctors}
        />
        <StatCard
          icon={XCircle}
          label="Отмен / Не пришли"
          value={cancelled + noShow}
          sub={noShow > 0 ? `в т.ч. ${noShow} не пришли` : undefined}
          color={cancelled + noShow > 0 ? 'red' : 'gray'}
          loading={loadingAppts}
        />
      </div>

      {/* Main content grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5 mb-5">

        {/* Status breakdown */}
        <div className="bg-white rounded-xl border border-gray-200 p-5">
          <div className="flex items-center gap-2 mb-4">
            <BarChart3 size={15} className="text-gray-400" />
            <h2 className="text-sm font-semibold text-gray-700">Статусы сегодня</h2>
          </div>
          {loadingAppts ? (
            <div className="space-y-2">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-10 bg-gray-100 rounded-lg animate-pulse" />
              ))}
            </div>
          ) : total === 0 ? (
            <p className="text-sm text-gray-400 text-center py-8">Записей нет</p>
          ) : (
            <div className="space-y-2">
              <StatusRow count={created} label="Ожидают подтверждения" color="blue" />
              <StatusRow count={confirmed} label="Подтверждены" color="green" />
              <StatusRow count={completed} label="Завершены" color="gray" />
              <StatusRow count={cancelled} label="Отменены" color="red" />
              <StatusRow count={noShow} label="Не пришли" color="amber" />
            </div>
          )}
        </div>

        {/* Doctor load */}
        <div className="bg-white rounded-xl border border-gray-200 p-5">
          <div className="flex items-center gap-2 mb-4">
            <UserRound size={15} className="text-gray-400" />
            <h2 className="text-sm font-semibold text-gray-700">Нагрузка врачей</h2>
          </div>
          {loading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-10 bg-gray-100 rounded animate-pulse" />
              ))}
            </div>
          ) : activeDoctors.length === 0 ? (
            <p className="text-sm text-gray-400 text-center py-8">Нет активных врачей</p>
          ) : (
            activeDoctors.map((d) => (
              <DoctorLoadRow key={d.id} doctor={d} appointments={todayAppts} />
            ))
          )}
        </div>

        {/* Upcoming appointments */}
        <div className="bg-white rounded-xl border border-gray-200 p-5">
          <div className="flex items-center gap-2 mb-4">
            <Clock size={15} className="text-gray-400" />
            <h2 className="text-sm font-semibold text-gray-700">Ближайшие записи</h2>
          </div>
          {loadingAppts ? (
            <div className="space-y-2">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-8 bg-gray-100 rounded animate-pulse" />
              ))}
            </div>
          ) : upcoming.length === 0 ? (
            <p className="text-sm text-gray-400 text-center py-8">Активных записей нет</p>
          ) : (
            upcoming.map((a) => <AppointmentRow key={a.id} appt={a} />)
          )}
        </div>
      </div>

      {/* Popular services */}
      {popularServices.length > 0 && (
        <div className="bg-white rounded-xl border border-gray-200 p-5 mb-5">
          <div className="flex items-center gap-2 mb-4">
            <TrendingUp size={15} className="text-gray-400" />
            <h2 className="text-sm font-semibold text-gray-700">Популярные услуги сегодня</h2>
          </div>
          <div className="flex flex-wrap gap-2">
            {popularServices.map(([name, count]) => (
              <div
                key={name}
                className="flex items-center gap-2 px-3 py-1.5 bg-gray-50 border border-gray-200 rounded-lg"
              >
                <span className="text-sm text-gray-700">{name}</span>
                <span className="text-xs font-semibold text-blue-600 bg-blue-50 px-1.5 py-0.5 rounded-full tabular-nums">
                  {count}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Revenue placeholder */}
      <div className="bg-white rounded-xl border border-dashed border-gray-200 p-5 flex items-start gap-3">
        <TrendingUp size={17} className="text-gray-300 mt-0.5 shrink-0" />
        <div>
          <p className="text-sm font-semibold text-gray-400">Финансовая статистика</p>
          <p className="text-xs text-gray-400 mt-0.5">
            Доступна после запуска кассового модуля (v0.3): выручка, оплаты, выплаты врачам, смена-отчёт.
          </p>
        </div>
      </div>

    </div>
  )
}
