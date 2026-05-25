import { useState, useEffect } from 'react'
import { Plus, Pencil, Trash2, ExternalLink, Phone } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import toast from 'react-hot-toast'
import Modal from '../../components/Modal'
import ConfirmDialog from '../../components/ConfirmDialog'

const STORAGE_KEY = 'referrers_v1'

const TYPES = [
  { value: 'doctor',  label: 'Врач' },
  { value: 'clinic',  label: 'Клиника' },
  { value: 'lab',     label: 'Лаборатория' },
  { value: 'other',   label: 'Другое' },
]

const TYPE_LABEL = Object.fromEntries(TYPES.map(({ value, label }) => [value, label]))

const schema = z.object({
  name:                  z.string().min(1, 'Введите имя или название'),
  phone:                 z.string().optional(),
  type:                  z.enum(['doctor', 'clinic', 'lab', 'other']),
  commission_service_pct: z
    .number({ invalid_type_error: 'Введите число' })
    .min(0)
    .max(100)
    .default(0),
  commission_lab_pct:    z
    .number({ invalid_type_error: 'Введите число' })
    .min(0)
    .max(100)
    .default(0),
  notes:                 z.string().optional(),
})

function loadReferrers() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : []
  } catch {
    return []
  }
}

function saveReferrers(list) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(list))
}

let _nextId = null
function nextId(list) {
  if (_nextId === null) _nextId = list.reduce((m, r) => Math.max(m, r.id ?? 0), 0) + 1
  return _nextId++
}

function ReferrerForm({ defaultValues, onSubmit, isLoading }) {
  const { register, handleSubmit, formState: { errors } } = useForm({
    resolver: zodResolver(schema),
    defaultValues: defaultValues ?? {
      name: '', phone: '', type: 'doctor',
      commission_service_pct: 0, commission_lab_pct: 0, notes: '',
    },
  })

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Имя / Организация <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            placeholder="Иванов А.П."
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            {...register('name')}
          />
          {errors.name && <p className="mt-1 text-xs text-red-600">{errors.name.message}</p>}
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Тип</label>
          <select
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            {...register('type')}
          >
            {TYPES.map(({ value, label }) => (
              <option key={value} value={value}>{label}</option>
            ))}
          </select>
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Телефон</label>
        <input
          type="tel"
          placeholder="+7 (999) 000-00-00"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          {...register('phone')}
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Комиссия за услуги (%)
          </label>
          <input
            type="number"
            min="0"
            max="100"
            step="0.5"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            {...register('commission_service_pct', { valueAsNumber: true })}
          />
          {errors.commission_service_pct && (
            <p className="mt-1 text-xs text-red-600">{errors.commission_service_pct.message}</p>
          )}
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Комиссия за лабораторию (%)
          </label>
          <input
            type="number"
            min="0"
            max="100"
            step="0.5"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            {...register('commission_lab_pct', { valueAsNumber: true })}
          />
          {errors.commission_lab_pct && (
            <p className="mt-1 text-xs text-red-600">{errors.commission_lab_pct.message}</p>
          )}
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Примечание</label>
        <textarea
          rows={2}
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
          {...register('notes')}
        />
      </div>

      <div className="flex justify-end pt-1">
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-60 transition-colors"
        >
          {isLoading ? 'Сохранение...' : 'Сохранить'}
        </button>
      </div>
    </form>
  )
}

function TypeBadge({ type }) {
  const colors = {
    doctor: 'bg-blue-50 text-blue-700 border-blue-200',
    clinic: 'bg-violet-50 text-violet-700 border-violet-200',
    lab:    'bg-cyan-50 text-cyan-700 border-cyan-200',
    other:  'bg-gray-50 text-gray-600 border-gray-200',
  }
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border ${colors[type] ?? colors.other}`}>
      {TYPE_LABEL[type] ?? type}
    </span>
  )
}

export default function ReferrersPage() {
  const [referrers, setReferrers] = useState(loadReferrers)
  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)

  useEffect(() => { _nextId = null }, [])

  const persist = (list) => {
    setReferrers(list)
    saveReferrers(list)
  }

  const handleCreate = (data) => {
    const newItem = { ...data, id: nextId(referrers), created_at: new Date().toISOString() }
    persist([...referrers, newItem])
    setCreateOpen(false)
    toast.success('Направитель добавлен')
  }

  const handleUpdate = (data) => {
    persist(referrers.map((r) => (r.id === editTarget.id ? { ...r, ...data } : r)))
    setEditTarget(null)
    toast.success('Направитель обновлён')
  }

  const handleDelete = () => {
    persist(referrers.filter((r) => r.id !== deleteTarget.id))
    setDeleteTarget(null)
    toast.success('Направитель удалён')
  }

  return (
    <div className="p-6 lg:p-8">
      <div className="mb-5 flex items-start justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Внешние направители</h1>
          <p className="text-sm text-gray-500 mt-0.5">
            Врачи, клиники и партнёры, направляющие пациентов
          </p>
        </div>
        <button
          onClick={() => setCreateOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
        >
          <Plus size={16} />
          Добавить направителя
        </button>
      </div>

      <div className="mb-3 px-1">
        <p className="text-xs text-amber-600 bg-amber-50 border border-amber-200 rounded-lg px-3 py-2 inline-block">
          MVP: данные хранятся локально. Интеграция с отчётами — в v0.3.
        </p>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        {referrers.length === 0 ? (
          <div className="p-10 text-center">
            <ExternalLink size={36} strokeWidth={1.25} className="mx-auto mb-3 text-gray-300" />
            <p className="text-sm font-medium text-gray-500">Направителей ещё нет</p>
            <p className="text-xs text-gray-400 mt-1">Добавьте врачей и клиники, которые направляют пациентов</p>
          </div>
        ) : (
          <table className="w-full text-left">
            <thead>
              <tr className="border-b border-gray-200 bg-gray-50/80">
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide">Направитель</th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide">Тип</th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide">Телефон</th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide text-right">Комиссия (услуги)</th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide text-right">Комиссия (лаб.)</th>
                <th className="px-4 py-2.5 w-20" />
              </tr>
            </thead>
            <tbody>
              {referrers.map((r) => (
                <tr key={r.id} className="border-t border-gray-100 hover:bg-gray-50 transition-colors">
                  <td className="px-4 py-3">
                    <p className="text-sm font-medium text-gray-900">{r.name}</p>
                    {r.notes && (
                      <p className="text-xs text-gray-400 mt-0.5 max-w-xs truncate">{r.notes}</p>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <TypeBadge type={r.type} />
                  </td>
                  <td className="px-4 py-3">
                    {r.phone ? (
                      <a
                        href={`tel:${r.phone}`}
                        className="flex items-center gap-1.5 text-xs text-gray-600 hover:text-blue-600 transition-colors"
                      >
                        <Phone size={12} />
                        {r.phone}
                      </a>
                    ) : (
                      <span className="text-xs text-gray-400">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-700 text-right">
                    {r.commission_service_pct > 0 ? `${r.commission_service_pct}%` : '—'}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-700 text-right">
                    {r.commission_lab_pct > 0 ? `${r.commission_lab_pct}%` : '—'}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1 justify-end">
                      <button
                        onClick={() => setEditTarget(r)}
                        className="p-1.5 text-gray-400 hover:text-blue-600 rounded transition-colors"
                      >
                        <Pencil size={14} />
                      </button>
                      <button
                        onClick={() => setDeleteTarget(r)}
                        className="p-1.5 text-gray-400 hover:text-red-600 rounded transition-colors"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <Modal isOpen={createOpen} onClose={() => setCreateOpen(false)} title="Новый направитель">
        <ReferrerForm onSubmit={handleCreate} />
      </Modal>

      <Modal isOpen={!!editTarget} onClose={() => setEditTarget(null)} title="Редактировать направителя">
        {editTarget && (
          <ReferrerForm
            key={editTarget.id}
            defaultValues={{
              name: editTarget.name,
              phone: editTarget.phone ?? '',
              type: editTarget.type,
              commission_service_pct: editTarget.commission_service_pct ?? 0,
              commission_lab_pct: editTarget.commission_lab_pct ?? 0,
              notes: editTarget.notes ?? '',
            }}
            onSubmit={handleUpdate}
          />
        )}
      </Modal>

      <ConfirmDialog
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Удалить направителя"
        message={`«${deleteTarget?.name}» будет удалён из списка.`}
        confirmLabel="Удалить"
      />
    </div>
  )
}
