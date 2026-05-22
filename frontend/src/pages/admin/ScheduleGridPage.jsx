import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { addDays, subDays, format, isToday } from 'date-fns'
import { ru } from 'date-fns/locale'
import { ChevronLeft, ChevronRight, Plus, Building2, Check, X, SquareCheck, UserX } from 'lucide-react'
import toast from 'react-hot-toast'
import AppointmentGrid from '../../components/AppointmentGrid'
import Modal from '../../components/Modal'
import Badge from '../../components/Badge'
import ConfirmDialog from '../../components/ConfirmDialog'
import { getBranches } from '../../api/branches'
import { getDoctors } from '../../api/doctors'
import { getDoctorServices } from '../../api/services'
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
    register,
    handleSubmit,
    setValue,
    formState: { errors },
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

  // Notify parent about pre-filled doctor so services load
  useEffect(() => {
    if (initialDoctorId) onDoctorChange(String(initialDoctorId))
  }, [initialDoctorId]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            ФИО пациента <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            {...register('patient_name')}
          />
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
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
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
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          {...register('start_at')}
        />
        {errors.start_at && (
          <p className="mt-1 text-xs text-red-600">{errors.start_at.message}</p>
        )}
        <p className="mt-1 text-xs text-gray-400">
          Длительность рассчитывается автоматически по услуге
        </p>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Комментарий</label>
        <input
          type="text"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
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
            Запись пациента <span className="font-medium">{target.patient_name}</span> будет
            отменена.
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
  const [branchId, setBranchId] = useState('')

  // Modal state
  const [selectedAppt, setSelectedAppt] = useState(null)
  const [createModal, setCreateModal] = useState(null) // { doctorId, startAt } | null
  const [cancelTarget, setCancelTarget] = useState(null)
  const [simpleAction, setSimpleAction] = useState(null)
  const [createDoctorId, setCreateDoctorId] = useState('')

  // ── Data ──
  const { data: branches = [] } = useQuery({
    queryKey: ['branches'],
    queryFn: getBranches,
  })

  const { data: doctors = [] } = useQuery({
    queryKey: ['doctors'],
    queryFn: getDoctors,
  })

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
    setCreateModal({
      doctorId,
      startAt: `${format(date, 'yyyy-MM-dd')}T${startTime}`,
    })
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
  const today = isToday(date)

  return (
    <div className="flex flex-col h-full bg-gray-50">

      {/* ── Toolbar ── */}
      <div className="shrink-0 flex items-center gap-3 px-4 py-2.5 bg-white border-b border-gray-200">

        {/* Date navigation */}
        <div className="flex items-center gap-1">
          <button
            onClick={() => setDate((d) => subDays(d, 1))}
            className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-500 transition-colors"
          >
            <ChevronLeft size={18} />
          </button>
          <button
            onClick={() => setDate(new Date())}
            className={`px-3 py-1.5 text-sm rounded-lg border transition-colors ${
              today
                ? 'border-blue-500 bg-blue-50 text-blue-700 font-medium'
                : 'border-gray-300 hover:bg-gray-50 text-gray-700'
            }`}
          >
            Сегодня
          </button>
          <button
            onClick={() => setDate((d) => addDays(d, 1))}
            className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-500 transition-colors"
          >
            <ChevronRight size={18} />
          </button>
        </div>

        {/* Date label */}
        <span className="text-sm font-medium text-gray-900 capitalize hidden sm:block">
          {dateLabel}
        </span>

        <div className="flex-1" />

        {/* Branch switcher (foundation) */}
        {branches.length > 0 && (
          <div className="flex items-center gap-2">
            <Building2 size={15} className="text-gray-400 shrink-0" />
            <select
              value={branchId}
              onChange={(e) => setBranchId(e.target.value)}
              className="border border-gray-300 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
            >
              <option value="">Все филиалы</option>
              {branches.map((b) => (
                <option key={b.id} value={b.id}>
                  {b.name}
                </option>
              ))}
            </select>
          </div>
        )}

        {/* New appointment button */}
        <button
          onClick={() => setCreateModal({ doctorId: null, startAt: null })}
          className="flex items-center gap-2 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
        >
          <Plus size={15} />
          Новая запись
        </button>
      </div>

      {/* ── Grid ── */}
      <AppointmentGrid
        date={date}
        branchId={branchId ? Number(branchId) : undefined}
        onEventClick={handleEventClick}
        onSlotClick={handleSlotClick}
      />

      {/* ── Modals ── */}

      {/* Event detail */}
      <EventDetailModal
        appt={selectedAppt}
        onClose={() => setSelectedAppt(null)}
        onConfirm={handleConfirmAppt}
        onCancel={(appt) => { setSelectedAppt(null); setCancelTarget(appt) }}
        onComplete={handleCompleteAppt}
        onNoShow={handleNoShowAppt}
      />

      {/* Create appointment */}
      <Modal
        isOpen={!!createModal}
        onClose={() => { setCreateModal(null); setCreateDoctorId('') }}
        title="Новая запись"
        maxWidth="max-w-lg"
      >
        {createModal && (
          <CreateForm
            doctors={doctors.filter((d) => d.is_active)}
            services={createServices}
            onDoctorChange={setCreateDoctorId}
            onSubmit={handleCreateSubmit}
            isLoading={createMut.isPending}
            initialDoctorId={createModal.doctorId}
            initialStartAt={createModal.startAt}
          />
        )}
      </Modal>

      {/* Cancel */}
      <CancelModal
        target={cancelTarget}
        onClose={() => setCancelTarget(null)}
        onConfirm={(comment) => cancelMut.mutate({ id: cancelTarget.id, comment })}
        isLoading={cancelMut.isPending}
      />

      {/* Confirm / Complete / No-show */}
      <ConfirmDialog
        isOpen={!!simpleAction}
        onClose={() => setSimpleAction(null)}
        onConfirm={handleSimpleConfirm}
        title={simpleAction?.title ?? ''}
        message={
          simpleAction
            ? `Пациент: ${simpleAction.appt.patient_name}`
            : ''
        }
        confirmLabel={simpleAction?.label ?? 'Подтвердить'}
        confirmVariant="primary"
        isLoading={simpleActionPending}
      />
    </div>
  )
}
