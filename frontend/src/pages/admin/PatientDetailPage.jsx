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
  Check, X, UserRound,
} from 'lucide-react'
import { format } from 'date-fns'
import { getPatient, updatePatient } from '../../api/patients'
import { getAppointments } from '../../api/appointments'
import Modal from '../../components/Modal'

const STATUS_DOT = {
  created:              'bg-blue-400',
  confirmed:            'bg-emerald-400',
  completed:            'bg-gray-300',
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

// ─── helpers ────────────────────────────────────────────────────────────────

const AVATAR_COLORS = [
  'bg-blue-500', 'bg-emerald-500', 'bg-violet-500',
  'bg-amber-500', 'bg-rose-500', 'bg-cyan-500',
]

function avatarBg(id) {
  return AVATAR_COLORS[id % AVATAR_COLORS.length]
}

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

// ─── InfoRow ────────────────────────────────────────────────────────────────

function InfoRow({ icon: Icon, label, value }) {
  return (
    <div className="flex items-start gap-3 py-3 border-b border-gray-100 last:border-0">
      <Icon size={16} className="text-gray-400 mt-0.5 shrink-0" />
      <span className="text-sm text-gray-500 w-36 shrink-0">{label}</span>
      <span className="text-sm text-gray-900 font-medium">{value || '—'}</span>
    </div>
  )
}

// ─── PlaceholderSection ──────────────────────────────────────────────────────

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

// ─── EditPatientModal ────────────────────────────────────────────────────────

const editSchema = z.object({
  full_name: z.string().min(1, 'Обязательное поле'),
  phone: z.string().min(1, 'Обязательное поле'),
  email: z.string().email('Некорректный email').or(z.literal('')).optional(),
  date_of_birth: z.string().optional(),
})

function EditPatientModal({ patient, isOpen, onClose, onSave, isSaving }) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(editSchema),
    defaultValues: {
      full_name: patient.full_name,
      phone: patient.phone,
      email: patient.email ?? '',
      date_of_birth: isoToDateInput(patient.date_of_birth),
    },
  })

  const onSubmit = (data) => {
    const payload = {
      full_name: data.full_name,
      phone: data.phone,
      email: data.email || undefined,
      date_of_birth: data.date_of_birth || undefined,
    }
    onSave(payload)
  }

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Редактировать пациента">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            ФИО <span className="text-red-500">*</span>
          </label>
          <input
            {...register('full_name')}
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm
                       focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          {errors.full_name && (
            <p className="text-xs text-red-600 mt-1">{errors.full_name.message}</p>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Телефон <span className="text-red-500">*</span>
          </label>
          <input
            {...register('phone')}
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm
                       focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          {errors.phone && (
            <p className="text-xs text-red-600 mt-1">{errors.phone.message}</p>
          )}
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
            <input
              {...register('email')}
              type="email"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm
                         focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
            {errors.email && (
              <p className="text-xs text-red-600 mt-1">{errors.email.message}</p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Дата рождения</label>
            <input
              {...register('date_of_birth')}
              type="date"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm
                         focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        </div>

        <div className="flex justify-end gap-3 pt-2">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300
                       rounded-lg hover:bg-gray-50 transition-colors"
          >
            Отмена
          </button>
          <button
            type="submit"
            disabled={isSaving}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg
                       hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {isSaving ? 'Сохранение…' : 'Сохранить'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

// ─── PatientDetailPage ───────────────────────────────────────────────────────

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
    queryFn: () => getAppointments({ patient_id: Number(id), limit: 30 }),
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

  const startEditComment = () => {
    setCommentDraft(patient?.comment ?? '')
    setEditingComment(true)
  }

  const cancelEditComment = () => {
    setEditingComment(false)
    setCommentDraft('')
  }

  // ── Loading skeleton ──────────────────────────────────────────────────────
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

  const age = calcAge(patient.date_of_birth)
  const label = ageLabel(age)

  return (
    <div className="p-8 max-w-3xl mx-auto">
      {/* Back */}
      <button
        type="button"
        onClick={() => navigate('/admin/patients')}
        className="flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-900 transition-colors mb-5"
      >
        <ArrowLeft size={15} />
        Пациенты
      </button>

      {/* ── Header card ────────────────────────────────────────────────── */}
      <div className="bg-white rounded-xl border border-gray-200 p-6 mb-4">
        <div className="flex items-start gap-5">
          {/* Avatar */}
          <div
            className={`w-16 h-16 rounded-full ${avatarBg(patient.id)} flex items-center justify-center
                        text-white text-xl font-bold shrink-0`}
          >
            {initials(patient.full_name)}
          </div>

          {/* Name + meta */}
          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-3">
              <h1 className="text-xl font-semibold text-gray-900 leading-tight">
                {patient.full_name}
              </h1>
              <button
                type="button"
                onClick={() => setEditOpen(true)}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-600
                           bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors shrink-0"
              >
                <Pencil size={13} />
                Изменить
              </button>
            </div>

            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 mt-2">
              <a
                href={`tel:${patient.phone}`}
                className="flex items-center gap-1.5 text-sm text-blue-600 hover:text-blue-700 transition-colors"
              >
                <Phone size={13} className="text-blue-400" />
                {patient.phone}
              </a>
              {patient.email && (
                <a
                  href={`mailto:${patient.email}`}
                  className="flex items-center gap-1.5 text-sm text-blue-500 hover:text-blue-600 transition-colors"
                >
                  <Mail size={13} className="text-blue-400" />
                  {patient.email}
                </a>
              )}
              {label && (
                <span className="text-sm text-gray-500 font-medium">{label}</span>
              )}
              <SourceBadge source={patient.source} />
            </div>
          </div>
        </div>
      </div>

      {/* ── Basic info ─────────────────────────────────────────────────── */}
      <div className="bg-white rounded-xl border border-gray-200 px-5 mb-4">
        <h2 className="text-xs font-semibold text-gray-400 uppercase tracking-wider pt-4 pb-3">
          Основная информация
        </h2>
        {patient.date_of_birth && (
          <InfoRow
            icon={Calendar}
            label="Дата рождения"
            value={`${formatDate(patient.date_of_birth)}${label ? ` (${label})` : ''}`}
          />
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

      {/* ── Comment ────────────────────────────────────────────────────── */}
      <div className="bg-white rounded-xl border border-gray-200 p-5 mb-4">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <MessageSquare size={15} className="text-gray-400" />
            <h2 className="text-sm font-semibold text-gray-700">Комментарий</h2>
          </div>
          {!editingComment && (
            <button
              type="button"
              onClick={startEditComment}
              className="text-gray-400 hover:text-blue-600 transition-colors"
              title="Редактировать комментарий"
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
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm resize-none
                         focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => updateMut.mutate({ comment: commentDraft })}
                disabled={updateMut.isPending}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white
                           bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                <Check size={12} />
                {updateMut.isPending ? 'Сохранение…' : 'Сохранить'}
              </button>
              <button
                type="button"
                onClick={cancelEditComment}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-gray-600
                           bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
              >
                <X size={12} />
                Отмена
              </button>
            </div>
          </div>
        ) : patient.comment ? (
          <p className="text-sm text-gray-700 leading-relaxed whitespace-pre-wrap">
            {patient.comment}
          </p>
        ) : (
          <p className="text-sm text-gray-400 italic">Комментарий не добавлен</p>
        )}
      </div>

      {/* ── Appointment history ─────────────────────────────────────── */}
      <div className="bg-white rounded-xl border border-gray-200 p-5 mb-4">
        <div className="flex items-center gap-2 mb-4">
          <CalendarClock size={15} className="text-gray-400" />
          <h2 className="text-sm font-semibold text-gray-700">История записей</h2>
          {history.length > 0 && (
            <span className="ml-auto text-xs text-gray-400 tabular-nums">{history.length}</span>
          )}
        </div>
        {loadingHistory ? (
          <div className="space-y-2">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-8 bg-gray-100 rounded animate-pulse" />
            ))}
          </div>
        ) : history.length === 0 ? (
          <p className="text-sm text-gray-400 italic">Записей нет</p>
        ) : (
          <div className="divide-y divide-gray-100">
            {history.map((a) => (
              <div key={a.id} className="flex items-center gap-3 py-2.5">
                <span className="text-xs text-gray-400 w-24 shrink-0 tabular-nums">
                  {format(new Date(a.start_at), 'dd.MM.yyyy')}
                </span>
                <span className={`w-2 h-2 rounded-full shrink-0 ${STATUS_DOT[a.status] ?? 'bg-gray-300'}`} />
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-gray-800 truncate">{a.doctor_full_name}</p>
                  {a.service_name && (
                    <p className="text-xs text-gray-400 truncate">{a.service_name}</p>
                  )}
                </div>
                <span className="text-xs text-gray-400 shrink-0 hidden sm:block">
                  {STATUS_LABEL[a.status] ?? a.status}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* ── Future sections ─────────────────────────────────────────── */}
      <div className="grid grid-cols-2 gap-3">
        <PlaceholderSection
          icon={BarChart2}
          title="Аналитика"
          description="Статистика посещений"
        />
        <PlaceholderSection
          icon={FileText}
          title="Документы и анализы"
          description="Результаты, PDF, файлы"
        />
        <PlaceholderSection
          icon={MessagesSquare}
          title="Коммуникации"
          description="SMS, WhatsApp, Telegram"
        />
      </div>

      {/* ── Edit modal ─────────────────────────────────────────────────── */}
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
