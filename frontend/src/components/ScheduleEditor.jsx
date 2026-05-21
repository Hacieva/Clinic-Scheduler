import { useState, useEffect } from 'react'

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
  // "0001-01-01T09:00:00Z" → "09:00"
  const t = isoTime?.split('T')[1] ?? ''
  return t.slice(0, 5) || '09:00'
}

function initRows(schedule) {
  const byDow = Object.fromEntries((schedule ?? []).map((h) => [h.day_of_week, h]))
  return DAYS.map(({ dow }) => ({
    dow,
    active: !!byDow[dow],
    start: byDow[dow] ? toHHMM(byDow[dow].start_time) : '09:00',
    end: byDow[dow] ? toHHMM(byDow[dow].end_time) : '18:00',
  }))
}

export default function ScheduleEditor({ schedule, onSubmit, isLoading }) {
  const [rows, setRows] = useState(() => initRows(schedule))

  useEffect(() => {
    setRows(initRows(schedule))
  }, [schedule])

  const toggle = (dow) =>
    setRows((prev) =>
      prev.map((r) => (r.dow === dow ? { ...r, active: !r.active } : r)),
    )

  const setTime = (dow, field, value) =>
    setRows((prev) =>
      prev.map((r) => (r.dow === dow ? { ...r, [field]: value } : r)),
    )

  const handleSubmit = (e) => {
    e.preventDefault()
    const items = rows
      .filter((r) => r.active)
      .map((r) => ({ day_of_week: r.dow, start_time: r.start, end_time: r.end }))
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
              className={`flex items-center gap-4 p-3 rounded-lg border transition-colors ${
                row.active ? 'border-blue-200 bg-blue-50/40' : 'border-gray-200 bg-white'
              }`}
            >
              <label className="flex items-center gap-2 w-36 cursor-pointer shrink-0">
                <input
                  type="checkbox"
                  className="rounded"
                  checked={row.active}
                  onChange={() => toggle(row.dow)}
                />
                <span
                  className={`text-sm font-medium ${
                    row.active ? 'text-gray-900' : 'text-gray-400'
                  }`}
                >
                  {day.label}
                </span>
              </label>
              <div className="flex items-center gap-2">
                <input
                  type="time"
                  value={row.start}
                  disabled={!row.active}
                  onChange={(e) => setTime(row.dow, 'start', e.target.value)}
                  className="border border-gray-300 rounded px-2 py-1 text-sm disabled:opacity-40 focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
                <span className="text-gray-400 text-sm">—</span>
                <input
                  type="time"
                  value={row.end}
                  disabled={!row.active}
                  onChange={(e) => setTime(row.dow, 'end', e.target.value)}
                  className="border border-gray-300 rounded px-2 py-1 text-sm disabled:opacity-40 focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
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
