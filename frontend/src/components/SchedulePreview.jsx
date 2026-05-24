import { useMemo } from 'react'
import { addDays, format, startOfDay } from 'date-fns'
import { ru } from 'date-fns/locale'

// Build a lookup: "YYYY-MM-DD" → exception object
function buildExceptionMap(exceptions) {
  const map = {}
  for (const ex of exceptions ?? []) {
    const key = ex.date?.slice(0, 10) ?? format(new Date(ex.date), 'yyyy-MM-dd')
    map[key] = ex
  }
  return map
}

// Build a lookup: day_of_week (1–7) → array of { start_time, end_time }
function buildScheduleMap(schedule) {
  const map = {}
  for (const h of schedule ?? []) {
    if (!map[h.day_of_week]) map[h.day_of_week] = []
    map[h.day_of_week].push(h)
  }
  return map
}

// Convert date to DB day_of_week (1=Mon … 7=Sun)
function toDow(date) {
  const wd = date.getDay() // 0=Sun
  return wd === 0 ? 7 : wd
}

function toHHMM(isoTime) {
  if (!isoTime) return ''
  const t = isoTime.split('T')[1] ?? isoTime
  return t.slice(0, 5)
}

function DayCell({ date, scheduleMap, exceptionMap }) {
  const dateStr = format(date, 'yyyy-MM-dd')
  const dow = toDow(date)
  const ex = exceptionMap[dateStr]
  const slots = scheduleMap[dow] ?? []

  let bg = 'bg-gray-50 text-gray-300'
  let label = null

  if (ex) {
    if (ex.type === 'day_off') {
      bg = 'bg-amber-50 text-amber-400'
      label = 'выходной'
    } else if (ex.type === 'custom_working_hours') {
      bg = 'bg-amber-50 text-amber-700'
      label = `${toHHMM(ex.start_time)}–${toHHMM(ex.end_time)}`
    }
  } else if (slots.length > 0) {
    bg = 'bg-emerald-50 text-emerald-700'
    const sorted = [...slots].sort((a, b) => (a.start_time < b.start_time ? -1 : 1))
    label = `${toHHMM(sorted[0].start_time)}–${toHHMM(sorted[sorted.length - 1].end_time)}`
  }

  return (
    <div className={`rounded-md p-1.5 text-center min-w-0 ${bg}`}>
      <div className="text-[11px] font-semibold leading-none mb-0.5">{format(date, 'd')}</div>
      {label && (
        <div className="text-[9px] leading-tight truncate">{label}</div>
      )}
    </div>
  )
}

export default function SchedulePreview({ schedule, exceptions }) {
  const scheduleMap = useMemo(() => buildScheduleMap(schedule), [schedule])
  const exceptionMap = useMemo(() => buildExceptionMap(exceptions), [exceptions])

  const today = startOfDay(new Date())
  const weeks = useMemo(() => {
    const result = []
    // Find Monday of current week
    const wd = today.getDay() || 7 // 1=Mon..7=Sun
    const monday = addDays(today, 1 - wd)
    for (let w = 0; w < 4; w++) {
      const week = []
      for (let d = 0; d < 7; d++) {
        week.push(addDays(monday, w * 7 + d))
      }
      result.push(week)
    }
    return result
  }, [today])

  const DAY_LABELS = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс']

  return (
    <div className="mt-6">
      <h3 className="text-sm font-medium text-gray-700 mb-3">Предпросмотр на 4 недели</h3>
      <div className="border border-gray-200 rounded-xl p-3 bg-white">
        {/* Header */}
        <div className="grid grid-cols-7 gap-1 mb-1">
          {DAY_LABELS.map((l) => (
            <div key={l} className="text-center text-[10px] font-medium text-gray-400 uppercase tracking-wide">
              {l}
            </div>
          ))}
        </div>

        {/* Weeks */}
        <div className="space-y-1">
          {weeks.map((week, wi) => (
            <div key={wi} className="grid grid-cols-7 gap-1">
              {week.map((date) => (
                <DayCell
                  key={date.toISOString()}
                  date={date}
                  scheduleMap={scheduleMap}
                  exceptionMap={exceptionMap}
                />
              ))}
            </div>
          ))}
        </div>

        {/* Legend */}
        <div className="flex items-center gap-4 mt-3 pt-2 border-t border-gray-100">
          <span className="flex items-center gap-1.5 text-[11px] text-gray-500">
            <span className="w-3 h-3 rounded bg-emerald-100 inline-block" /> Рабочий
          </span>
          <span className="flex items-center gap-1.5 text-[11px] text-gray-500">
            <span className="w-3 h-3 rounded bg-amber-100 inline-block" /> Исключение
          </span>
          <span className="flex items-center gap-1.5 text-[11px] text-gray-500">
            <span className="w-3 h-3 rounded bg-gray-100 inline-block" /> Выходной
          </span>
        </div>
      </div>
    </div>
  )
}
