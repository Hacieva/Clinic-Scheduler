import { format, startOfWeek, addDays, isSameDay } from 'date-fns'
import { ru } from 'date-fns/locale'
import { ChevronLeft, ChevronRight } from 'lucide-react'

const DAY_START_MIN = 8 * 60   // 480
const DAY_END_MIN = 20 * 60    // 1200
const GRID_HEIGHT = DAY_END_MIN - DAY_START_MIN  // 720px = 1px per minute

const STATUS_COLOR = {
  created: 'bg-blue-500 hover:bg-blue-600',
  confirmed: 'bg-green-500 hover:bg-green-600',
  completed: 'bg-gray-400 hover:bg-gray-500',
  cancelled_by_admin: 'bg-red-400 hover:bg-red-500',
  cancelled_by_patient: 'bg-red-400 hover:bg-red-500',
  no_show: 'bg-red-400 hover:bg-red-500',
}

const HOUR_MARKS = Array.from({ length: 13 }, (_, i) => ({
  label: `${String(8 + i).padStart(2, '0')}:00`,
  top: i * 60,
}))

function eventPos(event) {
  const startMin = event.start.getHours() * 60 + event.start.getMinutes()
  const endMin = event.end.getHours() * 60 + event.end.getMinutes()
  const clampedStart = Math.max(startMin, DAY_START_MIN)
  const clampedEnd = Math.min(endMin, DAY_END_MIN)
  return {
    top: `${clampedStart - DAY_START_MIN}px`,
    height: `${Math.max(clampedEnd - clampedStart, 20)}px`,
  }
}

export default function WeekCalendar({ events = [], view, date, onEventClick, onNavigate, onViewChange }) {
  const weekStart = startOfWeek(date, { weekStartsOn: 1 })
  const days = view === 'week'
    ? Array.from({ length: 7 }, (_, i) => addDays(weekStart, i))
    : [date]

  const headerLabel = view === 'week'
    ? `${format(weekStart, 'd MMM', { locale: ru })} – ${format(addDays(weekStart, 6), 'd MMM yyyy', { locale: ru })}`
    : format(date, 'd MMMM yyyy', { locale: ru })

  const eventsForDay = (day) => events.filter((e) => isSameDay(e.start, day))

  return (
    <div className="flex flex-col min-h-0">
      {/* Toolbar */}
      <div className="flex items-center justify-between px-4 py-3 bg-white border-b border-gray-200 sticky top-0 z-10 shrink-0">
        <div className="flex items-center gap-2">
          <button
            onClick={() => onNavigate('prev')}
            className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-500 transition-colors"
          >
            <ChevronLeft size={18} />
          </button>
          <button
            onClick={() => onNavigate('today')}
            className="px-3 py-1.5 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 text-gray-700 transition-colors"
          >
            Сегодня
          </button>
          <button
            onClick={() => onNavigate('next')}
            className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-500 transition-colors"
          >
            <ChevronRight size={18} />
          </button>
          <span className="ml-2 text-sm font-medium text-gray-900 capitalize">{headerLabel}</span>
        </div>
        <div className="flex rounded-lg border border-gray-200 overflow-hidden text-sm">
          <button
            onClick={() => onViewChange('week')}
            className={`px-3 py-1.5 transition-colors ${
              view === 'week' ? 'bg-blue-600 text-white' : 'text-gray-600 hover:bg-gray-50'
            }`}
          >
            Неделя
          </button>
          <button
            onClick={() => onViewChange('day')}
            className={`px-3 py-1.5 border-l border-gray-200 transition-colors ${
              view === 'day' ? 'bg-blue-600 text-white' : 'text-gray-600 hover:bg-gray-50'
            }`}
          >
            День
          </button>
        </div>
      </div>

      {/* Grid wrapper — scrolls vertically */}
      <div className="flex overflow-y-auto overflow-x-auto">
        {/* Time axis */}
        <div className="w-14 shrink-0 select-none">
          <div className="h-10 border-b border-gray-200 bg-white" />
          <div
            className="relative bg-white border-r border-gray-200"
            style={{ height: `${GRID_HEIGHT}px` }}
          >
            {HOUR_MARKS.map(({ label, top }) => (
              <div
                key={label}
                className="absolute right-2 text-xs text-gray-400 -translate-y-1/2"
                style={{ top: `${top}px` }}
              >
                {label}
              </div>
            ))}
          </div>
        </div>

        {/* Day columns */}
        <div className="flex flex-1 min-w-0">
          {days.map((day) => {
            const isToday = isSameDay(day, new Date())
            return (
              <div
                key={day.toISOString()}
                className="flex-1 flex flex-col min-w-0 border-r border-gray-100 last:border-r-0"
              >
                {/* Day header */}
                <div
                  className={`h-10 flex flex-col items-center justify-center border-b border-gray-200 shrink-0 ${
                    isToday ? 'bg-blue-50' : 'bg-white'
                  }`}
                >
                  <span className="text-xs text-gray-400 uppercase tracking-wide leading-none">
                    {format(day, 'EEE', { locale: ru })}
                  </span>
                  <span
                    className={`text-sm font-semibold leading-tight mt-0.5 ${
                      isToday ? 'text-blue-600' : 'text-gray-900'
                    }`}
                  >
                    {format(day, 'd')}
                  </span>
                </div>

                {/* Events area */}
                <div className="relative" style={{ height: `${GRID_HEIGHT}px` }}>
                  {/* Hour grid lines */}
                  {HOUR_MARKS.slice(1).map(({ top }) => (
                    <div
                      key={top}
                      className="absolute left-0 right-0 border-t border-gray-100 pointer-events-none"
                      style={{ top: `${top}px` }}
                    />
                  ))}

                  {/* Appointment events */}
                  {eventsForDay(day).map((event) => {
                    const pos = eventPos(event)
                    const colorClass = STATUS_COLOR[event.status] ?? 'bg-gray-400 hover:bg-gray-500'
                    return (
                      <button
                        key={event.id}
                        onClick={() => onEventClick(event)}
                        style={{ ...pos, position: 'absolute', left: '3px', right: '3px' }}
                        className={`${colorClass} text-white text-xs rounded-md px-1.5 py-0.5 text-left overflow-hidden transition-colors`}
                      >
                        <div className="font-medium truncate leading-tight">{event.title}</div>
                        <div className="opacity-75 truncate leading-tight">
                          {format(event.start, 'HH:mm')}–{format(event.end, 'HH:mm')}
                        </div>
                      </button>
                    )
                  })}
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
