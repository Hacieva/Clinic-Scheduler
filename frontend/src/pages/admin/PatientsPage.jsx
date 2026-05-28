import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Search, ChevronRight, UserRound, Phone, Mail, X, Plus, Baby, User } from 'lucide-react'
import toast from 'react-hot-toast'
import { getPatients, createPatient } from '../../api/patients'
import Modal from '../../components/Modal'

const LIMIT = 20

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

function SourceBadge({ source }) {
  if (source === 'telegram_bot') {
    return (
      <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-sky-50 text-sky-700 border border-sky-200">
        Telegram
      </span>
    )
  }
  return (
    <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-500 border border-gray-200">
      Администратор
    </span>
  )
}

function PatientRowSkeleton() {
  return (
    <div className="flex items-center gap-4 px-5 py-4 border-b border-gray-100 last:border-0">
      <div className="w-11 h-11 rounded-full bg-gray-200 animate-pulse shrink-0" />
      <div className="flex-1 min-w-0 space-y-2">
        <div className="h-4 w-48 bg-gray-200 rounded animate-pulse" />
        <div className="h-3 w-32 bg-gray-100 rounded animate-pulse" />
      </div>
      <div className="hidden sm:flex flex-col items-end gap-1.5">
        <div className="h-5 w-20 bg-gray-100 rounded-full animate-pulse" />
        <div className="h-3 w-24 bg-gray-100 rounded animate-pulse" />
      </div>
    </div>
  )
}

function PatientTypePip({ age }) {
  if (age === null) return null
  if (age < 18) return (
    <span className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-blue-50 text-blue-600 border border-blue-100">
      <Baby size={9} />
      Ребёнок
    </span>
  )
  return (
    <span className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-green-50 text-green-600 border border-green-100">
      <User size={9} />
      Взрослый
    </span>
  )
}

function fmtShort(iso) {
  if (!iso) return null
  return new Date(iso).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short', year: 'numeric' })
}

function PatientRow({ patient, onClick }) {
  const age   = calcAge(patient.date_of_birth)
  const label = ageLabel(age)

  return (
    <button
      type="button"
      onClick={onClick}
      className="w-full flex items-center gap-4 px-5 py-3.5 border-b border-gray-100 last:border-0 hover:bg-blue-50/50 transition-colors text-left group"
    >
      <div className={`w-10 h-10 rounded-full ${avatarBg(patient.id)} flex items-center justify-center text-white text-sm font-semibold shrink-0`}>
        {initials(patient.full_name)}
      </div>

      <div className="flex-1 min-w-0">
        <p className="text-sm font-semibold text-gray-900 truncate">{patient.full_name}</p>
        <div className="flex items-center gap-2.5 mt-0.5 flex-wrap">
          <a
            href={`tel:${patient.phone}`}
            onClick={(e) => e.stopPropagation()}
            className="flex items-center gap-1 text-xs text-gray-500 hover:text-blue-600 transition-colors"
          >
            <Phone size={11} />
            {patient.phone}
          </a>
          {patient.email && (
            <span className="flex items-center gap-1 text-xs text-gray-400 truncate max-w-[140px]">
              <Mail size={11} />
              {patient.email}
            </span>
          )}
          {label && <span className="text-xs text-gray-400">{label}</span>}
          <PatientTypePip age={age} />
        </div>
      </div>

      <div className="hidden sm:flex flex-col items-end gap-1.5 shrink-0 min-w-[110px]">
        <SourceBadge source={patient.source} />
        {patient.last_appointment_at ? (
          <span className="text-[10px] text-gray-400 tabular-nums">
            Посл.: {fmtShort(patient.last_appointment_at)}
          </span>
        ) : (
          <span className="text-[10px] text-gray-300 italic">Не записывался</span>
        )}
      </div>

      <ChevronRight size={16} className="text-gray-300 group-hover:text-blue-400 transition-colors shrink-0 ml-1" />
    </button>
  )
}

function EmptyState({ hasSearch }) {
  return (
    <div className="flex flex-col items-center justify-center py-16 gap-3 text-gray-400">
      <UserRound size={40} strokeWidth={1.25} />
      {hasSearch ? (
        <>
          <p className="text-sm font-medium text-gray-500">Пациенты не найдены</p>
          <p className="text-xs">Попробуйте изменить поисковый запрос</p>
        </>
      ) : (
        <>
          <p className="text-sm font-medium text-gray-500">Пациентов ещё нет</p>
          <p className="text-xs">Добавьте первого пациента</p>
        </>
      )}
    </div>
  )
}

// ─── Create patient schema ────────────────────────────────────────────────────

const createSchema = z.object({
  full_name: z.string().min(2, 'Введите ФИО'),
  phone: z.string().min(7, 'Введите телефон'),
  email: z.string().email('Некорректный email').optional().or(z.literal('')),
  date_of_birth: z.string().optional(),
  comment: z.string().optional(),
})

function CreatePatientModal({ isOpen, onClose, onCreated }) {
  const { register, handleSubmit, reset, formState: { errors } } = useForm({
    resolver: zodResolver(createSchema),
    defaultValues: { full_name: '', phone: '', email: '', date_of_birth: '', comment: '' },
  })

  const mut = useMutation({
    mutationFn: (data) => {
      const payload = {
        full_name: data.full_name,
        phone: data.phone,
        ...(data.email ? { email: data.email } : {}),
        ...(data.date_of_birth ? { date_of_birth: data.date_of_birth } : {}),
        ...(data.comment ? { comment: data.comment } : {}),
      }
      return createPatient(payload)
    },
    onSuccess: (patient) => {
      reset()
      onCreated(patient)
      toast.success('Пациент добавлен')
    },
    onError: () => toast.error('Не удалось добавить пациента'),
  })

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Новый пациент">
      <form onSubmit={handleSubmit((d) => mut.mutate(d))} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            ФИО <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            placeholder="Иванов Иван Иванович"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            {...register('full_name')}
          />
          {errors.full_name && <p className="mt-1 text-xs text-red-600">{errors.full_name.message}</p>}
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Телефон <span className="text-red-500">*</span>
            </label>
            <input
              type="tel"
              placeholder="+7 (999) 000-00-00"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              {...register('phone')}
            />
            {errors.phone && <p className="mt-1 text-xs text-red-600">{errors.phone.message}</p>}
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
            <input
              type="email"
              placeholder="ivan@example.com"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              {...register('email')}
            />
            {errors.email && <p className="mt-1 text-xs text-red-600">{errors.email.message}</p>}
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Дата рождения</label>
          <input
            type="date"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            {...register('date_of_birth')}
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Примечание</label>
          <textarea
            rows={2}
            placeholder="Аллергии, особенности..."
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
            {...register('comment')}
          />
        </div>

        <div className="flex justify-end pt-1">
          <button
            type="submit"
            disabled={mut.isPending}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-60 transition-colors"
          >
            {mut.isPending ? 'Сохранение...' : 'Добавить пациента'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

// ─── PatientsPage ─────────────────────────────────────────────────────────────

export default function PatientsPage() {
  const navigate = useNavigate()
  const qc = useQueryClient()

  const [rawSearch, setRawSearch] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(0)
  const [sourceFilter, setSourceFilter] = useState('')
  const [createOpen, setCreateOpen] = useState(false)

  useEffect(() => {
    setPage(0)
    const t = setTimeout(() => setSearch(rawSearch), 400)
    return () => clearTimeout(t)
  }, [rawSearch])

  useEffect(() => { setPage(0) }, [sourceFilter])

  const { data: patients = [], isLoading } = useQuery({
    queryKey: ['patients', search, page, sourceFilter],
    queryFn: () =>
      getPatients({
        search: search || undefined,
        source: sourceFilter || undefined,
        limit: LIMIT,
        offset: page * LIMIT,
      }),
  })

  const hasPrev = page > 0
  const hasNext = patients.length === LIMIT

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Пациенты</h1>
          <p className="text-sm text-gray-500 mt-0.5">База пациентов клиники</p>
        </div>
        <button
          onClick={() => setCreateOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
        >
          <Plus size={16} />
          Добавить пациента
        </button>
      </div>

      {/* Search + filter row */}
      <div className="flex gap-3 mb-5 flex-wrap">
        <div className="relative flex-1 min-w-[200px]">
          <Search
            size={16}
            className="absolute left-3.5 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none"
          />
          <input
            type="text"
            value={rawSearch}
            onChange={(e) => setRawSearch(e.target.value)}
            placeholder="Поиск по имени, телефону или email..."
            className="w-full pl-10 pr-10 py-2.5 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
          />
          {rawSearch && (
            <button
              type="button"
              onClick={() => setRawSearch('')}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"
            >
              <X size={15} />
            </button>
          )}
        </div>

        <select
          value={sourceFilter}
          onChange={(e) => setSourceFilter(e.target.value)}
          className="border border-gray-300 rounded-lg px-3 py-2.5 text-sm text-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
        >
          <option value="">Все источники</option>
          <option value="admin_panel">Администратор</option>
          <option value="telegram_bot">Telegram bot</option>
        </select>
      </div>

      {/* Patient list */}
      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        {isLoading ? (
          Array.from({ length: 5 }).map((_, i) => <PatientRowSkeleton key={i} />)
        ) : patients.length === 0 ? (
          <EmptyState hasSearch={!!search || !!sourceFilter} />
        ) : (
          patients.map((p) => (
            <PatientRow
              key={p.id}
              patient={p}
              onClick={() => navigate(`/admin/patients/${p.id}`)}
            />
          ))
        )}
      </div>

      {/* Pagination */}
      {(hasPrev || hasNext) && (
        <div className="flex items-center justify-between mt-4">
          <button
            type="button"
            onClick={() => setPage((p) => p - 1)}
            disabled={!hasPrev}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            ← Назад
          </button>
          <span className="text-sm text-gray-500">Страница {page + 1}</span>
          <button
            type="button"
            onClick={() => setPage((p) => p + 1)}
            disabled={!hasNext}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            Вперёд →
          </button>
        </div>
      )}

      <CreatePatientModal
        isOpen={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={() => {
          setCreateOpen(false)
          qc.invalidateQueries({ queryKey: ['patients'] })
        }}
      />
    </div>
  )
}
