import { useQuery } from '@tanstack/react-query'
import { format, parseISO } from 'date-fns'
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

// ─── DoctorCol ────────────────────────────────────────────────────────────────

function DoctorCol({ doctor, appointments, onEventClick, onSlotClick }) {
  const appts = appointments.filter((a) => a.doctor_id === doctor.id)

  const handleClick = (e) => {
    const y = e.clientY - e.currentTarget.getBoundingClientRect().top
    const snapped = Math.min(Math.floor(y / 30) * 30, GRID_H - 30)
    onSlotClick(doctor.id, minToTime(DAY_START + snapped))
  }

  return (
    <div
      className="relative border-r border-gray-100 last:border-r-0 cursor-cell group"
      style={{ height: `${GRID_H}px`, minWidth: `${MIN_COL_W}px` }}
      onClick={handleClick}
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

      {/* Subtle hover tint on empty space */}
      <div className="absolute inset-0 opacity-0 group-hover:opacity-100 bg-blue-50/20 pointer-events-none transition-opacity" />

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
            className={`absolute left-1 right-1 ${bg} ${txt} rounded-sm overflow-hidden text-left px-2 py-0.5 hover:brightness-95 transition-all shadow-sm z-10`}
            style={{ top: `${top}px`, height: `${height}px` }}
          >
            <div className="font-semibold text-xs leading-tight truncate">
              {appt.patient_name}
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
          </button>
        )
      })}
    </div>
  )
}

// ─── AppointmentGrid ──────────────────────────────────────────────────────────

export default function AppointmentGrid({ date, branchId, onEventClick, onSlotClick }) {
  const dateStr = format(date, 'yyyy-MM-dd')

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
      <div className="flex items-center justify-center flex-1 text-sm text-gray-400">
        Загрузка...
      </div>
    )
  }

  if (!loading && doctors.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center flex-1 gap-2 text-gray-400">
        <p className="text-sm font-medium text-gray-500">Нет активных врачей</p>
        <p className="text-xs">Добавьте врачей в разделе «Врачи»</p>
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
          {/* Doctor name cells */}
          {doctors.map((d) => (
            <div
              key={d.id}
              className="px-3 py-2.5 border-r border-gray-100 last:border-r-0"
              style={{ minWidth: `${MIN_COL_W}px`, flex: '1 0 0' }}
            >
              <p className="text-sm font-semibold text-gray-800 truncate">{fullName(d)}</p>
              {d.cabinet && (
                <p className="text-xs text-gray-400 truncate mt-0.5">Кабинет {d.cabinet}</p>
              )}
            </div>
          ))}
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
          </div>

          {/* Doctor columns */}
          {doctors.map((d) => (
            <DoctorCol
              key={d.id}
              doctor={d}
              appointments={appointments}
              onEventClick={onEventClick}
              onSlotClick={onSlotClick}
            />
          ))}

        </div>
      </div>
    </div>
  )
}
