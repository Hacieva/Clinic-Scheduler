import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Check, X, SquareCheck, UserX } from 'lucide-react'
import { format, parseISO } from 'date-fns'
import toast from 'react-hot-toast'
import {
  getAppointments,
  createAppointment,
  confirmAppointment,
  cancelAppointment,
  completeAppointment,
  noShowAppointment,
} from '../../api/appointments'
import { getDoctors } from '../../api/doctors'
import { getAssignedServices } from '../../api/services'
import DataTable from '../../components/DataTable'
import Modal from '../../components/Modal'
import ConfirmDialog from '../../components/ConfirmDialog'
import Badge from '../../components/Badge'

// ─── Constants ───────────────────────────────────────────────────────────────

const LIMIT = 50

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

const STATUS_OPTIONS = [
  { value: '', label: 'Все статусы' },
  { value: 'created', label: 'Создан' },
  { value: 'confirmed', label: 'Подтверждён' },
  { value: 'cancelled_by_admin', label: 'Отменён (адм.)' },
  { value: 'completed', label: 'Завершён' },
  { value: 'no_show', label: 'Не пришёл' },
]

const TERMINAL = new Set([
  'cancelled_by_admin',
  'cancelled_by_patient',
  'completed',
  'no_show',
])

// ─── Pure helpers ─────────────────────────────────────────────────────────────

// Maps UI filter state to the query params the API accepts.
function toQueryParams(filters, offset) {
  const params = { limit: LIMIT, offset }
  if (filters.doctor_id) params.doctor_id = filters.doctor_id
  if (filters.status) params.status = filters.status
  if (filters.date_from) params.date_from = filters.date_from
  if (filters.date_to) params.date_to = filters.date_to
  return params
}

function fmtDateTime(iso) {
  try {
    return format(parseISO(iso), 'dd.MM.yy HH:mm')
  } catch {
    return iso
  }
}

function sourceLabel(src) {
  return src === 'admin_panel' ? 'Адм.' : 'Бот'
}

// ─── AppointmentFilters (dumb — receives state + setters) ─────────────────────

function AppointmentFilters({ filters, setFilters, doctors }) {
  const set = (key) => (e) => setFilters((f) => ({ ...f, [key]: e.target.value }))

  const reset = () =>
    setFilters({ doctor_id: '', status: '', date_from: '', date_to: '' })

  const hasFilters = Object.values(filters).some(Boolean)

  return (
    <div className="flex flex-wrap items-end gap-3 mb-5 p-4 bg-white rounded-xl border border-gray-200">
      <div>
        <label className="block text-xs font-medium text-gray-500 mb-1">Врач</label>
        <select
          value={filters.doctor_id}
          onChange={set('doctor_id')}
          className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 min-w-[180px]"
        >
          <option value="">Все врачи</option>
          {doctors.map((d) => (
            <option key={d.id} value={d.id}>
              {[d.last_name, d.first_name, d.middle_name].filter(Boolean).join(' ')}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-500 mb-1">Статус</label>
        <select
          value={filters.status}
          onChange={set('status')}
          className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {STATUS_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-500 mb-1">Дата с</label>
        <input
          type="date"
          value={filters.date_from}
          onChange={set('date_from')}
          className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-500 mb-1">Дата по</label>
        <input
          type="date"
          value={filters.date_to}
          onChange={set('date_to')}
          className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      {hasFilters && (
        <button
          onClick={reset}
          className="px-3 py-2 text-sm text-gray-600 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
        >
          Сбросить
        </button>
      )}
    </div>
  )
}

// ─── AppointmentsTable (dumb — receives data + action callbacks) ───────────────

function ActionButtons({ row, onConfirm, onCancel, onComplete, onNoShow }) {
  if (TERMINAL.has(row.status)) return null

  return (
    <div className="flex items-center gap-1 justify-end">
      {row.status === 'created' && (
        <button
          onClick={() => onConfirm(row)}
          className="p-1.5 text-gray-400 hover:text-green-600 rounded transition-colors"
          title="Подтвердить"
        >
          <Check size={15} />
        </button>
      )}
      {row.status === 'confirmed' && (
        <>
          <button
            onClick={() => onComplete(row)}
            className="p-1.5 text-gray-400 hover:text-blue-600 rounded transition-colors"
            title="Завершить"
          >
            <SquareCheck size={15} />
          </button>
          <button
            onClick={() => onNoShow(row)}
            className="p-1.5 text-gray-400 hover:text-orange-500 rounded transition-colors"
            title="Не пришёл"
          >
            <UserX size={15} />
          </button>
        </>
      )}
      <button
        onClick={() => onCancel(row)}
        className="p-1.5 text-gray-400 hover:text-red-600 rounded transition-colors"
        title="Отменить"
      >
        <X size={15} />
      </button>
    </div>
  )
}

function AppointmentsTable({ appointments, isLoading, onConfirm, onCancel, onComplete, onNoShow }) {
  const columns = [
    {
      key: 'start_at',
      label: 'Дата/время',
      render: (row) => (
        <span className="text-sm font-medium whitespace-nowrap">
          {fmtDateTime(row.start_at)}
        </span>
      ),
    },
    {
      key: 'patient',
      label: 'Пациент',
      render: (row) => row.patient_name,
    },
    {
      key: 'phone',
      label: 'Телефон',
      render: (row) => (
        <span className="text-sm text-gray-600 whitespace-nowrap">{row.patient_phone}</span>
      ),
    },
    {
      key: 'doctor',
      label: 'Врач',
      render: (row) => (
        <span className="text-sm whitespace-nowrap">{row.doctor_full_name}</span>
      ),
    },
    {
      key: 'service',
      label: 'Услуга',
      render: (row) => row.service_name,
    },
    {
      key: 'source',
      label: 'Источник',
      render: (row) => (
        <span className="text-xs text-gray-500">{sourceLabel(row.source)}</span>
      ),
    },
    {
      key: 'status',
      label: 'Статус',
      render: (row) => (
        <Badge variant={STATUS_VARIANTS[row.status] ?? 'inactive'}>
          {STATUS_LABELS[row.status] ?? row.status}
        </Badge>
      ),
    },
    {
      key: 'actions',
      label: '',
      render: (row) => (
        <ActionButtons
          row={row}
          onConfirm={onConfirm}
          onCancel={onCancel}
          onComplete={onComplete}
          onNoShow={onNoShow}
        />
      ),
    },
  ]

  return (
    <DataTable
      columns={columns}
      data={appointments}
      loading={isLoading}
      emptyText="Записей нет"
    />
  )
}

// ─── CancelModal (dumb — owns comment field, no API calls) ────────────────────

function CancelModal({ target, onClose, onConfirm, isLoading }) {
  const [comment, setComment] = useState('')

  return (
    <Modal isOpen={!!target} onClose={onClose} title="Отменить запись">
      {target && (
        <>
          <p className="text-sm text-gray-600 mb-4">
            Запись пациента <span className="font-medium">{target.patient_name}</span> на{' '}
            {fmtDateTime(target.start_at)} будет отменена.
          </p>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Комментарий (необязательно)
            </label>
            <input
              type="text"
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Причина отмены"
            />
          </div>
          <div className="flex justify-end gap-3">
            <button
              onClick={onClose}
              disabled={isLoading}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-60 transition-colors"
            >
              Назад
            </button>
            <button
              onClick={() => onConfirm(comment || undefined)}
              disabled={isLoading}
              className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 disabled:opacity-60 transition-colors"
            >
              {isLoading ? 'Отмена...' : 'Отменить запись'}
            </button>
          </div>
        </>
      )}
    </Modal>
  )
}

// ─── CreateAppointmentForm (dumb — no API calls) ──────────────────────────────

const createSchema = z.object({
  patient_name: z.string().min(1, 'Введите ФИО пациента'),
  patient_phone: z.string().min(7, 'Введите номер телефона'),
  doctor_id: z.string().min(1, 'Выберите врача'),
  service_id: z.string().min(1, 'Выберите услугу'),
  start_at: z.string().min(1, 'Укажите дату и время'),
  patient_comment: z.string().optional(),
})

function CreateAppointmentForm({ doctors, services, onDoctorChange, onSubmit, isLoading }) {
  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(createSchema),
    defaultValues: {
      doctor_id: '',
      service_id: '',
      patient_name: '',
      patient_phone: '',
      start_at: '',
      patient_comment: '',
    },
  })

  const { onChange: rhfDoctorChange, ...restDoctor } = register('doctor_id')

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
          Дата и время начала <span className="text-red-500">*</span>
        </label>
        <input
          type="datetime-local"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          {...register('start_at')}
        />
        {errors.start_at && (
          <p className="mt-1 text-xs text-red-600">{errors.start_at.message}</p>
        )}
        <p className="mt-1 text-xs text-gray-500">
          Длительность и время окончания рассчитываются автоматически по выбранной услуге.
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

      <div className="flex justify-end pt-2">
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-60 transition-colors"
        >
          {isLoading ? 'Создание...' : 'Создать запись'}
        </button>
      </div>
    </form>
  )
}

// ─── AppointmentsPage (owns state, queries, mutations) ────────────────────────

export default function AppointmentsPage() {
  const qc = useQueryClient()

  // ── Filters state (separate concern) ──
  const [filters, setFilters] = useState({
    doctor_id: '',
    status: '',
    date_from: '',
    date_to: '',
  })

  // ── Pagination state (ready, no UI yet) ──
  const [offset] = useState(0)

  // ── Action targets ──
  const [cancelTarget, setCancelTarget] = useState(null)
  const [simpleAction, setSimpleAction] = useState(null) // { row, action, title, confirmLabel }
  const [createOpen, setCreateOpen] = useState(false)
  const [createDoctorId, setCreateDoctorId] = useState('')

  // ── Data fetching (separate concern) ──
  const { data: appointments = [], isLoading } = useQuery({
    queryKey: ['appointments', filters, offset],
    queryFn: () => getAppointments(toQueryParams(filters, offset)),
  })

  const { data: doctors = [] } = useQuery({
    queryKey: ['doctors'],
    queryFn: getDoctors,
  })

  const { data: createServices = [] } = useQuery({
    queryKey: ['assigned-services', createDoctorId],
    queryFn: () => getAssignedServices(Number(createDoctorId)),
    enabled: !!createDoctorId,
  })

  const invalidateList = () => qc.invalidateQueries({ queryKey: ['appointments'] })

  // ── Mutations — one per action (no generic mutate) ──

  const createMut = useMutation({
    mutationFn: (data) => createAppointment(data),
    onSuccess: () => {
      invalidateList()
      setCreateOpen(false)
      setCreateDoctorId('')
      toast.success('Запись создана')
    },
    onError: (err) => {
      const msg = err?.response?.data?.error ?? ''
      const status = err?.response?.status
      if (status === 409 && msg.includes('slot')) {
        toast.error('Время уже занято. Выберите другой слот.')
      } else if (status === 422 && msg.includes('inactive')) {
        toast.error('Врач неактивен.')
      } else if (status === 422 && msg.includes('outside')) {
        toast.error('Время вне рабочих часов врача.')
      } else {
        toast.error('Не удалось создать запись.')
      }
    },
  })

  const confirmMut = useMutation({
    mutationFn: (id) => confirmAppointment(id),
    onSuccess: () => {
      invalidateList()
      setSimpleAction(null)
      toast.success('Запись подтверждена')
    },
    onError: (err) => {
      if (err?.response?.status === 409) toast.error('Недопустимое изменение статуса.')
      else toast.error('Не удалось подтвердить запись.')
    },
  })

  const cancelMut = useMutation({
    mutationFn: ({ id, comment }) => cancelAppointment(id, comment),
    onSuccess: () => {
      invalidateList()
      setCancelTarget(null)
      toast.success('Запись отменена')
    },
    onError: (err) => {
      if (err?.response?.status === 409) toast.error('Недопустимое изменение статуса.')
      else toast.error('Не удалось отменить запись.')
    },
  })

  const completeMut = useMutation({
    mutationFn: (id) => completeAppointment(id),
    onSuccess: () => {
      invalidateList()
      setSimpleAction(null)
      toast.success('Запись завершена')
    },
    onError: (err) => {
      if (err?.response?.status === 409) toast.error('Недопустимое изменение статуса.')
      else toast.error('Не удалось завершить запись.')
    },
  })

  const noShowMut = useMutation({
    mutationFn: (id) => noShowAppointment(id),
    onSuccess: () => {
      invalidateList()
      setSimpleAction(null)
      toast.success('Отмечено: не пришёл')
    },
    onError: (err) => {
      if (err?.response?.status === 409) toast.error('Недопустимое изменение статуса.')
      else toast.error('Не удалось обновить статус.')
    },
  })

  // ── Action handlers (passed down to table as explicit callbacks) ──

  const handleConfirm = (row) =>
    setSimpleAction({ row, action: 'confirm', title: 'Подтвердить запись', confirmLabel: 'Подтвердить', confirmVariant: 'primary' })

  const handleComplete = (row) =>
    setSimpleAction({ row, action: 'complete', title: 'Завершить запись', confirmLabel: 'Завершить', confirmVariant: 'success' })

  const handleNoShow = (row) =>
    setSimpleAction({ row, action: 'noShow', title: 'Отметить как «Не пришёл»', confirmLabel: 'Отметить', confirmVariant: 'warning' })

  const handleSimpleConfirm = () => {
    if (!simpleAction) return
    const id = simpleAction.row.id
    if (simpleAction.action === 'confirm') confirmMut.mutate(id)
    else if (simpleAction.action === 'complete') completeMut.mutate(id)
    else if (simpleAction.action === 'noShow') noShowMut.mutate(id)
  }

  const isSimpleActionPending =
    confirmMut.isPending || completeMut.isPending || noShowMut.isPending

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

  const simpleActionMessage = simpleAction
    ? `Пациент: ${simpleAction.row.patient_name}, ${fmtDateTime(simpleAction.row.start_at)}`
    : ''

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Записи</h1>
        <button
          onClick={() => setCreateOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
        >
          <Plus size={16} />
          Новая запись
        </button>
      </div>

      {/* Filters (separate concern) */}
      <AppointmentFilters filters={filters} setFilters={setFilters} doctors={doctors} />

      {/* Table UI (separate concern) */}
      <AppointmentsTable
        appointments={appointments}
        isLoading={isLoading}
        onConfirm={handleConfirm}
        onCancel={setCancelTarget}
        onComplete={handleComplete}
        onNoShow={handleNoShow}
      />

      {/* Create modal */}
      <Modal
        isOpen={createOpen}
        onClose={() => {
          setCreateOpen(false)
          setCreateDoctorId('')
        }}
        title="Новая запись"
        maxWidth="max-w-lg"
      >
        <CreateAppointmentForm
          doctors={doctors}
          services={createServices}
          onDoctorChange={setCreateDoctorId}
          onSubmit={handleCreateSubmit}
          isLoading={createMut.isPending}
        />
      </Modal>

      {/* Cancel modal (separate — has comment field) */}
      <CancelModal
        target={cancelTarget}
        onClose={() => setCancelTarget(null)}
        onConfirm={(comment) => cancelMut.mutate({ id: cancelTarget.id, comment })}
        isLoading={cancelMut.isPending}
      />

      {/* Confirm / Complete / No-show — shared ConfirmDialog, explicit per-action mutations */}
      <ConfirmDialog
        isOpen={!!simpleAction}
        onClose={() => setSimpleAction(null)}
        onConfirm={handleSimpleConfirm}
        title={simpleAction?.title ?? ''}
        message={simpleActionMessage}
        confirmLabel={simpleAction?.confirmLabel ?? 'Подтвердить'}
        confirmVariant={simpleAction?.confirmVariant ?? 'primary'}
        isLoading={isSimpleActionPending}
      />
    </div>
  )
}
