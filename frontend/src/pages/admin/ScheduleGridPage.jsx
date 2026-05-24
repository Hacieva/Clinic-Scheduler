import { useState, useEffect, useRef, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  addDays, subDays, format, isToday, startOfMonth, endOfMonth,
  getDay, addMonths, subMonths, isSameMonth, isSameDay,
} from 'date-fns'
import { ru } from 'date-fns/locale'
import {
  ChevronLeft, ChevronRight, Plus, Check, X, SquareCheck, UserX, Users,
} from 'lucide-react'
import toast from 'react-hot-toast'
import AppointmentGrid from '../../components/AppointmentGrid'
import Modal from '../../components/Modal'
import Badge from '../../components/Badge'
import ConfirmDialog from '../../components/ConfirmDialog'
import useBranchStore from '../../stores/branch'
import { getDoctors } from '../../api/doctors'
import { getDoctorServices } from '../../api/services'
import { getPatients } from '../../api/patients'
import {
  createAppointment,
  confirmAppointment,
  cancelAppointment,
  completeAppointment,
  noShowAppointment,
} from '../../api/appointments'

// ─── Constants ────────────────────────────────────────────────────────────────

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

const TERMINAL = new Set(['cancelled_by_admin', 'cancelled_by_patient', 'completed', 'no_show'])

const AVATAR_COLORS = [
  'bg-blue-500', 'bg-emerald-500', 'bg-violet-500',
  'bg-amber-500', 'bg-rose-500', 'bg-cyan-500',
]
function avatarBg(id) { return AVATAR_COLORS[id % AVATAR_COLORS.length] }

// ─── MiniCalendar ─────────────────────────────────────────────────────────────

function MiniCalendar({ selected, onChange }) {
  const [view, setView] = useState(() => startOfMonth(selected))

  useEffect(() => {
    if (!isSameMonth(selected, view)) {
      setView(startOfMonth(selected))
    }
  }, [selected]) // eslint-disable-line react-hooks/exhaustive-deps

  const monthEnd = endOfMonth(view)
  const startCol = (getDay(view) + 6) % 7 // Mon=0

  const cells = []
  for (let i = 0; i < startCol; i++) cells.push(null)
  for (let d = 1; d <= monthEnd.getDate(); d++) {
    cells.push(new Date(view.getFullYear(), view.getMonth(), d))
  }

  return (
    <div className="px-3 pb-3">
      <div className="flex items-center justify-between mb-2">
        <button
          onClick={() => setView((v) => subMonths(v, 1))}
          className="p-1 rounded hover:bg-gray-100 text-gray-500 transition-colors"
        >
          <ChevronLeft size={13} />
        </button>
        <span className="text-xs font-semibold text-gray-700 capitalize select-none">
          {format(view, 'LLLL yyyy', { locale: ru })}
        </span>
        <button
          onClick={() => setView((v) => addMonths(v, 1))}
          className="p-1 rounded hover:bg-gray-100 text-gray-500 transition-colors"
        >
          <ChevronRight size={13} />
        </button>
      </div>

      <div className="grid grid-cols-7 gap-0.5 text-center mb-0.5">
        {['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'].map((d) => (
          <span key={d} className="text-[10px] text-gray-400 py-0.5 font-medium">
            {d}
          </span>
        ))}
      </div>

      <div className="grid grid-cols-7 gap-0.5">
        {cells.map((day, i) =>
          day ? (
            <button
              key={i}
              onClick={() => onChange(day)}
              className={`text-[11px] rounded py-1 leading-none transition-colors ${
                isSameDay(day, selected)
                  ? 'bg-blue-600 text-white font-semibold'
                  : isToday(day)
                  ? 'bg-blue-50 text-blue-700 font-semibold'
                  : 'text-gray-700 hover:bg-gray-100'
              }`}
            >
              {day.getDate()}
            </button>
          ) : (
            <span key={i} />
          ),
        )}
      </div>
    </div>
  )
}

// ─── CreateForm ───────────────────────────────────────────────────────────────

const createSchema = z.object({
  patient_name: z.string().min(1, 'Введите ФИО пациента'),
  patient_phone: z.string().min(7, 'Введите номер телефона'),
  doctor_id: z.string().min(1, 'Выберите врача'),
  service_id: z.string().min(1, 'Выберите услугу'),
  start_at: z.string().min(1, 'Укажите дату и время'),
  patient_comment: z.string().optional(),
})

function CreateForm({ doctors, services, onDoctorChange, onSubmit, isLoading, initialDoctorId, initialStartAt }) {
  const {
    register, handleSubmit, setValue, formState: { errors },
  } = useForm({
    resolver: zodResolver(createSchema),
    defaultValues: {
      doctor_id: initialDoctorId ? String(initialDoctorId) : '',
      service_id: '',
      patient_name: '',
      patient_phone: '',
      start_at: initialStartAt ?? '',
      patient_comment: '',
    },
  })

  const { onChange: rhfDoctorChange, ...restDoctor } = register('doctor_id')
  const { ref: patientNameRef } = register('patient_name')

  const [patientSearch, setPatientSearch] = useState('')
  const [patientQuery, setPatientQuery] = useState('')
  const [showDropdown, setShowDropdown] = useState(false)
  const [selectedIndex, setSelectedIndex] = useState(-1)
  const containerRef = useRef(null)

  useEffect(() => {
    const t = setTimeout(() => setPatientQuery(patientSearch), 300)
    return () => clearTimeout(t)
  }, [patientSearch])

  const { data: patientResults = [] } = useQuery({
    queryKey: ['patient-search', patientQuery],
    queryFn: () => getPatients({ search: patientQuery, limit: 5 }),
    enabled: patientQuery.trim().length >= 2,
  })

  useEffect(() => { setSelectedIndex(-1) }, [patientResults])

  useEffect(() => {
    const handler = (e) => {
      if (containerRef.current && !containerRef.current.contains(e.target)) {
        setShowDropdown(false)
        setSelectedIndex(-1)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const capitalizeWords = (val) => val.replace(/(?:^|\s)\S/g, (c) => c.toUpperCase())

  const highlight = (text, query) => {
    if (!query || query.length < 2) return text
    const idx = text.toLowerCase().indexOf(query.toLowerCase())
    if (idx === -1) return text
    return (
      <>
        {text.slice(0, idx)}
        <mark className="bg-yellow-100 text-yellow-900 not-italic font-semibold rounded-sm">
          {text.slice(idx, idx + query.length)}
        </mark>
        {text.slice(idx + query.length)}
      </>
    )
  }

  const handleKeyDown = (e) => {
    if (!showDropdown || patientResults.length === 0) return
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setSelectedIndex((i) => Math.min(i + 1, patientResults.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setSelectedIndex((i) => Math.max(i - 1, -1))
    } else if (e.key === 'Enter' && selectedIndex >= 0) {
      e.preventDefault()
      selectPatient(patientResults[selectedIndex])
    } else if (e.key === 'Escape') {
      setShowDropdown(false)
      setSelectedIndex(-1)
    }
  }

  const selectPatient = (p) => {
    setValue('patient_name', p.full_name ?? '')
    setValue('patient_phone', p.phone ?? '')
    setPatientSearch(p.full_name ?? '')
    setShowDropdown(false)
    setSelectedIndex(-1)
  }

  useEffect(() => {
    if (initialDoctorId) onDoctorChange(String(initialDoctorId))
  }, [initialDoctorId]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div className="relative" ref={containerRef}>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            ФИО пациента <span className="text-red-500">*</span>
          </label>
          <input
            ref={patientNameRef}
            type="text"
            value={patientSearch}
            onChange={(e) => {
              const val = capitalizeWords(e.target.value)
              setPatientSearch(val)
              setValue('patient_name', val)
              setShowDropdown(true)
            }}
            onFocus={() => patientQuery.trim().length >= 2 && setShowDropdown(true)}
            onKeyDown={handleKeyDown}
            placeholder="Введите ФИО или телефон…"
            autoComplete="off"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          {showDropdown && patientResults.length > 0 && (
            <div className="absolute z-50 mt-1 w-full bg-white border border-gray-200 rounded-lg shadow-lg overflow-hidden">
              {patientResults.map((p, idx) => (
                <button
                  key={p.id}
                  type="button"
                  onMouseDown={() => selectPatient(p)}
                  className={`w-full text-left px-3 py-2.5 text-sm border-b border-gray-100 last:border-b-0 transition-colors ${
                    idx === selectedIndex ? 'bg-blue-50' : 'hover:bg-gray-50'
                  }`}
                >
                  <span className="font-medium text-gray-900">
                    {highlight(p.full_name, patientQuery)}
                  </span>
                  {p.phone && <span className="text-gray-400 ml-2 text-xs">{p.phone}</span>}
                </button>
              ))}
            </div>
          )}
          {errors.patient_name && (
            <p className="mt-1 text-xs text-red-600">{errors.patient_name.message}</p>
          )}
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Телефон <span className="text-red-500">*</span>
          </label>
          <input
            type="tel"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            {...register('patient_phone')}
          />
          {errors.patient_phone && (
            <p className="mt-1 text-xs text-red-600">{errors.patient_phone.message}</p>
          )}
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Врач <span className="text-red-500">*</span>
        </label>
        <select
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          {...restDoctor}
          onChange={(e) => {
            rhfDoctorChange(e)
            setValue('service_id', '')
            onDoctorChange(e.target.value)
          }}
        >
          <option value="">Выберите врача</option>
          {doctors.map((d) => (
            <option key={d.id} value={d.id}>
              {[d.last_name, d.first_name, d.middle_name].filter(Boolean).join(' ')}
            </option>
          ))}
        </select>
        {errors.doctor_id && (
          <p className="mt-1 text-xs text-red-600">{errors.doctor_id.message}</p>
        )}
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Услуга <span className="text-red-500">*</span>
        </label>
        <select
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-50 disabled:text-gray-400"
          {...register('service_id')}
          disabled={services.length === 0}
        >
          <option value="">
            {services.length === 0 ? 'Сначала выберите врача' : 'Выберите услугу'}
          </option>
          {services.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name} ({s.duration_minutes} мин)
            </option>
          ))}
        </select>
        {errors.service_id && (
          <p className="mt-1 text-xs text-red-600">{errors.service_id.message}</p>
        )}
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Дата и время <span className="text-red-500">*</span>
        </label>
        <input
          type="datetime-local"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          {...register('start_at')}
        />
        {errors.start_at && (
          <p className="mt-1 text-xs text-red-600">{errors.start_at.message}</p>
        )}
        <p className="mt-1 text-xs text-gray-400">Длительность рассчитывается по услуге</p>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Комментарий</label>
        <input
          type="text"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          {...register('patient_comment')}
        />
      </div>

      <div className="flex justify-end pt-1">
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-60 transition-colors"
        >
          {isLoading ? 'Создание…' : 'Создать запись'}
        </button>
      </div>
    </form>
  )
}

// ─── CancelModal ──────────────────────────────────────────────────────────────

function CancelModal({ target, onClose, onConfirm, isLoading }) {
  const [comment, setComment] = useState('')
  return (
    <Modal isOpen={!!target} onClose={onClose} title="Отменить запись">
      {target && (
        <>
          <p className="text-sm text-gray-600 mb-4">
            Запись пациента <span className="font-medium">{target.patient_name}</span> будет отменена.
          </p>
          <input
            type="text"
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            placeholder="Причина отмены (необязательно)"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mb-4 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <div className="flex justify-end gap-3">
            <button
              onClick={onClose}
              disabled={isLoading}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-60"
            >
              Назад
            </button>
            <button
              onClick={() => onConfirm(comment || undefined)}
              disabled={isLoading}
              className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 disabled:opacity-60"
            >
              {isLoading ? 'Отмена…' : 'Отменить запись'}
            </button>
          </div>
        </>
      )}
    </Modal>
  )
}

// ─── EventDetailModal ─────────────────────────────────────────────────────────

function EventRow({ label, value }) {
  return (
    <div className="flex items-start justify-between py-2 border-b border-gray-100 last:border-b-0 gap-4">
      <span className="text-sm text-gray-500 shrink-0">{label}</span>
      <span className="text-sm text-gray-900 font-medium text-right">{value ?? '—'}</span>
    </div>
  )
}

function EventDetailModal({ appt, onClose, onConfirm, onCancel, onComplete, onNoShow }) {
  if (!appt) return null
  const terminal = TERMINAL.has(appt.status)

  return (
    <Modal isOpen={!!appt} onClose={onClose} title="Запись">
      <EventRow label="Пациент" value={appt.patient_name} />
      <EventRow label="Врач" value={appt.doctor_full_name} />
      <EventRow label="Услуга" value={appt.service_name} />
      <EventRow
        label="Время"
        value={`${format(new Date(appt.start_at), 'dd.MM.yyyy HH:mm')} – ${format(new Date(appt.end_at), 'HH:mm')}`}
      />
      <div className="flex items-center justify-between py-2 border-b border-gray-100">
        <span className="text-sm text-gray-500">Статус</span>
        <Badge variant={STATUS_VARIANTS[appt.status] ?? 'inactive'}>
          {STATUS_LABELS[appt.status] ?? appt.status}
        </Badge>
      </div>

      {!terminal && (
        <div className="flex items-center justify-end gap-2 pt-4">
          {appt.status === 'created' && (
            <button
              onClick={() => onConfirm(appt)}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-white bg-green-600 rounded-lg hover:bg-green-700 transition-colors"
            >
              <Check size={14} /> Подтвердить
            </button>
          )}
          {appt.status === 'confirmed' && (
            <>
              <button
                onClick={() => onComplete(appt)}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
              >
                <SquareCheck size={14} /> Завершить
              </button>
              <button
                onClick={() => onNoShow(appt)}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-white bg-amber-500 rounded-lg hover:bg-amber-600 transition-colors"
              >
                <UserX size={14} /> Не пришёл
              </button>
            </>
          )}
          <button
            onClick={() => onCancel(appt)}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
          >
            <X size={14} /> Отменить
          </button>
        </div>
      )}
    </Modal>
  )
}

// ─── ScheduleGridPage ─────────────────────────────────────────────────────────

export default function ScheduleGridPage() {
  const qc = useQueryClient()

  const [date, setDate] = useState(new Date())
  const activeBranchId = useBranchStore((s) => s.activeBranchId)

  // Left panel filter state: null = all visible
  const [checkedDoctorIds, setCheckedDoctorIds] = useState(null)

  // Modal state
  const [selectedAppt, setSelectedAppt] = useState(null)
  const [createModal, setCreateModal] = useState(null)
  const [cancelTarget, setCancelTarget] = useState(null)
  const [simpleAction, setSimpleAction] = useState(null)
  const [createDoctorId, setCreateDoctorId] = useState('')

  // ── Doctor list for left panel (same query key → shared cache with AppointmentGrid) ──
  const { data: allDoctors = [] } = useQuery({
    queryKey: ['grid-doctors', activeBranchId ?? null],
    queryFn: () => getDoctors(activeBranchId ? { branch_id: activeBranchId } : undefined),
  })
  const activeDoctors = useMemo(() => allDoctors.filter((d) => d.is_active), [allDoctors])

  // Visible doctor IDs: null = all, otherwise the checked set
  const visibleDoctorIds = useMemo(() => {
    if (checkedDoctorIds === null) return null
    return checkedDoctorIds.length === 0 ? activeDoctors.map((d) => d.id) : checkedDoctorIds
  }, [checkedDoctorIds, activeDoctors])

  const toggleDoctor = (id) => {
    setCheckedDoctorIds((prev) => {
      const set = prev ?? activeDoctors.map((d) => d.id)
      return set.includes(id) ? set.filter((x) => x !== id) : [...set, id]
    })
  }

  const allChecked = checkedDoctorIds === null || checkedDoctorIds.length === activeDoctors.length
  const toggleAll = () => setCheckedDoctorIds(allChecked ? [] : null)

  // ── Services for create form ──
  const { data: createServices = [] } = useQuery({
    queryKey: ['services', createDoctorId],
    queryFn: () => getDoctorServices(Number(createDoctorId)),
    enabled: !!createDoctorId,
  })

  const invalidateGrid = () => {
    qc.invalidateQueries({ queryKey: ['grid-appointments'] })
    qc.invalidateQueries({ queryKey: ['appointments'] })
  }

  // ── Mutations ──
  const createMut = useMutation({
    mutationFn: createAppointment,
    onSuccess: () => {
      invalidateGrid()
      setCreateModal(null)
      setCreateDoctorId('')
      toast.success('Запись создана')
    },
    onError: (err) => {
      const msg = err?.response?.data?.error ?? ''
      const status = err?.response?.status
      if (status === 409) toast.error('Время уже занято. Выберите другой слот.')
      else if (status === 422 && msg.includes('inactive')) toast.error('Врач неактивен.')
      else if (status === 422 && msg.includes('outside')) toast.error('Время вне рабочих часов врача.')
      else toast.error('Не удалось создать запись.')
    },
  })

  const confirmMut = useMutation({
    mutationFn: (id) => confirmAppointment(id),
    onSuccess: () => { invalidateGrid(); setSimpleAction(null); setSelectedAppt(null); toast.success('Запись подтверждена') },
    onError: () => toast.error('Не удалось подтвердить запись.'),
  })

  const cancelMut = useMutation({
    mutationFn: ({ id, comment }) => cancelAppointment(id, comment),
    onSuccess: () => { invalidateGrid(); setCancelTarget(null); setSelectedAppt(null); toast.success('Запись отменена') },
    onError: () => toast.error('Не удалось отменить запись.'),
  })

  const completeMut = useMutation({
    mutationFn: (id) => completeAppointment(id),
    onSuccess: () => { invalidateGrid(); setSimpleAction(null); setSelectedAppt(null); toast.success('Запись завершена') },
    onError: () => toast.error('Не удалось завершить запись.'),
  })

  const noShowMut = useMutation({
    mutationFn: (id) => noShowAppointment(id),
    onSuccess: () => { invalidateGrid(); setSimpleAction(null); setSelectedAppt(null); toast.success('Отмечено: не пришёл') },
    onError: () => toast.error('Не удалось обновить статус.'),
  })

  // ── Handlers ──
  const handleSlotClick = (doctorId, startTime) => {
    setCreateModal({ doctorId, startAt: `${format(date, 'yyyy-MM-dd')}T${startTime}` })
    setCreateDoctorId(String(doctorId))
  }

  const handleEventClick = (appt) => setSelectedAppt(appt)
  const handleConfirmAppt = (appt) => setSimpleAction({ appt, action: 'confirm', title: 'Подтвердить запись', label: 'Подтвердить' })
  const handleCompleteAppt = (appt) => setSimpleAction({ appt, action: 'complete', title: 'Завершить запись', label: 'Завершить' })
  const handleNoShowAppt = (appt) => setSimpleAction({ appt, action: 'noShow', title: 'Отметить «Не пришёл»', label: 'Отметить' })

  const handleSimpleConfirm = () => {
    if (!simpleAction) return
    const id = simpleAction.appt.id
    if (simpleAction.action === 'confirm') confirmMut.mutate(id)
    else if (simpleAction.action === 'complete') completeMut.mutate(id)
    else if (simpleAction.action === 'noShow') noShowMut.mutate(id)
  }

  const handleCreateSubmit = (data) => {
    createMut.mutate({
      patient_name: data.patient_name,
      patient_phone: data.patient_phone,
      doctor_id: Number(data.doctor_id),
      service_id: Number(data.service_id),
      start_at: new Date(data.start_at).toISOString(),
      ...(data.patient_comment ? { patient_comment: data.patient_comment } : {}),
    })
  }

  const simpleActionPending = confirmMut.isPending || completeMut.isPending || noShowMut.isPending
  const dateLabel = format(date, 'EEEE, d MMMM yyyy', { locale: ru })
  const viewingToday = isToday(date)

  return (
    <div className="flex h-full bg-gray-50 overflow-hidden">

      {/* ── Left panel ── */}
      <aside className="w-52 shrink-0 bg-white border-r border-gray-200 flex flex-col overflow-y-auto">

        {/* Today button + date label */}
        <div className="p-3 pb-2 border-b border-gray-100">
          <button
            onClick={() => setDate(new Date())}
            className={`w-full px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
              viewingToday
                ? 'bg-blue-600 text-white'
                : 'border border-gray-300 text-gray-700 hover:bg-gray-50'
            }`}
          >
            Сегодня
          </button>
        </div>

        {/* Mini calendar */}
        <div className="pt-2">
          <MiniCalendar selected={date} onChange={setDate} />
        </div>

        {/* Doctor filter */}
        {activeDoctors.length > 0 && (
          <div className="border-t border-gray-100 pt-3 flex-1">
            <div className="px-3 mb-2 flex items-center justify-between">
              <span className="text-[11px] font-semibold text-gray-500 uppercase tracking-wide">
                Сотрудники
              </span>
              <button
                onClick={toggleAll}
                className="text-[10px] text-blue-600 hover:text-blue-800 font-medium"
              >
                {allChecked ? 'Снять все' : 'Все'}
              </button>
            </div>
            <div className="px-2 space-y-0.5">
              {activeDoctors.map((d) => {
                const checked =
                  checkedDoctorIds === null || checkedDoctorIds.includes(d.id)
                const name = [d.last_name, d.first_name].filter(Boolean).join(' ')
                const initial = (d.last_name ?? d.first_name ?? '?')[0]
                return (
                  <label
                    key={d.id}
                    className="flex items-center gap-2 px-1 py-1 rounded-md cursor-pointer hover:bg-gray-50 select-none transition-colors"
                  >
                    <input
                      type="checkbox"
                      className="rounded shrink-0 accent-blue-600"
                      checked={checked}
                      onChange={() => toggleDoctor(d.id)}
                    />
                    <div
                      className={`w-5 h-5 rounded-full ${avatarBg(d.id)} flex items-center justify-center text-white text-[9px] font-bold shrink-0`}
                    >
                      {initial}
                    </div>
                    <span className="text-xs text-gray-700 truncate">{name}</span>
                  </label>
                )
              })}
            </div>
          </div>
        )}
      </aside>

      {/* ── Main area ── */}
      <div className="flex flex-col flex-1 min-w-0 overflow-hidden">

        {/* Toolbar */}
        <div className="shrink-0 flex items-center gap-3 px-4 py-2.5 bg-white border-b border-gray-200">
          <div className="flex items-center gap-1">
            <button
              onClick={() => setDate((d) => subDays(d, 1))}
              className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-500 transition-colors"
            >
              <ChevronLeft size={18} />
            </button>
            <button
              onClick={() => setDate((d) => addDays(d, 1))}
              className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-500 transition-colors"
            >
              <ChevronRight size={18} />
            </button>
          </div>

          <span className="text-sm font-medium text-gray-900 capitalize hidden sm:block">
            {dateLabel}
          </span>

          {/* Filter indicator */}
          {checkedDoctorIds !== null && checkedDoctorIds.length < activeDoctors.length && (
            <span className="text-xs text-blue-700 bg-blue-50 border border-blue-200 px-2 py-0.5 rounded-full font-medium">
              {checkedDoctorIds.length} из {activeDoctors.length}
            </span>
          )}

          <div className="flex-1" />

          <button
            onClick={() => setCreateModal({ doctorId: null, startAt: null })}
            className="flex items-center gap-2 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
          >
            <Plus size={15} />
            Новая запись
          </button>
        </div>

        {/* Grid */}
        <AppointmentGrid
          date={date}
          branchId={activeBranchId ?? undefined}
          onEventClick={handleEventClick}
          onSlotClick={handleSlotClick}
          visibleDoctorIds={visibleDoctorIds}
        />

        {/* Waiting patients placeholder */}
        <div className="shrink-0 border-t border-gray-200 bg-white px-4 py-2 flex items-center gap-2">
          <Users size={13} className="text-gray-400 shrink-0" />
          <span className="text-xs text-gray-400">Ожидающие пациенты — появится в v0.3</span>
        </div>
      </div>

      {/* ── Modals ── */}

      <EventDetailModal
        appt={selectedAppt}
        onClose={() => setSelectedAppt(null)}
        onConfirm={handleConfirmAppt}
        onCancel={(appt) => { setSelectedAppt(null); setCancelTarget(appt) }}
        onComplete={handleCompleteAppt}
        onNoShow={handleNoShowAppt}
      />

      <Modal
        isOpen={!!createModal}
        onClose={() => { setCreateModal(null); setCreateDoctorId('') }}
        title="Новая запись"
        maxWidth="max-w-lg"
      >
        {createModal && (
          <CreateForm
            doctors={activeDoctors}
            services={createServices}
            onDoctorChange={setCreateDoctorId}
            onSubmit={handleCreateSubmit}
            isLoading={createMut.isPending}
            initialDoctorId={createModal.doctorId}
            initialStartAt={createModal.startAt}
          />
        )}
      </Modal>

      <CancelModal
        target={cancelTarget}
        onClose={() => setCancelTarget(null)}
        onConfirm={(comment) => cancelMut.mutate({ id: cancelTarget.id, comment })}
        isLoading={cancelMut.isPending}
      />

      <ConfirmDialog
        isOpen={!!simpleAction}
        onClose={() => setSimpleAction(null)}
        onConfirm={handleSimpleConfirm}
        title={simpleAction?.title ?? ''}
        message={simpleAction ? `Пациент: ${simpleAction.appt.patient_name}` : ''}
        confirmLabel={simpleAction?.label ?? 'Подтвердить'}
        confirmVariant="primary"
        isLoading={simpleActionPending}
      />
    </div>
  )
}
