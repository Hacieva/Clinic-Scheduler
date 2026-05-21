import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  startOfWeek,
  endOfWeek,
  addWeeks,
  subWeeks,
  addDays,
  subDays,
  format,
  parseISO,
} from 'date-fns'
import { getDoctorAppointments } from '../../api/appointments'
import WeekCalendar from '../../components/WeekCalendar'
import Modal from '../../components/Modal'
import Badge from '../../components/Badge'

const STATUS_LABELS = {
  created: 'Создан',
  confirmed: 'Подтверждён',
  cancelled_by_admin: 'Отменён (адм.)',
  cancelled_by_patient: 'Отменён (пац.)',
  completed: 'Завершён',
  no_show: 'Не пришёл',
}

const STATUS_VARIANTS = {
  created: 'pending',
  confirmed: 'active',
  cancelled_by_admin: 'cancelled',
  cancelled_by_patient: 'cancelled',
  completed: 'inactive',
  no_show: 'inactive',
}

function toEvents(list) {
  return list.map((a) => ({
    id: a.id,
    title: a.patient_name,
    start: parseISO(a.start_at),
    end: parseISO(a.end_at),
    status: a.status,
    _raw: a,
  }))
}

function DetailRow({ label, value }) {
  return (
    <div className="flex items-start justify-between py-2 border-b border-gray-100 last:border-b-0 gap-4">
      <span className="text-sm text-gray-500 shrink-0">{label}</span>
      <span className="text-sm text-gray-900 font-medium text-right">{value ?? '—'}</span>
    </div>
  )
}

export default function SchedulePage() {
  const [view, setView] = useState('week')
  const [date, setDate] = useState(new Date())
  const [selectedEvent, setSelectedEvent] = useState(null)

  const weekStart = startOfWeek(date, { weekStartsOn: 1 })
  const weekEnd = endOfWeek(date, { weekStartsOn: 1 })
  const dateFrom = view === 'week' ? format(weekStart, 'yyyy-MM-dd') : format(date, 'yyyy-MM-dd')
  const dateTo = view === 'week' ? format(weekEnd, 'yyyy-MM-dd') : format(date, 'yyyy-MM-dd')

  const { data: rawList = [], isLoading, error } = useQuery({
    queryKey: ['doctor-schedule', dateFrom, dateTo],
    queryFn: () => getDoctorAppointments({ date_from: dateFrom, date_to: dateTo, limit: 100 }),
  })

  const handleNavigate = (direction) => {
    if (direction === 'today') {
      setDate(new Date())
      return
    }
    if (view === 'week') {
      setDate(direction === 'next' ? addWeeks(date, 1) : subWeeks(date, 1))
    } else {
      setDate(direction === 'next' ? addDays(date, 1) : subDays(date, 1))
    }
  }

  if (error) {
    const status = error?.response?.status
    const msg =
      status === 403 || status === 404
        ? 'Профиль врача не настроен. Обратитесь к администратору.'
        : 'Не удалось загрузить расписание. Попробуйте обновить страницу.'
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-sm text-gray-500">{msg}</p>
      </div>
    )
  }

  const events = toEvents(rawList)

  return (
    <div className="relative">
      {isLoading && (
        <div className="absolute inset-0 bg-white/70 z-20 flex items-center justify-center pointer-events-none">
          <span className="text-sm text-gray-400">Загрузка...</span>
        </div>
      )}

      <WeekCalendar
        events={events}
        view={view}
        date={date}
        onEventClick={setSelectedEvent}
        onNavigate={handleNavigate}
        onViewChange={setView}
      />

      <Modal
        isOpen={!!selectedEvent}
        onClose={() => setSelectedEvent(null)}
        title="Детали записи"
      >
        {selectedEvent && (
          <div>
            <DetailRow label="Пациент" value={selectedEvent._raw.patient_name} />
            <DetailRow label="Услуга" value={selectedEvent._raw.service_name} />
            <DetailRow label="Начало" value={format(selectedEvent.start, 'dd.MM.yyyy HH:mm')} />
            <DetailRow label="Окончание" value={format(selectedEvent.end, 'dd.MM.yyyy HH:mm')} />
            <div className="flex items-center justify-between py-2">
              <span className="text-sm text-gray-500">Статус</span>
              <Badge variant={STATUS_VARIANTS[selectedEvent._raw.status] ?? 'inactive'}>
                {STATUS_LABELS[selectedEvent._raw.status] ?? selectedEvent._raw.status}
              </Badge>
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}
