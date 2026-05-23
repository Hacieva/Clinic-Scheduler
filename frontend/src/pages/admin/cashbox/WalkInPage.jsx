import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  ArrowLeft, Search, UserRound, Plus, Trash2,
  CreditCard, Banknote, Smartphone, AlertCircle,
} from 'lucide-react'
import { getPatients } from '../../../api/patients'
import { getDoctors } from '../../../api/doctors'
import { getDoctorServices } from '../../../api/services'

function formatPrice(kopecks) {
  if (!kopecks && kopecks !== 0) return '—'
  return `${(kopecks / 100).toLocaleString('ru-RU')} ₽`
}

function doctorFullName(d) {
  return [d.last_name, d.first_name].filter(Boolean).join(' ')
}

export default function WalkInPage() {
  const navigate = useNavigate()

  // ── Patient search ────────────────────────────────────────────────────────
  const [search, setSearch] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedPatient, setSelectedPatient] = useState(null)
  const [showResults, setShowResults] = useState(false)

  useEffect(() => {
    const t = setTimeout(() => setSearchQuery(search.trim()), 300)
    return () => clearTimeout(t)
  }, [search])

  const { data: searchResults = [] } = useQuery({
    queryKey: ['patient-search', searchQuery],
    queryFn: () => getPatients({ search: searchQuery, limit: 5 }),
    enabled: searchQuery.length >= 2,
  })

  // ── Service items ─────────────────────────────────────────────────────────
  const [items, setItems] = useState([])
  const [newItem, setNewItem] = useState({ doctor_id: '', service_id: '' })

  const { data: doctors = [] } = useQuery({
    queryKey: ['doctors'],
    queryFn: getDoctors,
  })
  const activeDoctors = doctors.filter((d) => d.is_active)

  const { data: doctorServices = [] } = useQuery({
    queryKey: ['doctor-services', newItem.doctor_id],
    queryFn: () => getDoctorServices(newItem.doctor_id),
    enabled: !!newItem.doctor_id,
  })
  const activeServices = doctorServices.filter((s) => s.is_active)

  const addItem = () => {
    if (!newItem.doctor_id || !newItem.service_id) return
    const doctor = activeDoctors.find((d) => String(d.id) === newItem.doctor_id)
    const service = activeServices.find((s) => String(s.id) === newItem.service_id)
    if (!doctor || !service) return
    setItems((prev) => [
      ...prev,
      {
        doctor_id: doctor.id,
        doctor_name: doctorFullName(doctor),
        service_id: service.id,
        service_name: service.name,
        price: service.price ?? 0,
        quantity: 1,
      },
    ])
    setNewItem((prev) => ({ ...prev, service_id: '' }))
  }

  const removeItem = (idx) => setItems((prev) => prev.filter((_, i) => i !== idx))

  const subtotal = items.reduce((s, i) => s + i.price * i.quantity, 0)

  // ── Payment method ────────────────────────────────────────────────────────
  const [payMethod, setPayMethod] = useState('cash')

  return (
    <div className="p-6 lg:p-8 max-w-2xl mx-auto">
      {/* Back */}
      <button
        type="button"
        onClick={() => navigate('/admin/cashbox')}
        className="flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-900 transition-colors mb-5"
      >
        <ArrowLeft size={15} />
        Касса
      </button>

      <h1 className="text-2xl font-semibold text-gray-900 mb-6">Walk-in визит</h1>

      {/* v0.3 notice */}
      <div className="flex items-start gap-3 bg-amber-50 border border-amber-200 rounded-xl p-4 mb-6">
        <AlertCircle size={15} className="text-amber-500 shrink-0 mt-0.5" />
        <div>
          <p className="text-sm font-medium text-amber-800">Кассовый модуль — v0.3</p>
          <p className="text-xs text-amber-700 mt-0.5">
            Провести оплату и открыть визит можно будет после запуска v0.3. Сейчас вы можете собрать состав визита.
          </p>
        </div>
      </div>

      {/* ── PATIENT ─────────────────────────────────────────────────────────── */}
      <div className="bg-white rounded-xl border border-gray-200 p-5 mb-4">
        <h2 className="text-sm font-semibold text-gray-700 mb-3">Пациент</h2>

        {selectedPatient ? (
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-full bg-blue-500 flex items-center justify-center text-white text-sm font-bold shrink-0">
              {selectedPatient.full_name.trim()[0]}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-semibold text-gray-900">{selectedPatient.full_name}</p>
              <p className="text-xs text-gray-400">{selectedPatient.phone}</p>
            </div>
            <button
              onClick={() => { setSelectedPatient(null); setSearch('') }}
              className="text-xs text-gray-400 hover:text-gray-700 transition-colors px-2 py-1 rounded hover:bg-gray-100"
            >
              Изменить
            </button>
          </div>
        ) : (
          <div className="relative">
            <div className="relative">
              <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none" />
              <input
                type="text"
                value={search}
                onChange={(e) => { setSearch(e.target.value); setShowResults(true) }}
                onFocus={() => setShowResults(true)}
                onBlur={() => setTimeout(() => setShowResults(false), 150)}
                placeholder="Поиск по имени или телефону…"
                className="w-full pl-9 pr-3 py-2 text-sm border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            {showResults && searchResults.length > 0 && (
              <div className="absolute z-10 top-full left-0 right-0 mt-1 bg-white border border-gray-200 rounded-lg shadow-lg divide-y divide-gray-100">
                {searchResults.map((p) => (
                  <button
                    key={p.id}
                    onMouseDown={() => { setSelectedPatient(p); setSearch('') }}
                    className="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-gray-50 transition-colors first:rounded-t-lg last:rounded-b-lg"
                  >
                    <UserRound size={15} className="text-gray-300 shrink-0" />
                    <div>
                      <p className="text-sm text-gray-900">{p.full_name}</p>
                      <p className="text-xs text-gray-400">{p.phone}</p>
                    </div>
                  </button>
                ))}
              </div>
            )}
            {searchQuery.length >= 2 && searchResults.length === 0 && (
              <p className="mt-2 text-xs text-gray-400 px-1">Пациент не найден — уточните запрос</p>
            )}
          </div>
        )}
      </div>

      {/* ── SERVICES ────────────────────────────────────────────────────────── */}
      <div className="bg-white rounded-xl border border-gray-200 p-5 mb-4">
        <h2 className="text-sm font-semibold text-gray-700 mb-3">Услуги</h2>

        {items.length > 0 && (
          <div className="mb-3 divide-y divide-gray-100 border border-gray-100 rounded-lg overflow-hidden">
            {items.map((item, idx) => (
              <div key={idx} className="flex items-center gap-3 px-3 py-2.5">
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-gray-800 truncate">{item.service_name}</p>
                  <p className="text-xs text-gray-400 truncate">{item.doctor_name}</p>
                </div>
                <span className="text-sm text-gray-700 font-medium shrink-0 tabular-nums">
                  {formatPrice(item.price)}
                </span>
                <button
                  onClick={() => removeItem(idx)}
                  className="text-gray-300 hover:text-red-500 transition-colors shrink-0 ml-1"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            ))}
          </div>
        )}

        {/* Add service row */}
        <div className="flex gap-2">
          <select
            value={newItem.doctor_id}
            onChange={(e) => setNewItem({ doctor_id: e.target.value, service_id: '' })}
            className="flex-1 min-w-0 text-sm border border-gray-300 rounded-lg px-2 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
          >
            <option value="">Врач…</option>
            {activeDoctors.map((d) => (
              <option key={d.id} value={d.id}>
                {doctorFullName(d)}
              </option>
            ))}
          </select>
          <select
            value={newItem.service_id}
            onChange={(e) => setNewItem((prev) => ({ ...prev, service_id: e.target.value }))}
            disabled={!newItem.doctor_id}
            className="flex-1 min-w-0 text-sm border border-gray-300 rounded-lg px-2 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white disabled:bg-gray-50 disabled:text-gray-400"
          >
            <option value="">Услуга…</option>
            {activeServices.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}{s.price ? ` — ${formatPrice(s.price)}` : ''}
              </option>
            ))}
          </select>
          <button
            onClick={addItem}
            disabled={!newItem.doctor_id || !newItem.service_id}
            className="flex items-center gap-1 px-3 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors shrink-0"
          >
            <Plus size={15} />
          </button>
        </div>
      </div>

      {/* ── PAYMENT ─────────────────────────────────────────────────────────── */}
      <div className="bg-white rounded-xl border border-gray-200 p-5">
        <h2 className="text-sm font-semibold text-gray-700 mb-4">Оплата</h2>

        {/* Totals */}
        <div className="flex items-center justify-between py-3 border-b border-gray-100 mb-4">
          <span className="text-sm text-gray-500">Итого к оплате</span>
          <span className="text-2xl font-bold text-gray-900 tabular-nums">
            {items.length === 0 ? '—' : formatPrice(subtotal)}
          </span>
        </div>

        {/* Payment method */}
        <div className="flex gap-2 mb-4">
          {[
            { key: 'cash',   label: 'Наличные', Icon: Banknote },
            { key: 'card',   label: 'Карта',    Icon: CreditCard },
            { key: 'online', label: 'Онлайн',   Icon: Smartphone },
          ].map(({ key, label, Icon }) => (
            <button
              key={key}
              onClick={() => setPayMethod(key)}
              className={`flex-1 flex flex-col items-center gap-1.5 py-3 rounded-xl border-2 text-xs font-medium transition-colors ${
                payMethod === key
                  ? 'border-blue-500 bg-blue-50 text-blue-700'
                  : 'border-gray-200 text-gray-500 hover:border-gray-300'
              }`}
            >
              <Icon size={18} />
              {label}
            </button>
          ))}
        </div>

        {/* Disabled submit */}
        <div className="space-y-2">
          <button
            disabled
            className="w-full py-3 text-sm font-semibold text-white bg-blue-300 rounded-xl cursor-not-allowed"
          >
            Провести оплату и открыть визит
          </button>
          <p className="text-center text-xs text-gray-400">
            Оформление оплаты будет доступно в v0.3
          </p>
        </div>
      </div>
    </div>
  )
}
