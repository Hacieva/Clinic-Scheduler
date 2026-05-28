import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import toast from 'react-hot-toast'
import {
  ArrowLeft, Pencil, Phone, Mail, Calendar, MessageSquare,
  CalendarClock, Lock, BarChart2, FileText, MessagesSquare,
  Check, X, UserRound, Baby, User, Plus, RefreshCw,
  Clock, MapPin,
} from 'lucide-react'
import { format, isPast, isFuture, parseISO } from 'date-fns'
import { ru } from 'date-fns/locale'
import { getPatient, updatePatient } from '../../api/patients'
import { getAppointments } from '../../api/appointments'
import Modal from '../../components/Modal'

// ─── Status maps ─────────────────────────────────────────────────────────────

const STATUS_DOT = {
  created:              'bg-blue-400',
  confirmed:            'bg-emerald-400',
  completed:            'bg-gray-400',
  cancelled_by_admin:   'bg-rose-400',
  cancelled_by_patient: 'bg-rose-400',
  no_show:              'bg-amber-400',
}

const STATUS_LABEL = {
  created:              'Ожидает',
  confirmed:            'Подтверждён',
  completed:            'Завершён',
  cancelled_by_admin:   'Отменён',
  cancelled_by_patient: 'Отменён пациентом',
  no_show:              'Не пришёл',
}

const TERMINAL = new Set(['cancelled_by_admin', 'cancelled_by_patient', 'completed', 'no_show'])

// ─── Helpers ──────────────────────────────────────────────────────────────────

const AVATAR_COLORS = [
  'bg-blue-500', 'bg-emerald-500', 'bg-violet-500',
  'bg-amber-500', 'bg-rose-500', 'bg-cyan-500',
]
function avatarBg(id) { return AVATAR_COLORS[id % AVATAR_COLORS.length] }

function initials(name) {
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase()
  return parts[0].slice(0, 2).toUpperCase()
}

function calcAge(iso) {
  if (!iso) return null
  const dob = new Date(iso)
  const today = new Date()
  let age = today.getFullYear() - dob.getFullYear()
  const m = today.getMonth() - dob.getMonth()
  if (m < 0 || (m === 0 && today.getDate() < dob.getDate())) age--
  return age
}

function ageLabel(n) {
  if (n === null) return null
  if (n % 100 >= 11 && n % 100 <= 14) return `${n} лет`
  switch (n % 10) {
    case 1: return `${n} год`
    case 2: case 3: case 4: return `${n} года`
    default: return `${n} лет`
  }
}

function formatDate(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('ru-RU', {
    day: 'numeric', month: 'long', year: 'numeric',
  })
}

function isoToDateInput(iso) {
  if (!iso) return ''
  return iso.substring(0, 10)
}

function fmtApptDate(iso) {
  try { return format(parseISO(iso), 'dd MMM yyyy', { locale: ru }) } catch { return iso }
}

function fmtApptTime(iso) {
  try { return format(parseISO(iso), 'HH:mm') } catch { return '' }
}

// ─── Sub-components ───────────────────────────────────────────────────────────

function SourceBadge({ source }) {
  if (source === 'telegram_bot') {
    return (
      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-sky-50 text-sky-700 border border-sky-200">
        Telegram
      </span>
    )
  }
  return (
    <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-500 border border-gray-200">
      Администратор
    </span>
  )
}

function PatientTypeBadge({ age }) {
  if (age === null) return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-gray-50 text-gray-400 border border-gray-200">
      <UserRound size={10} />
      Возраст не указан
    </span>
  )
  if (age < 18) return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700 border border-blue-200">
      <Baby size={10} />
      Ребёнок
    </span>
  )
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-50 text-green-700 border border-green-200">
      <User size={10} />
      Взрослый
    </span>
  )
}

function InfoRow({ icon: Icon, label, value }) {
  return (
    <div className="flex items-start gap-3 py-3 border-b border-gray-100 last:border-0">
      <Icon size={16} className="text-gray-400 mt-0.5 shrink-0" />
      <span className="text-sm text-gray-500 w-36 shrink-0">{label}</span>
      <span className="text-sm text-gray-900 font-medium">{value || '—'}</span>
    </div>
  )
}

function PlaceholderSection({ icon: Icon, title, description }) {
  return (
    <div className="bg-white rounded-xl border border-dashed border-gray-200 p-5">
      <div className="flex items-center justify-between mb-1">
        <div className="flex items-center gap-2.5">
          <Icon size={17} className="text-gray-300" />
          <h3 className="text-sm font-semibold text-gray-400">{title}</h3>
        </div>
        <span className="flex items-center gap-1 text-xs text-gray-300 font-medium">
          <Lock size={11} />
          Скоро
        </span>
      </div>
      <p className="text-xs text-gray-400 ml-[29px]">{description}</p>
    </div>
  )
}

// ─── AppointmentRow ───────────────────────────────────────────────────────────

function AppointmentRow({ appt, onRepeat }) {
  const isTerminal = TERMINAL.has(appt.status)
  return (
    <div className="flex items-start gap-3 py-3 border-b border-gray-100 last:border-0 group">
      {/* Date + time */}
      <div className="w-24 shrink-0 text-right">
        <p className="text-xs font-medium text-gray-700 tabular-nums">{fmtApptDate(appt.start_at)}</p>
        <p className="text-[10px] text-gray-400 tabular-nums flex items-center justify-end gap-0.5 mt-0.5">
          <Clock size={9} />
          {fmtApptTime(appt.start_at)}
        </p>
      </div>

      {/* Status dot */}
      <span className={`w-2 h-2 rounded-full shrink-0 mt-1 ${STATUS_DOT[appt.status] ?? 'bg-gray-300'}`} />

      {/* Main content */}
      <div className="flex-1 min-w-0">
        <p className="text-sm text-gray-800 truncate font-medium">{appt.doctor_full_name}</p>
        {appt.service_name && (
          <p className="text-xs text-gray-500 truncate mt-0.5">{appt.service_name}</p>
        )}
        <div className="flex flex-wrap items-center gap-2 mt-1">
          {appt.branch_name && (
            <span className="flex items-center gap-0.5 text-[10px] text-gray-400">
              <MapPin size={9} />
              {appt.branch_name}
            </span>
          )}
          {appt.patient_comment && (
            <span className="text-[10px] text-gray-400 italic truncate max-w-[160px]">
              «{appt.patient_comment}»
            </span>
          )}
        </div>
      </div>

      {/* Status + repeat */}
      <div className="shrink-0 flex flex-col items-end gap-1.5">
        <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded-full ${
          appt.status === 'completed'            ? 'bg-gray-100 text-gray-500' :
          appt.status === 'confirmed'            ? 'bg-emerald-50 text-emerald-700' :
          appt.status === 'created'              ? 'bg-blue-50 text-blue-700' :
          appt.status === 'no_show'              ? 'bg-amber-50 text-amber-700' :
          'bg-rose-50 text-rose-600'
        }`}>
          {STATUS_LABEL[appt.status] ?? appt.status}
        </span>
        {isTerminal && onRepeat && (
          <button
            type="button"
            onClick={() => onRepeat(appt)}
            className="flex items-center gap-1 text-[10px] text-gray-400 hover:text-blue-600 transition-colors opacity-0 group-hover:opacity-100"
            title="Повторить запись с тем же врачом/услугой"
          >
            <RefreshCw size={9} />
            Повторить
          </button>
        )}
      </div>
    </div>
  )
}

// ─── AppointmentSection ───────────────────────────────────────────────────────

function AppointmentSection({ title, appointments, onRepeat, emptyText }) {
  if (appointments.length === 0) return null
  return (
    <div className="mb-1">
      <p className="text-[10px] font-semibold text-gray-400 uppercase tracking-wider mb-1">{title}</p>
      {appointments.map((a) => (
        <AppointmentRow key={a.id} appt={a} onRepeat={onRepeat} />
      ))}
    </div>
  )
}

// ─── EditPatientModal ─────────────────────────────────────────────────────────

const editSchema = z.object({
  full_name:    z.string().min(1, 'Обязательное поле'),
  phone:        z.string().min(1, 'Обязательное поле'),
  email:        z.string().email('Некорректный email').or(z.literal('')).optional(),
  date_of_birth: z.string().optional(),
})

function EditPatientModal({ patient, isOpen, onClose, onSave, isSaving }) {
  const { register, handleSubmit, formState: { errors } } = useForm({
    resolver: zodResolver(editSchema),
    defaultValues: {
      full_name:    patient.full_name,
      phone:        patient.phone,
      email:        patient.email ?? '',
      date_of_birth: isoToDateInput(patient.date_of_birth),
    },
  })

  const onSubmit = (data) => onSave({
    full_name:    data.full_name,
    phone:        data.phone,
    email:        data.email || undefined,
    date_of_birth: data.date_of_birth || undefined,
  })

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Редактировать пациента">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            ФИО <span className="text-red-500">*</span>
          </label>
          <input {...register('full_name')}
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          {errors.full_name && <p className="text-xs text-red-600 mt-1">{errors.full_name.message}</p>}
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Телефон <span className="text-red-500">*</span>
          </label>
          <input {...register('phone')}
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          {errors.phone && <p className="text-xs text-red-600 mt-1">{errors.phone.message}</p>}
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
            <input {...register('email')} type="email"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
            {errors.email && <p className="text-xs text-red-600 mt-1">{errors.email.message}</p>}
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Дата рождения</label>
            <input {...register('date_of_birth')} type="date"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        </div>

        <div className="flex justify-end gap-3 pt-2">
          <button type="button" onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
          >Отмена</button>
          <button type="submit" disabled={isSaving}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >{isSaving ? 'Сохранение…' : 'Сохранить'}</button>
        </div>
      </form>
    </Modal>
  )
}

// ─── PatientDetailPage ────────────────────────────────────────────────────────

export default function PatientDetailPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const qc = useQueryClient()

  const [editOpen, setEditOpen] = useState(false)
  const [editingComment, setEditingComment] = useState(false)
  const [commentDraft, setCommentDraft] = useState('')

  const { data: patient, isLoading } = useQuery({
    queryKey: ['patient', id],
    queryFn: () => getPatient(id),
  })

  const { data: history = [], isLoading: loadingHistory } = useQuery({
    queryKey: ['patient-appointments', id],
    queryFn: () => getAppointments({ patient_id: Number(id), limit: 100 }),
    enabled: !!patient,
  })

  const updateMut = useMutation({
    mutationFn: (data) => updatePatient(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['patient', id] })
      qc.invalidateQueries({ queryKey: ['patients'] })
      toast.success('Изменения сохранены')
      setEditOpen(false)
      setEditingComment(false)
    },
    onError: () => toast.error('Не удалось сохранить изменения'),
  })

  // ── Appointment grouping ────────────────────────────────────────────────────
  const now = new Date()
  const upcoming = history.filter((a) =>
    !TERMINAL.has(a.status) && isFuture(parseISO(a.start_at))
  )
  const past = history.filter((a) =>
    a.status === 'completed' || a.status === 'no_show' ||
    (a.status !== 'cancelled_by_admin' && a.status !== 'cancelled_by_patient' && isPast(parseISO(a.start_at)))
  )
  const cancelled = history.filter((a) =>
    a.status === 'cancelled_by_admin' || a.status === 'cancelled_by_patient'
  )

  // ── "Repeat booking" handler ────────────────────────────────────────────────
  // TODO: prefill doctor/service in appointment create modal once supported.
  // For now, navigate to appointments page with patient phone pre-filled.
  const handleRepeat = (appt) => {
    navigate('/admin/appointments', {
      state: {
        openCreate: true,
        patientPhone: patient?.phone,
        patientName: patient?.full_name,
        // Future: prefill doctorId + serviceId from appt
        _hint: `repeat from appt #${appt.id}`,
      },
    })
  }

  const handleNewAppointment = () => {
    navigate('/admin/appointments', {
      state: {
        openCreate: true,
        patientPhone: patient?.phone,
        patientName: patient?.full_name,
      },
    })
  }

  // ── Loading skeleton ────────────────────────────────────────────────────────
  if (isLoading) {
    return (
      <div className="p-8 max-w-3xl mx-auto space-y-4">
        <div className="h-4 w-24 bg-gray-200 rounded animate-pulse" />
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <div className="flex items-center gap-5">
            <div className="w-16 h-16 rounded-full bg-gray-200 animate-pulse" />
            <div className="space-y-2 flex-1">
              <div className="h-5 w-48 bg-gray-200 rounded animate-pulse" />
              <div className="h-4 w-32 bg-gray-100 rounded animate-pulse" />
            </div>
          </div>
        </div>
        <div className="bg-white rounded-xl border border-gray-200 p-6 space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-4 bg-gray-100 rounded animate-pulse" />
          ))}
        </div>
      </div>
    )
  }

  if (!patient) return null

  const age   = calcAge(patient.date_of_birth)
  const label = ageLabel(age)

  return (
    <div className="p-6 max-w-3xl mx-auto">

      {/* Back */}
      <button
        type="button"
        onClick={() => navigate('/admin/patients')}
        className="flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-900 transition-colors mb-5"
      >
        <ArrowLeft size={15} />
        Пациенты
      </button>

      {/* ── Header card ──────────────────────────────────────────────────── */}
      <div className="bg-white rounded-xl border border-gray-200 p-5 mb-3">
        <div className="flex items-start gap-4">
          {/* Avatar */}
          <div className={`w-14 h-14 rounded-full ${avatarBg(patient.id)} flex items-center justify-center text-white text-lg font-bold shrink-0`}>
            {initials(patient.full_name)}
          </div>

          {/* Name + meta */}
          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-3">
              <h1 className="text-xl font-semibold text-gray-900 leading-tight">{patient.full_name}</h1>
              <button
                type="button"
                onClick={() => setEditOpen(true)}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-600 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors shrink-0"
              >
                <Pencil size={13} />
                Изменить
              </button>
            </div>

            <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 mt-2">
              <a href={`tel:${patient.phone}`}
                className="flex items-center gap-1.5 text-sm text-blue-600 hover:text-blue-700 transition-colors"
              >
                <Phone size={13} className="text-blue-400" />
                {patient.phone}
              </a>
              {patient.email && (
                <a href={`mailto:${patient.email}`}
                  className="flex items-center gap-1.5 text-sm text-blue-500 hover:text-blue-600 transition-colors"
                >
                  <Mail size={13} className="text-blue-400" />
                  {patient.email}
                </a>
              )}
              {label && <span className="text-sm text-gray-500 font-medium">{label}</span>}
              <PatientTypeBadge age={age} />
              <SourceBadge source={patient.source} />
            </div>
          </div>
        </div>

        {/* Quick actions */}
        <div className="flex items-center gap-2 mt-4 pt-4 border-t border-gray-100">
          <button
            type="button"
            onClick={handleNewAppointment}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
          >
            <Plus size={14} />
            Записать пациента
          </button>
          <span className="text-xs text-gray-400">
            {history.length > 0
              ? `${history.length} записей в истории`
              : 'Записей нет'}
          </span>
        </div>
      </div>

      {/* ── Basic info ────────────────────────────────────────────────────── */}
      <div className="bg-white rounded-xl border border-gray-200 px-5 mb-3">
        <h2 className="text-xs font-semibold text-gray-400 uppercase tracking-wider pt-4 pb-3">
          Основная информация
        </h2>
        {patient.date_of_birth ? (
          <InfoRow
            icon={Calendar}
            label="Дата рождения"
            value={`${formatDate(patient.date_of_birth)}${label ? ` (${label})` : ''}`}
          />
        ) : (
          <InfoRow icon={Calendar} label="Дата рождения" value={null} />
        )}
        <InfoRow icon={Mail} label="Email" value={patient.email} />
        <InfoRow
          icon={UserRound}
          label="Источник"
          value={patient.source === 'telegram_bot' ? 'Telegram Bot' : 'Администратор'}
        />
        <InfoRow
          icon={CalendarClock}
          label="Добавлен"
          value={formatDate(patient.created_at)}
        />
      </div>

      {/* ── Comment ───────────────────────────────────────────────────────── */}
      <div className="bg-white rounded-xl border border-gray-200 p-5 mb-3">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <MessageSquare size={15} className="text-gray-400" />
            <h2 className="text-sm font-semibold text-gray-700">Комментарий</h2>
          </div>
          {!editingComment && (
            <button
              type="button"
              onClick={() => { setCommentDraft(patient.comment ?? ''); setEditingComment(true) }}
              className="text-gray-400 hover:text-blue-600 transition-colors"
            >
              <Pencil size={14} />
            </button>
          )}
        </div>

        {editingComment ? (
          <div className="space-y-2">
            <textarea
              value={commentDraft}
              onChange={(e) => setCommentDraft(e.target.value)}
              rows={3}
              placeholder="Введите комментарий…"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => updateMut.mutate({ comment: commentDraft })}
                disabled={updateMut.isPending}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                <Check size={12} />
                {updateMut.isPending ? 'Сохранение…' : 'Сохранить'}
              </button>
              <button
                type="button"
                onClick={() => setEditingComment(false)}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-600 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
              >
                <X size={12} />
                Отмена
              </button>
            </div>
          </div>
        ) : patient.comment ? (
          <p className="text-sm text-gray-700 leading-relaxed whitespace-pre-wrap">{patient.comment}</p>
        ) : (
          <p className="text-sm text-gray-400 italic">Комментарий не добавлен</p>
        )}
      </div>

      {/* ── Appointment history ────────────────────────────────────────────── */}
      <div className="bg-white rounded-xl border border-gray-200 p-5 mb-3">
        <div className="flex items-center gap-2 mb-4">
          <CalendarClock size={15} className="text-gray-400" />
          <h2 className="text-sm font-semibold text-gray-700">История записей</h2>
          {history.length > 0 && (
            <span className="ml-auto text-xs text-gray-400 tabular-nums">{history.length}</span>
          )}
        </div>

        {loadingHistory ? (
          <div className="space-y-2">
            {[1, 2, 3].map((i) => <div key={i} className="h-10 bg-gray-100 rounded animate-pulse" />)}
          </div>
        ) : history.length === 0 ? (
          <div className="flex flex-col items-center py-6 gap-2 text-gray-400">
            <CalendarClock size={28} strokeWidth={1.25} className="text-gray-300" />
            <p className="text-sm italic">Записей нет</p>
            <button
              type="button"
              onClick={handleNewAppointment}
              className="flex items-center gap-1.5 mt-1 px-3 py-1.5 text-xs font-medium text-blue-600 border border-blue-200 rounded-lg hover:bg-blue-50 transition-colors"
            >
              <Plus size={12} />
              Записать пациента
            </button>
          </div>
        ) : (
          <div>
            <AppointmentSection
              title="Предстоящие"
              appointments={upcoming}
              onRepeat={null}
            />
            <AppointmentSection
              title="Прошедшие"
              appointments={past}
              onRepeat={handleRepeat}
            />
            <AppointmentSection
              title="Отменённые"
              appointments={cancelled}
              onRepeat={handleRepeat}
            />
          </div>
        )}
      </div>

      {/* ── Future sections ────────────────────────────────────────────────── */}
      <div className="grid grid-cols-2 gap-3">
        <PlaceholderSection icon={BarChart2}     title="Аналитика"          description="Статистика посещений" />
        <PlaceholderSection icon={FileText}      title="Документы и анализы" description="Результаты, PDF, файлы" />
        <PlaceholderSection icon={MessagesSquare} title="Коммуникации"       description="SMS, WhatsApp, Telegram" />
      </div>

      {/* ── Edit modal ─────────────────────────────────────────────────────── */}
      {editOpen && (
        <EditPatientModal
          patient={patient}
          isOpen={editOpen}
          onClose={() => setEditOpen(false)}
          onSave={(data) => updateMut.mutate(data)}
          isSaving={updateMut.isPending}
        />
      )}
    </div>
  )
}
