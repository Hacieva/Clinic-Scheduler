import { useState, useEffect, useRef, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient, useQueries } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  addDays, subDays, format, isToday, startOfMonth, endOfMonth,
  getDay, addMonths, subMonths, isSameMonth, isSameDay, parseISO,
} from 'date-fns'
import { ru } from 'date-fns/locale'
import {
  ChevronLeft, ChevronRight, Plus, Check, X, SquareCheck, UserX, Users, Search,
} from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import toast from 'react-hot-toast'
import AppointmentGrid from '../../components/AppointmentGrid'
import Modal from '../../components/Modal'
import Badge from '../../components/Badge'
import ConfirmDialog from '../../components/ConfirmDialog'
import useBranchStore from '../../stores/branch'
import { getDoctors } from '../../api/doctors'
import { getAllServices, getAssignedServices } from '../../api/services'
import { getPatients, createPatient as apiCreatePatient } from '../../api/patients'
import { getWorkingHours } from '../../api/schedule'
import { getBranches } from '../../api/branches'
import {
  getAppointments,
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
  patient_name:    z.string().min(1, 'Введите ФИО пациента'),
  patient_phone:   z.string().min(7, 'Введите номер телефона'),
  doctor_id:       z.string().min(1, 'Выберите врача'),
  service_id:      z.string().min(1, 'Выберите услугу'),
  start_at:        z.string().min(1, 'Укажите дату и время'),
  patient_comment: z.string().optional(),
})

function CreateForm({ doctors, onSubmit, isLoading, initialDoctorId, initialStartAt, onClose }) {
  const navigate = useNavigate()
  const qc       = useQueryClient()

  const { register, handleSubmit, setValue, watch, formState: { errors } } = useForm({
    resolver: zodResolver(createSchema),
    defaultValues: {
      doctor_id:       initialDoctorId ? String(initialDoctorId) : '',
      service_id:      '',
      patient_name:    '',
      patient_phone:   '',
      start_at:        initialStartAt ?? '',
      patient_comment: '',
    },
  })

  const watchedDoctorId = watch('doctor_id')

  const { data: assignedServices = [], isFetching: servicesFetching } = useQuery({
    queryKey: ['assigned-services', watchedDoctorId],
    queryFn:  () => getAssignedServices(Number(watchedDoctorId)),
    enabled:  !!watchedDoctorId,
  })

  const [patientSearch, setPatientSearch] = useState('')
  const [patientQuery,  setPatientQuery]  = useState('')
  const [showDropdown,  setShowDropdown]  = useState(false)
  const [selectedIndex, setSelectedIndex] = useState(-1)
  const containerRef = useRef(null)

  useEffect(() => {
    const t = setTimeout(() => setPatientQuery(patientSearch), 300)
    return () => clearTimeout(t)
  }, [patientSearch])

  const { data: patientResults = [] } = useQuery({
    queryKey: ['patient-search', patientQuery],
    queryFn:  () => getPatients({ search: patientQuery, limit: 5 }),
    enabled:  patientQuery.trim().length >= 2,
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

  const [showNewPatient, setShowNewPatient] = useState(false)
  const [newName,  setNewName]  = useState('')
  const [newPhone, setNewPhone] = useState('')

  const newPatientMut = useMutation({
    mutationFn: () => apiCreatePatient({ full_name: newName, phone: newPhone }),
    onSuccess: (patient) => {
      setValue('patient_name',  patient.full_name ?? '')
      setValue('patient_phone', patient.phone ?? '')
      setPatientSearch(patient.full_name ?? '')
      setShowNewPatient(false)
      setNewName('')
      setNewPhone('')
      qc.invalidateQueries({ queryKey: ['patient-search'] })
      toast.success('Пациент создан')
    },
    onError: () => toast.error('Не удалось создать пациента'),
  })

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
    setValue('patient_name',  p.full_name ?? '')
    setValue('patient_phone', p.phone ?? '')
    setPatientSearch(p.full_name ?? '')
    setShowDropdown(false)
    setSelectedIndex(-1)
  }

  const { ref: patientNameRef } = register('patient_name')
  const { onChange: rhfDoctorChange, ...restDoctor } = register('doctor_id')

  const selectedDoctor = doctors.find((d) => d.id === Number(initialDoctorId))
  const doctorDisplayName = selectedDoctor
    ? [selectedDoctor.last_name, selectedDoctor.first_name, selectedDoctor.middle_name]
        .filter(Boolean).join(' ')
    : ''

  const noServicesAndSelected = !!watchedDoctorId && !servicesFetching && assignedServices.length === 0

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">

      {/* ── Patient ── */}
      <div>
        <div className="flex items-center justify-between mb-1">
          <label className="text-sm font-medium text-gray-700">
            Пациент <span className="text-red-500">*</span>
          </label>
          {!showNewPatient && (
            <button
              type="button"
              onClick={() => setShowNewPatient(true)}
              className="text-xs text-blue-600 hover:text-blue-800 font-medium"
            >
              + Новый пациент
            </button>
          )}
        </div>

        {showNewPatient ? (
          <div className="border border-blue-200 rounded-lg p-3 bg-blue-50 space-y-2">
            <p className="text-xs font-semibold text-blue-800">Создать нового пациента</p>
            <input
              type="text"
              value={newName}
              onChange={(e) => setNewName(capitalizeWords(e.target.value))}
              placeholder="ФИО *"
              className="w-full border border-gray-300 rounded-md px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              type="tel"
              value={newPhone}
              onChange={(e) => setNewPhone(e.target.value)}
              placeholder="Телефон *"
              className="w-full border border-gray-300 rounded-md px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <div className="flex gap-2 pt-1">
              <button
                type="button"
                onClick={() => newPatientMut.mutate()}
                disabled={!newName.trim() || !newPhone.trim() || newPatientMut.isPending}
                className="flex-1 px-3 py-1.5 text-xs font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-60 transition-colors"
              >
                {newPatientMut.isPending ? 'Создание…' : 'Создать и выбрать'}
              </button>
              <button
                type="button"
                onClick={() => { setShowNewPatient(false); setNewName(''); setNewPhone('') }}
                className="px-3 py-1.5 text-xs text-gray-500 hover:text-gray-900"
              >
                Отмена
              </button>
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-3">
            <div className="relative" ref={containerRef}>
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
            </div>
            <div>
              <input
                type="tel"
                placeholder="Телефон *"
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                {...register('patient_phone')}
              />
              {errors.patient_phone && (
                <p className="mt-1 text-xs text-red-600">{errors.patient_phone.message}</p>
              )}
            </div>
          </div>
        )}
        {!showNewPatient && errors.patient_name && (
          <p className="mt-1 text-xs text-red-600">{errors.patient_name.message}</p>
        )}
      </div>

      {/* ── Doctor ── */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Врач <span className="text-red-500">*</span>
        </label>
        {initialDoctorId ? (
          <>
            <div className="border border-gray-200 bg-gray-50 rounded-lg px-3 py-2 text-sm text-gray-700 font-medium">
              {doctorDisplayName || 'Врач не найден'}
            </div>
            <input type="hidden" {...restDoctor} />
          </>
        ) : (
          <select
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            {...restDoctor}
            onChange={(e) => {
              rhfDoctorChange(e)
              setValue('service_id', '')
            }}
          >
            <option value="">Выберите врача</option>
            {doctors.map((d) => (
              <option key={d.id} value={d.id}>
                {[d.last_name, d.first_name, d.middle_name].filter(Boolean).join(' ')}
              </option>
            ))}
          </select>
        )}
        {errors.doctor_id && (
          <p className="mt-1 text-xs text-red-600">{errors.doctor_id.message}</p>
        )}
      </div>

      {/* ── Service ── */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Услуга <span className="text-red-500">*</span>
        </label>
        {noServicesAndSelected ? (
          <div className="flex items-center justify-between gap-3 p-3 border border-amber-200 bg-amber-50 rounded-lg">
            <span className="text-sm text-amber-800">У врача нет привязанных услуг.</span>
            <button
              type="button"
              onClick={() => { onClose?.(); navigate(`/admin/doctors/${watchedDoctorId}`) }}
              className="shrink-0 text-xs font-medium text-blue-600 hover:text-blue-800 underline"
            >
              Настроить услуги
            </button>
          </div>
        ) : (
          <select
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-50 disabled:text-gray-400"
            {...register('service_id')}
            disabled={!watchedDoctorId || servicesFetching}
          >
            <option value="">
              {!watchedDoctorId ? 'Сначала выберите врача' : servicesFetching ? 'Загрузка…' : 'Выберите услугу'}
            </option>
            {assignedServices.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name} ({s.duration_minutes} мин)
              </option>
            ))}
          </select>
        )}
        {!noServicesAndSelected && errors.service_id && (
          <p className="mt-1 text-xs text-red-600">{errors.service_id.message}</p>
        )}
      </div>

      {/* ── Date/time ── */}
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

      {/* ── Comment ── */}
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
          disabled={isLoading || noServicesAndSelected}
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
  const activeBranchId    = useBranchStore((s) => s.activeBranchId)
  const setActiveBranchId = useBranchStore((s) => s.setActiveBranchId)

  // Filter state
  const [specFilter,       setSpecFilter]       = useState('')
  const [doctorNameFilter, setDoctorNameFilter] = useState('')
  const [serviceFilter,    setServiceFilter]    = useState('')

  // Modal state
  const [selectedAppt,   setSelectedAppt]   = useState(null)
  const [createModal,    setCreateModal]    = useState(null)
  const [cancelTarget,   setCancelTarget]   = useState(null)
  const [simpleAction,   setSimpleAction]   = useState(null)

  // ── Doctor list (shared cache with AppointmentGrid) ──
  const { data: allDoctors = [] } = useQuery({
    queryKey: ['grid-doctors', activeBranchId ?? null],
    queryFn: () => getDoctors(activeBranchId ? { branch_id: activeBranchId } : undefined),
  })
  const activeDoctors = useMemo(() => allDoctors.filter((d) => d.is_active), [allDoctors])

  // ── Branches ──
  const { data: branches = [] } = useQuery({
    queryKey: ['branches'],
    queryFn: getBranches,
    staleTime: 10 * 60 * 1000,
  })

  // ── Catalog services for filter dropdown ──
  const { data: catalogServices = [] } = useQuery({
    queryKey: ['catalog-services'],
    queryFn: () => getAllServices(true),
    staleTime: 5 * 60 * 1000,
  })

  // ── Working hours per doctor ──
  const workingHoursQueries = useQueries({
    queries: activeDoctors.map((d) => ({
      queryKey: ['doctor-working-hours', d.id],
      queryFn: () => getWorkingHours(d.id),
      staleTime: 5 * 60 * 1000,
    })),
  })

  // ── Today's appointments (shared cache key with AppointmentGrid) ──
  const dateStr = format(date, 'yyyy-MM-dd')
  const { data: todayAppointments = [] } = useQuery({
    queryKey: ['grid-appointments', dateStr, activeBranchId ?? null],
    queryFn: () =>
      getAppointments({
        date_from: dateStr,
        date_to:   dateStr,
        limit:     200,
        ...(activeBranchId ? { branch_id: activeBranchId } : {}),
      }),
  })

  // ── Derived: unique specializations ──
  const uniqueSpecs = useMemo(() => {
    const set = new Set()
    activeDoctors.forEach((d) => (d.directions ?? []).forEach((dir) => set.add(dir.name)))
    return [...set].sort((a, b) => a.localeCompare(b, 'ru'))
  }, [activeDoctors])

  // ── Derived: selected day of week (DB: 1=Mon … 7=Sun) ──
  const selectedDow = useMemo(() => {
    const j = date.getDay()
    return j === 0 ? 7 : j
  }, [date])

  // ── Derived: working hours map keyed by doctor id ──
  const workingHoursMap = useMemo(() => {
    const map = new Map()
    activeDoctors.forEach((d, i) => {
      const q = workingHoursQueries[i]
      if (!q?.data) return
      const wh = q.data.find((w) => w.day_of_week === selectedDow)
      if (!wh) {
        map.set(d.id, null)
      } else {
        const st = parseISO(wh.start_time)
        const et = parseISO(wh.end_time)
        map.set(d.id, {
          startMin: st.getUTCHours() * 60 + st.getUTCMinutes(),
          endMin:   et.getUTCHours() * 60 + et.getUTCMinutes(),
        })
      }
    })
    return map
  }, [activeDoctors, workingHoursQueries, selectedDow])

  const workingHoursAllLoaded =
    workingHoursQueries.length === 0 || workingHoursQueries.every((q) => q.isFetched)

  // ── Derived: visible doctor IDs (filters applied) ──
  const visibleDoctorIds = useMemo(() => {
    if (!workingHoursAllLoaded) return null

    let ids = activeDoctors.map((d) => d.id)

    // Hide doctors who don't work on the selected day
    ids = ids.filter((id) => {
      if (!workingHoursMap.has(id)) return true   // no data loaded → keep
      return workingHoursMap.get(id) !== null
    })

    // Specialty filter
    if (specFilter) {
      ids = ids.filter((id) => {
        const d = activeDoctors.find((x) => x.id === id)
        return d && (d.directions ?? []).some((dir) => dir.name === specFilter)
      })
    }

    // Doctor name search
    if (doctorNameFilter.trim()) {
      const q = doctorNameFilter.trim().toLowerCase()
      ids = ids.filter((id) => {
        const d = activeDoctors.find((x) => x.id === id)
        return (
          d &&
          [d.last_name, d.first_name, d.middle_name]
            .filter(Boolean)
            .join(' ')
            .toLowerCase()
            .includes(q)
        )
      })
    }

    // Service filter — match doctors who have today's appointments for that service
    if (serviceFilter) {
      const withSvc = new Set(
        todayAppointments
          .filter((a) => String(a.service_id) === serviceFilter)
          .map((a) => a.doctor_id),
      )
      if (withSvc.size > 0) {
        ids = ids.filter((id) => withSvc.has(id))
      }
    }

    return ids
  }, [
    activeDoctors, workingHoursAllLoaded, workingHoursMap,
    specFilter, doctorNameFilter, serviceFilter, todayAppointments,
  ])

  // ── Derived: queue stats ──
  const queueStats = useMemo(() => ({
    total:   todayAppointments.filter(
      (a) => !['cancelled_by_admin', 'cancelled_by_patient'].includes(a.status),
    ).length,
    waiting: todayAppointments.filter(
      (a) => a.status === 'created' || a.status === 'confirmed',
    ).length,
    done: todayAppointments.filter((a) => a.status === 'completed').length,
  }), [todayAppointments])

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
  }

  const handleEventClick      = (appt) => setSelectedAppt(appt)
  const handleConfirmAppt     = (appt) => setSimpleAction({ appt, action: 'confirm',  title: 'Подтвердить запись',       label: 'Подтвердить' })
  const handleCompleteAppt    = (appt) => setSimpleAction({ appt, action: 'complete', title: 'Завершить запись',         label: 'Завершить' })
  const handleNoShowAppt      = (appt) => setSimpleAction({ appt, action: 'noShow',   title: 'Отметить «Не пришёл»',     label: 'Отметить' })

  const handleSimpleConfirm = () => {
    if (!simpleAction) return
    const id = simpleAction.appt.id
    if (simpleAction.action === 'confirm')  confirmMut.mutate(id)
    else if (simpleAction.action === 'complete') completeMut.mutate(id)
    else if (simpleAction.action === 'noShow')   noShowMut.mutate(id)
  }

  const handleCreateSubmit = (data) => {
    createMut.mutate({
      patient_name:  data.patient_name,
      patient_phone: data.patient_phone,
      doctor_id:     Number(data.doctor_id),
      service_id:    Number(data.service_id),
      start_at:      `${data.start_at.slice(0, 16)}:00.000Z`,
      ...(data.patient_comment ? { patient_comment: data.patient_comment } : {}),
    })
  }

  const simpleActionPending = confirmMut.isPending || completeMut.isPending || noShowMut.isPending
  const dateLabel   = format(date, 'EEEE, d MMMM yyyy', { locale: ru })
  const viewingToday = isToday(date)

  const anyFilterActive = specFilter || doctorNameFilter || serviceFilter

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

        {/* ── Filters ── */}
        <div className="border-t border-gray-100 pt-2 px-3 pb-3 space-y-2.5">
          <p className="text-[10px] font-semibold text-gray-400 uppercase tracking-wide">
            Фильтры
          </p>

          {/* Specialty */}
          <div>
            <label className="text-[10px] text-gray-500 block mb-1">Специализация</label>
            <select
              value={specFilter}
              onChange={(e) => setSpecFilter(e.target.value)}
              className="w-full text-xs border border-gray-200 rounded-md px-2 py-1.5 text-gray-700 bg-white focus:outline-none focus:ring-1 focus:ring-blue-400"
            >
              <option value="">Все</option>
              {uniqueSpecs.map((s) => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
          </div>

          {/* Doctor name search */}
          <div>
            <label className="text-[10px] text-gray-500 block mb-1">Врач</label>
            <div className="relative">
              <Search size={11} className="absolute left-2 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none" />
              <input
                type="text"
                value={doctorNameFilter}
                onChange={(e) => setDoctorNameFilter(e.target.value)}
                placeholder="Поиск по ФИО…"
                className="w-full pl-6 pr-2 py-1.5 text-xs border border-gray-200 rounded-md focus:outline-none focus:ring-1 focus:ring-blue-400"
              />
            </div>
          </div>

          {/* Branch — only when multiple exist */}
          {branches.length > 1 && (
            <div>
              <label className="text-[10px] text-gray-500 block mb-1">Филиал</label>
              <select
                value={activeBranchId ?? ''}
                onChange={(e) => setActiveBranchId(e.target.value ? Number(e.target.value) : null)}
                className="w-full text-xs border border-gray-200 rounded-md px-2 py-1.5 text-gray-700 bg-white focus:outline-none focus:ring-1 focus:ring-blue-400"
              >
                <option value="">Все филиалы</option>
                {branches.map((b) => (
                  <option key={b.id} value={b.id}>{b.name}</option>
                ))}
              </select>
            </div>
          )}

          {/* Service */}
          {catalogServices.length > 0 && (
            <div>
              <label className="text-[10px] text-gray-500 block mb-1">Услуга</label>
              <select
                value={serviceFilter}
                onChange={(e) => setServiceFilter(e.target.value)}
                className="w-full text-xs border border-gray-200 rounded-md px-2 py-1.5 text-gray-700 bg-white focus:outline-none focus:ring-1 focus:ring-blue-400"
              >
                <option value="">Все услуги</option>
                {catalogServices.map((s) => (
                  <option key={s.id} value={String(s.id)}>{s.name}</option>
                ))}
              </select>
            </div>
          )}

          {/* Reset link */}
          {anyFilterActive && (
            <button
              onClick={() => { setSpecFilter(''); setDoctorNameFilter(''); setServiceFilter('') }}
              className="text-[10px] text-blue-600 hover:text-blue-800 font-medium w-full text-left"
            >
              × Сбросить фильтры
            </button>
          )}
        </div>
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
          {visibleDoctorIds !== null && visibleDoctorIds.length < activeDoctors.length && (
            <span className="text-xs text-blue-700 bg-blue-50 border border-blue-200 px-2 py-0.5 rounded-full font-medium">
              {visibleDoctorIds.length} из {activeDoctors.length}
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
          workingHoursMap={workingHoursMap}
        />

        {/* ── Queue stats panel ── */}
        <div className="shrink-0 border-t border-gray-200 bg-white px-4 py-2 flex items-center gap-3">
          <Users size={13} className="text-gray-400 shrink-0" />
          <span className="text-xs text-gray-600 font-medium">
            {format(date, 'd MMM', { locale: ru })}:
          </span>
          <span className="text-xs text-gray-700">{queueStats.total} зап.</span>
          {queueStats.waiting > 0 && (
            <span className="text-xs bg-amber-50 text-amber-700 px-2 py-0.5 rounded-full font-medium border border-amber-100">
              {queueStats.waiting} ожидают
            </span>
          )}
          {queueStats.done > 0 && (
            <span className="text-xs bg-emerald-50 text-emerald-700 px-2 py-0.5 rounded-full font-medium border border-emerald-100">
              {queueStats.done} завершено
            </span>
          )}
          {queueStats.total === 0 && (
            <span className="text-xs text-gray-400">Записей нет</span>
          )}
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
        onClose={() => setCreateModal(null)}
        title="Новая запись"
        maxWidth="max-w-lg"
      >
        {createModal && (
          <CreateForm
            doctors={activeDoctors}
            onSubmit={handleCreateSubmit}
            isLoading={createMut.isPending}
            initialDoctorId={createModal.doctorId}
            initialStartAt={createModal.startAt}
            onClose={() => setCreateModal(null)}
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
