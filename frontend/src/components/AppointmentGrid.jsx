import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { format, parseISO, isToday } from 'date-fns'
import { Users, MoreVertical } from 'lucide-react'
import { getAppointments } from '../api/appointments'
import { getDoctors } from '../api/doctors'

// ─── Layout constants ─────────────────────────────────────────────────────────

const DAY_START = 8 * 60    // 480 min → 08:00
const DAY_END = 20 * 60     // 1200 min → 20:00
const GRID_H = DAY_END - DAY_START  // 720 px (1 px = 1 min)
const TIME_W = 64
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

const HATCH = {
  backgroundImage: 'repeating-linear-gradient(135deg, transparent, transparent 3px, rgba(107,114,128,0.1) 3px, rgba(107,114,128,0.1) 6px)',
  backgroundColor: 'rgba(243,244,246,0.7)',
}

function DoctorCol({ doctor, appointments, onEventClick, onSlotClick, nowTop, workHours, onDayAction, date }) {
  const appts = appointments.filter((a) => a.doctor_id === doctor.id)
  const [hoverSnap, setHoverSnap] = useState(null)

  const getSnap = (e) => {
    const y = e.clientY - e.currentTarget.getBoundingClientRect().top
    return Math.min(Math.floor(y / 30) * 30, GRID_H - 30)
  }

  const handleClick = (e) => {
    const snap = getSnap(e)
    const absMin = DAY_START + snap
    if (workHours != null) {
      if (absMin < workHours.startMin || absMin >= workHours.endMin) return
    }
    onSlotClick(doctor.id, minToTime(absMin))
  }

  const preBlockH    = workHours != null ? Math.max(0, workHours.startMin - DAY_START) : 0
  const postBlockTop = workHours != null ? Math.max(0, workHours.endMin   - DAY_START) : GRID_H

  return (
    <div
      className="relative border-r border-gray-100 last:border-r-0 cursor-cell"
      style={{ height: `${GRID_H}px`, minWidth: `${MIN_COL_W}px` }}
      onClick={handleClick}
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

      {/* Pre-work blocked zone — clickable to manage schedule */}
      {preBlockH > 0 && (
        <div
          className="absolute inset-x-0 top-0 cursor-pointer hover:brightness-95 transition-colors group"
          style={{ height: `${preBlockH}px`, zIndex: 6, ...HATCH }}
          onClick={(e) => { e.stopPropagation(); onDayAction?.({ doctorId: doctor.id, doctor, date }) }}
          title="Управление расписанием"
        >
          <span className="absolute bottom-1 left-1/2 -translate-x-1/2 text-[9px] text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap bg-white/80 px-1.5 py-0.5 rounded-sm shadow-sm">
            Нажмите для управления
          </span>
        </div>
      )}

      {/* Post-work blocked zone — clickable to manage schedule */}
      {postBlockTop < GRID_H && (
        <div
          className="absolute inset-x-0 cursor-pointer hover:brightness-95 transition-colors group"
          style={{ top: `${postBlockTop}px`, bottom: 0, zIndex: 6, ...HATCH }}
          onClick={(e) => { e.stopPropagation(); onDayAction?.({ doctorId: doctor.id, doctor, date }) }}
          title="Управление расписанием"
        >
          <span className="absolute top-1 left-1/2 -translate-x-1/2 text-[9px] text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap bg-white/80 px-1.5 py-0.5 rounded-sm shadow-sm">
            Нажмите для управления
          </span>
        </div>
      )}

      {/* Past-time overlay (today only) */}
      {nowTop !== null && nowTop > 0 && (
        <div
          className="absolute inset-x-0 top-0 pointer-events-none bg-gray-100/40"
          style={{ height: `${nowTop}px`, zIndex: 5 }}
        />
      )}

      {/* Current time line */}
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
        const bg  = EVT_BG[appt.status]   ?? EVT_BG.created
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

export default function AppointmentGrid({
  date, branchId, onEventClick, onSlotClick, visibleDoctorIds, workingHoursMap, onDayAction,
}) {
  const dateStr      = format(date, 'yyyy-MM-dd')
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

  const activeDoctors = allDoctors.filter((d) => d.is_active)
  const doctors = visibleDoctorIds
    ? activeDoctors.filter((d) => visibleDoctorIds.includes(d.id))
    : activeDoctors

  const { data: appointments = [], isLoading: loadingAp } = useQuery({
    queryKey: ['grid-appointments', dateStr, branchId ?? null],
    queryFn: () =>
      getAppointments({
        date_from: dateStr,
        date_to:   dateStr,
        limit:     200,
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
          <p className="text-sm font-medium text-gray-500">Нет врачей на этот день</p>
          <p className="text-xs mt-0.5">Попробуйте изменить дату или фильтры</p>
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
          {/* Doctor header cells — compact */}
          {doctors.map((d) => {
            const { count, loadPct } = doctorStats(appointments, d.id)
            return (
              <div
                key={d.id}
                className="relative border-r border-gray-100 last:border-r-0 overflow-hidden"
                style={{ minWidth: `${MIN_COL_W}px`, flex: '1 0 0' }}
              >
                <div className="px-2 py-2 flex items-center gap-2">
                  <div className={`w-6 h-6 rounded-full ${avatarBg(d.id)} flex items-center justify-center text-white text-[10px] font-bold shrink-0`}>
                    {(d.last_name ?? d.first_name ?? '?')[0]}
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="text-xs font-semibold text-gray-800 truncate leading-tight">{fullName(d)}</p>
                    <div className="flex items-center gap-1.5 mt-0.5 flex-wrap">
                      {d.cabinet && (
                        <span className="text-[10px] text-gray-400">Каб.{d.cabinet}</span>
                      )}
                      {count > 0 && (
                        <span className="text-[10px] text-gray-500 font-medium">{count} зап.</span>
                      )}
                      {(d.directions ?? []).slice(0, 1).map((dir) => (
                        <span key={dir.id} className="text-[10px] px-1 rounded bg-blue-50 text-blue-600 leading-tight">
                          {dir.name}
                        </span>
                      ))}
                      {(d.directions ?? []).length > 1 && (
                        <span className="text-[10px] text-gray-400">+{d.directions.length - 1}</span>
                      )}
                    </div>
                  </div>
                  {/* Quick day-action button */}
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      onDayAction?.({ doctorId: d.id, doctor: d, date })
                    }}
                    className="shrink-0 p-1 rounded hover:bg-gray-100 text-gray-300 hover:text-gray-600 transition-colors"
                    title="Управление расписанием врача"
                  >
                    <MoreVertical size={13} />
                  </button>
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
                className="absolute right-2 text-xs text-gray-500 font-medium -translate-y-1/2"
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
              workHours={workingHoursMap?.get(d.id)}
              onDayAction={onDayAction}
              date={date}
            />
          ))}

        </div>
      </div>
    </div>
  )
}
