import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Search, ChevronRight, UserRound, Phone, Mail, X } from 'lucide-react'
import { getPatients } from '../../api/patients'

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

function PatientRow({ patient, onClick }) {
  const age = calcAge(patient.date_of_birth)
  const label = ageLabel(age)

  return (
    <button
      type="button"
      onClick={onClick}
      className="w-full flex items-center gap-4 px-5 py-4 border-b border-gray-100 last:border-0 hover:bg-blue-50/50 transition-colors text-left group"
    >
      {/* Avatar */}
      <div
        className={`w-11 h-11 rounded-full ${avatarBg(patient.id)} flex items-center justify-center text-white text-sm font-semibold shrink-0`}
      >
        {initials(patient.full_name)}
      </div>

      {/* Main info */}
      <div className="flex-1 min-w-0">
        <p className="text-sm font-semibold text-gray-900 truncate">{patient.full_name}</p>
        <div className="flex items-center gap-3 mt-0.5 flex-wrap">
          <span className="flex items-center gap-1 text-xs text-gray-500">
            <Phone size={11} />
            {patient.phone}
          </span>
          {patient.email && (
            <span className="flex items-center gap-1 text-xs text-gray-400 truncate max-w-[160px]">
              <Mail size={11} />
              {patient.email}
            </span>
          )}
        </div>
      </div>

      {/* Right meta */}
      <div className="hidden sm:flex flex-col items-end gap-1.5 shrink-0">
        <div className="flex items-center gap-2">
          {label && (
            <span className="text-xs text-gray-500 font-medium">{label}</span>
          )}
          <SourceBadge source={patient.source} />
        </div>
        <span className="text-xs text-gray-400">
          {new Date(patient.created_at).toLocaleDateString('ru-RU', {
            day: 'numeric', month: 'short', year: 'numeric',
          })}
        </span>
      </div>

      <ChevronRight
        size={16}
        className="text-gray-300 group-hover:text-blue-400 transition-colors shrink-0 ml-1"
      />
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

export default function PatientsPage() {
  const navigate = useNavigate()
  const [rawSearch, setRawSearch] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(0)

  useEffect(() => {
    setPage(0)
    const t = setTimeout(() => setSearch(rawSearch), 400)
    return () => clearTimeout(t)
  }, [rawSearch])

  const { data: patients = [], isLoading } = useQuery({
    queryKey: ['patients', search, page],
    queryFn: () =>
      getPatients({
        search: search || undefined,
        limit: LIMIT,
        offset: page * LIMIT,
      }),
  })

  const hasPrev = page > 0
  const hasNext = patients.length === LIMIT

  return (
    <div className="p-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Пациенты</h1>
          <p className="text-sm text-gray-500 mt-0.5">База пациентов клиники</p>
        </div>
      </div>

      {/* Search */}
      <div className="relative mb-5">
        <Search
          size={16}
          className="absolute left-3.5 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none"
        />
        <input
          type="text"
          value={rawSearch}
          onChange={(e) => setRawSearch(e.target.value)}
          placeholder="Поиск по имени, телефону или email..."
          className="w-full pl-10 pr-10 py-2.5 border border-gray-300 rounded-lg text-sm
                     focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent
                     bg-white"
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

      {/* Patient list */}
      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        {isLoading ? (
          Array.from({ length: 5 }).map((_, i) => <PatientRowSkeleton key={i} />)
        ) : patients.length === 0 ? (
          <EmptyState hasSearch={!!search} />
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
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300
                       rounded-lg hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed
                       transition-colors"
          >
            ← Назад
          </button>
          <span className="text-sm text-gray-500">Страница {page + 1}</span>
          <button
            type="button"
            onClick={() => setPage((p) => p + 1)}
            disabled={!hasNext}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300
                       rounded-lg hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed
                       transition-colors"
          >
            Вперёд →
          </button>
        </div>
      )}
    </div>
  )
}
