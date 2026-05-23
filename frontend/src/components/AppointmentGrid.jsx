import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { format, parseISO, isToday } from 'date-fns'
import { Users } from 'lucide-react'
import { getAppointments } from '../api/appointments'
import { getDoctors } from '../api/doctors'

// ─── Layout constants ─────────────────────────────────────────────────────────

const DAY_START = 8 * 60    // 480 min → 08:00
const DAY_END = 20 * 60     // 1200 min → 20:00
const GRID_H = DAY_END - DAY_START  // 720 px (1 px = 1 min)
const TIME_W = 56
const MIN_COL_W = 220

const HOUR_LABELS = Array.from({ length: 13 }, (_, i) => ({
  label: `${String(8 + i).padStart(2, '0')}:00`,
  top: i * 60,
}))

// 24 lines: 08:30, 09:00, 09:30, …, 19:30, 20:00
const GRID_LINES = Array.from({ length: 24 }, (_, i) => ({
  top: (i + 1) * 30,
  major: (i + 1) % 2 === 0,
}))

// ─── Status colour map ────────────────────────────────────────────────────────

const EVT_BG = {
  created:              'bg-blue-50   border-l-[3px] border-blue-500',
  confirmed:            'bg-emerald-50 border-l-[3px] border-emerald-500',
  completed:            'bg-gray-100  border-l-[3px] border-gray-400',
  cancelled_by_admin:   'bg-rose-50   border-l-[3px] border-rose-400',
  cancelled_by_patient: 'bg-rose-50   border-l-[3px] border-rose-400',
  no_show:              'bg-amber-50  border-l-[3px] border-amber-400',
}

const EVT_TEXT = {
  created:              'text-blue-900',
  confirmed:            'text-emerald-900',
  completed:            'text-gray-500',
  cancelled_by_admin:   'text-rose-700',
  cancelled_by_patient: 'text-rose-700',
  no_show:              'text-amber-800',
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function fullName(d) {
  return [d.last_name, d.first_name, d.middle_name].filter(Boolean).join(' ')
}

function eventPos(appt) {
  const s = parseISO(appt.start_at)
  const e = parseISO(appt.end_at)
  const sMin = s.getHours() * 60 + s.getMinutes()
  const eMin = e.getHours() * 60 + e.getMinutes()
  const top = Math.max(sMin, DAY_START) - DAY_START
  const height = Math.max(Math.min(eMin, DAY_END) - Math.max(sMin, DAY_START), 24)
  return { top, height }
}

function minToTime(absMin) {
  const h = Math.floor(absMin / 60)
  const m = absMin % 60
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
}

function getNowTop() {
  const n = new Date()
  const mins = n.getHours() * 60 + n.getMinutes()
  if (mins < DAY_START || mins > DAY_END) return null
  return mins - DAY_START
}

function loadColor(pct) {
  if (pct >= 0.75) return 'bg-rose-400'
  if (pct >= 0.5) return 'bg-amber-400'
  return 'bg-emerald-400'
}

const AVATAR_COLORS = [
  'bg-blue-500', 'bg-emerald-500', 'bg-violet-500',
  'bg-amber-500', 'bg-rose-500', 'bg-cyan-500',
]
function avatarBg(id) { return AVATAR_COLORS[id % AVATAR_COLORS.length] }

const STATUS_DOT = {
  created:              'bg-blue-500',
  confirmed:            'bg-emerald-500',
  completed:            'bg-gray-400',
  cancelled_by_admin:   'bg-rose-400',
  cancelled_by_patient: 'bg-rose-400',
  no_show:              'bg-amber-400',
}

function doctorStats(appointments, doctorId) {
  const active = appointments.filter(
    (a) =>
      a.doctor_id === doctorId &&
      a.status !== 'cancelled_by_admin' &&
      a.status !== 'cancelled_by_patient',
  )
  const mins = active.reduce(
    (s, a) => s + (new Date(a.end_at) - new Date(a.start_at)) / 60000,
    0,
  )
  return { count: active.length, loadPct: Math.min(mins / GRID_H, 1) }
}

// ─── DoctorCol ────────────────────────────────────────────────────────────────

function DoctorCol({ doctor, appointments, onEventClick, onSlotClick, nowTop }) {
  const appts = appointments.filter((a) => a.doctor_id === doctor.id)
  const [hoverSnap, setHoverSnap] = useState(null)

  const getSnap = (e) => {
    const y = e.clientY - e.currentTarget.getBoundingClientRect().top
    return Math.min(Math.floor(y / 30) * 30, GRID_H - 30)
  }

  return (
    <div
      className="relative border-r border-gray-100 last:border-r-0 cursor-cell"
      style={{ height: `${GRID_H}px`, minWidth: `${MIN_COL_W}px` }}
      onClick={(e) => onSlotClick(doctor.id, minToTime(DAY_START + getSnap(e)))}
      onMouseMove={(e) => setHoverSnap(getSnap(e))}
      onMouseLeave={() => setHoverSnap(null)}
    >
      {/* Hour / half-hour guide lines */}
      {GRID_LINES.map(({ top, major }) => (
        <div
          key={top}
          className={`absolute inset-x-0 pointer-events-none ${
            major
              ? 'border-t border-gray-200'
              : 'border-t border-dashed border-gray-100'
          }`}
          style={{ top: `${top}px` }}
        />
      ))}

      {/* Current time line — spans each column for correct sticky behaviour */}
      {nowTop !== null && (
        <div
          className="absolute inset-x-0 h-px bg-red-400 pointer-events-none"
          style={{ top: `${nowTop}px`, zIndex: 11 }}
        />
      )}

      {/* Slot hover highlight + time label */}
      {hoverSnap !== null && (
        <div
          className="absolute inset-x-0 pointer-events-none bg-blue-50/70 border border-dashed border-blue-300 rounded-sm"
          style={{ top: `${hoverSnap}px`, height: '30px', zIndex: 12 }}
        >
          <span className="absolute right-1 top-0.5 text-[10px] text-blue-600 font-medium leading-none bg-white/80 px-1 rounded">
            {minToTime(DAY_START + hoverSnap)}
          </span>
        </div>
      )}

      {/* Appointment events */}
      {appts.map((appt) => {
        const { top, height } = eventPos(appt)
        const bg = EVT_BG[appt.status] ?? EVT_BG.created
        const txt = EVT_TEXT[appt.status] ?? EVT_TEXT.created
        return (
          <button
            key={appt.id}
            onClick={(e) => {
              e.stopPropagation()
              onEventClick(appt)
            }}
            className={`absolute left-0.5 right-0.5 ${bg} ${txt} rounded overflow-hidden text-left px-2 py-1 hover:brightness-95 active:brightness-90 transition-all shadow-sm z-10 select-none`}
            style={{ top: `${top}px`, height: `${height}px` }}
          >
            <div className="font-semibold text-xs leading-tight flex items-center gap-1 min-w-0">
              <span className={`w-1.5 h-1.5 rounded-full shrink-0 opacity-90 ${STATUS_DOT[appt.status] ?? 'bg-gray-400'}`} />
              <span className="truncate">{appt.patient_name}</span>
            </div>
            {height >= 36 && (
              <div className="text-[10px] opacity-70 leading-tight mt-0.5 truncate">
                {format(parseISO(appt.start_at), 'HH:mm')}–{format(parseISO(appt.end_at), 'HH:mm')}
              </div>
            )}
            {height >= 54 && appt.service_name && (
              <div className="text-[10px] opacity-60 leading-tight truncate">
                {appt.service_name}
              </div>
            )}
            {height >= 72 && appt.patient_phone && (
              <div className="text-[10px] opacity-60 leading-tight truncate">
                {appt.patient_phone}
              </div>
            )}
          </button>
        )
      })}
    </div>
  )
}

// ─── AppointmentGrid ──────────────────────────────────────────────────────────

export default function AppointmentGrid({ date, branchId, onEventClick, onSlotClick }) {
  const dateStr = format(date, 'yyyy-MM-dd')
  const viewingToday = isToday(date)

  const [nowTop, setNowTop] = useState(() => (viewingToday ? getNowTop() : null))

  useEffect(() => {
    if (!viewingToday) { setNowTop(null); return }
    setNowTop(getNowTop())
    const id = setInterval(() => setNowTop(getNowTop()), 60_000)
    return () => clearInterval(id)
  }, [viewingToday])

  const { data: allDoctors = [], isLoading: loadingDr } = useQuery({
    queryKey: ['grid-doctors', branchId ?? null],
    queryFn: () => getDoctors(branchId ? { branch_id: branchId } : undefined),
  })

  const doctors = allDoctors.filter((d) => d.is_active)

  const { data: appointments = [], isLoading: loadingAp } = useQuery({
    queryKey: ['grid-appointments', dateStr, branchId ?? null],
    queryFn: () =>
      getAppointments({
        date_from: dateStr,
        date_to: dateStr,
        limit: 200,
        ...(branchId ? { branch_id: branchId } : {}),
      }),
  })

  const loading = loadingDr || loadingAp

  if (loading && doctors.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center flex-1 gap-3">
        <div className="flex gap-1.5">
          {[0, 1, 2].map((i) => (
            <div
              key={i}
              className="w-2 h-2 rounded-full bg-blue-300 animate-bounce"
              style={{ animationDelay: `${i * 0.15}s` }}
            />
          ))}
        </div>
        <p className="text-xs text-gray-400">Загрузка расписания…</p>
      </div>
    )
  }

  if (!loading && doctors.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center flex-1 gap-3 text-gray-400">
        <Users size={36} strokeWidth={1.25} className="text-gray-300" />
        <div className="text-center">
          <p className="text-sm font-medium text-gray-500">Нет активных врачей</p>
          <p className="text-xs mt-0.5">Добавьте врачей в разделе «Врачи»</p>
        </div>
      </div>
    )
  }

  return (
    <div className="overflow-auto flex-1 min-h-0 relative bg-white">
      {/* Refresh overlay — doesn't block interaction */}
      {loading && (
        <div className="absolute top-2 right-3 z-40 pointer-events-none">
          <span className="text-[10px] text-gray-400 bg-white/80 px-2 py-0.5 rounded-full shadow-sm">
            Обновление…
          </span>
        </div>
      )}

      <div style={{ minWidth: `${TIME_W + doctors.length * MIN_COL_W}px` }}>

        {/* ── Sticky doctor header row ── */}
        <div className="sticky top-0 z-20 flex border-b border-gray-200 bg-white shadow-sm">
          {/* Corner cell */}
          <div
            className="sticky left-0 z-30 shrink-0 bg-white border-r border-gray-200"
            style={{ width: `${TIME_W}px` }}
          />
          {/* Doctor header cells */}
          {doctors.map((d) => {
            const { count, loadPct } = doctorStats(appointments, d.id)
            return (
              <div
                key={d.id}
                className="relative border-r border-gray-100 last:border-r-0 overflow-hidden"
                style={{ minWidth: `${MIN_COL_W}px`, flex: '1 0 0' }}
              >
                <div className="px-3 py-2.5 flex items-start gap-2">
                  <div className={`w-7 h-7 rounded-full ${avatarBg(d.id)} flex items-center justify-center text-white text-[11px] font-bold shrink-0 mt-0.5`}>
                    {(d.last_name ?? d.first_name ?? '?')[0]}
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-semibold text-gray-800 truncate">{fullName(d)}</p>
                    <div className="flex items-center gap-2 mt-0.5">
                      {d.cabinet && (
                        <span className="text-xs text-gray-400">Каб. {d.cabinet}</span>
                      )}
                      {count > 0 && (
                        <span className="text-[10px] text-gray-400 font-medium shrink-0">{count} зап.</span>
                      )}
                    </div>
                    {(d.directions ?? []).length > 0 && (
                      <div className="flex gap-1 mt-1 flex-wrap">
                        {(d.directions ?? []).slice(0, 2).map((dir) => (
                          <span key={dir.id} className="text-[10px] px-1.5 py-0.5 rounded-full bg-blue-50 text-blue-600 border border-blue-100 leading-none">
                            {dir.name}
                          </span>
                        ))}
                        {(d.directions ?? []).length > 2 && (
                          <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-gray-100 text-gray-500 leading-none">
                            +{d.directions.length - 2}
                          </span>
                        )}
                      </div>
                    )}
                  </div>
                </div>
                {/* Doctor load bar */}
                {loadPct > 0 && (
                  <div className="absolute bottom-0 left-0 right-0 h-[3px] bg-gray-100">
                    <div
                      className={`h-full ${loadColor(loadPct)} transition-all duration-300`}
                      style={{ width: `${loadPct * 100}%` }}
                    />
                  </div>
                )}
              </div>
            )
          })}
        </div>

        {/* ── Body: time axis + doctor columns ── */}
        <div className="flex">

          {/* Time axis — sticky left */}
          <div
            className="sticky left-0 z-10 shrink-0 bg-white border-r border-gray-200 select-none relative"
            style={{ width: `${TIME_W}px`, height: `${GRID_H}px` }}
          >
            {HOUR_LABELS.map(({ label, top }) => (
              <span
                key={label}
                className="absolute right-2 text-[11px] text-gray-400 -translate-y-1/2"
                style={{ top: `${top}px` }}
              >
                {label}
              </span>
            ))}
            {/* Current time dot anchored to time axis right edge */}
            {nowTop !== null && (
              <div
                className="absolute z-20 -translate-y-1/2 pointer-events-none"
                style={{ top: `${nowTop}px`, right: '-5px' }}
              >
                <div className="w-2.5 h-2.5 bg-red-500 rounded-full ring-2 ring-red-200" />
              </div>
            )}
          </div>

          {/* Doctor columns */}
          {doctors.map((d) => (
            <DoctorCol
              key={d.id}
              doctor={d}
              appointments={appointments}
              onEventClick={onEventClick}
              onSlotClick={onSlotClick}
              nowTop={nowTop}
            />
          ))}

        </div>
      </div>
    </div>
  )
}
