import { useState, useEffect } from 'react'
import { Plus, X } from 'lucide-react'

const DAYS = [
  { dow: 1, label: 'Понедельник' },
  { dow: 2, label: 'Вторник' },
  { dow: 3, label: 'Среда' },
  { dow: 4, label: 'Четверг' },
  { dow: 5, label: 'Пятница' },
  { dow: 6, label: 'Суббота' },
  { dow: 7, label: 'Воскресенье' },
]

function toHHMM(isoTime) {
  // "0001-01-01T09:00:00Z" or "HH:MM:SS" → "HH:MM"
  const t = isoTime?.split('T')[1] ?? isoTime ?? ''
  return t.slice(0, 5) || '09:00'
}

// Build row state from the raw schedule array.
// A day with two rows is treated as having a lunch break.
function initRows(schedule) {
  // Group rows by day_of_week, sorted by start_time ascending.
  const byDow = {}
  for (const h of schedule ?? []) {
    const dow = h.day_of_week
    if (!byDow[dow]) byDow[dow] = []
    byDow[dow].push(h)
  }
  for (const dow in byDow) {
    byDow[dow].sort((a, b) => (a.start_time < b.start_time ? -1 : 1))
  }

  return DAYS.map(({ dow }) => {
    const rows = byDow[dow]
    if (!rows || rows.length === 0) {
      return { dow, active: false, start: '09:00', end: '18:00', hasLunch: false, lunchStart: '13:00', lunchEnd: '14:00' }
    }
    if (rows.length >= 2) {
      return {
        dow,
        active: true,
        start: toHHMM(rows[0].start_time),
        lunchStart: toHHMM(rows[0].end_time),
        lunchEnd: toHHMM(rows[1].start_time),
        end: toHHMM(rows[1].end_time),
        hasLunch: true,
      }
    }
    return {
      dow,
      active: true,
      start: toHHMM(rows[0].start_time),
      end: toHHMM(rows[0].end_time),
      hasLunch: false,
      lunchStart: '13:00',
      lunchEnd: '14:00',
    }
  })
}

function TimeInput({ value, disabled, onChange, className = '' }) {
  return (
    <input
      type="time"
      value={value}
      disabled={disabled}
      onChange={(e) => onChange(e.target.value)}
      className={`border border-gray-300 rounded px-2 py-1 text-sm disabled:opacity-40 focus:outline-none focus:ring-1 focus:ring-blue-500 ${className}`}
    />
  )
}

export default function ScheduleEditor({ schedule, onSubmit, isLoading }) {
  const [rows, setRows] = useState(() => initRows(schedule))

  useEffect(() => {
    setRows(initRows(schedule))
  }, [schedule])

  const update = (dow, patch) =>
    setRows((prev) => prev.map((r) => (r.dow === dow ? { ...r, ...patch } : r)))

  const handleSubmit = (e) => {
    e.preventDefault()
    const items = []
    for (const row of rows) {
      if (!row.active) continue
      if (row.hasLunch) {
        items.push({ day_of_week: row.dow, start_time: row.start, end_time: row.lunchStart })
        items.push({ day_of_week: row.dow, start_time: row.lunchEnd, end_time: row.end })
      } else {
        items.push({ day_of_week: row.dow, start_time: row.start, end_time: row.end })
      }
    }
    onSubmit(items)
  }

  return (
    <form onSubmit={handleSubmit}>
      <div className="space-y-2">
        {rows.map((row) => {
          const day = DAYS.find((d) => d.dow === row.dow)
          return (
            <div
              key={row.dow}
              className={`p-3 rounded-lg border transition-colors ${
                row.active ? 'border-blue-200 bg-blue-50/40' : 'border-gray-200 bg-white'
              }`}
            >
              {/* Day toggle + main times */}
              <div className="flex items-center gap-4">
                <label className="flex items-center gap-2 w-36 cursor-pointer shrink-0">
                  <input
                    type="checkbox"
                    className="rounded"
                    checked={row.active}
                    onChange={() => update(row.dow, { active: !row.active })}
                  />
                  <span className={`text-sm font-medium ${row.active ? 'text-gray-900' : 'text-gray-400'}`}>
                    {day.label}
                  </span>
                </label>

                {row.hasLunch ? (
                  /* Lunch mode: start — lunchStart | lunchEnd — end */
                  <div className="flex items-center gap-2 flex-wrap">
                    <TimeInput value={row.start} disabled={!row.active} onChange={(v) => update(row.dow, { start: v })} />
                    <span className="text-gray-400 text-sm">—</span>
                    <TimeInput value={row.lunchStart} disabled={!row.active} onChange={(v) => update(row.dow, { lunchStart: v })} />
                    <span className="text-xs text-gray-400 px-1">обед</span>
                    <TimeInput value={row.lunchEnd} disabled={!row.active} onChange={(v) => update(row.dow, { lunchEnd: v })} />
                    <span className="text-gray-400 text-sm">—</span>
                    <TimeInput value={row.end} disabled={!row.active} onChange={(v) => update(row.dow, { end: v })} />
                    <button
                      type="button"
                      onClick={() => update(row.dow, { hasLunch: false })}
                      disabled={!row.active}
                      className="ml-1 p-0.5 text-gray-400 hover:text-red-500 disabled:opacity-30 transition-colors"
                      title="Убрать обед"
                    >
                      <X size={14} />
                    </button>
                  </div>
                ) : (
                  /* Normal mode: start — end + add lunch button */
                  <div className="flex items-center gap-2">
                    <TimeInput value={row.start} disabled={!row.active} onChange={(v) => update(row.dow, { start: v })} />
                    <span className="text-gray-400 text-sm">—</span>
                    <TimeInput value={row.end} disabled={!row.active} onChange={(v) => update(row.dow, { end: v })} />
                    {row.active && (
                      <button
                        type="button"
                        onClick={() => update(row.dow, { hasLunch: true })}
                        className="ml-1 flex items-center gap-1 text-xs text-blue-500 hover:text-blue-700 transition-colors"
                        title="Добавить обед"
                      >
                        <Plus size={12} />
                        обед
                      </button>
                    )}
                  </div>
                )}
              </div>
            </div>
          )
        })}
      </div>
      <div className="mt-4 flex justify-end">
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-60 transition-colors"
        >
          {isLoading ? 'Сохранение...' : 'Сохранить расписание'}
        </button>
      </div>
    </form>
  )
}
