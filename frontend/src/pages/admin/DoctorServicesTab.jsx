import { useState, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, X } from 'lucide-react'
import toast from 'react-hot-toast'
import { getAssignedServices, setDoctorServices, getAllServices } from '../../api/services'
import DataTable from '../../components/DataTable'
import Modal from '../../components/Modal'
import ConfirmDialog from '../../components/ConfirmDialog'

function fmtPrice(kopecks) {
  if (kopecks == null) return '—'
  return `${(kopecks / 100).toLocaleString('ru-RU', { minimumFractionDigits: 0 })} ₽`
}

// ─── Catalog picker modal ─────────────────────────────────────────────────────

function CatalogPickerModal({ isOpen, onClose, onSave, currentIds, allServices }) {
  const [selected, setSelected] = useState(() => new Set(currentIds))
  const [search, setSearch] = useState('')

  const filtered = useMemo(() => {
    if (!search.trim()) return allServices
    const q = search.toLowerCase()
    return allServices.filter(
      (s) =>
        s.name.toLowerCase().includes(q) ||
        (s.category && s.category.toLowerCase().includes(q)),
    )
  }, [allServices, search])

  const toggle = (id) =>
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })

  const handleSave = () => {
    onSave([...selected])
    onClose()
  }

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Назначить услуги из каталога">
      <div className="space-y-3">
        <input
          type="text"
          placeholder="Поиск по названию или категории..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          autoFocus
        />

        <div className="max-h-72 overflow-y-auto border border-gray-200 rounded-lg divide-y divide-gray-100">
          {filtered.length === 0 && (
            <p className="text-sm text-gray-400 text-center py-6">Ничего не найдено</p>
          )}
          {filtered.map((s) => (
            <label key={s.id} className="flex items-center gap-3 px-3 py-2.5 cursor-pointer hover:bg-gray-50">
              <input
                type="checkbox"
                checked={selected.has(s.id)}
                onChange={() => toggle(s.id)}
                className="rounded"
              />
              <div className="flex-1 min-w-0">
                <span className="text-sm font-medium text-gray-800">{s.name}</span>
                {s.category && (
                  <span className="ml-2 text-xs text-gray-400">{s.category}</span>
                )}
              </div>
              <span className="text-xs text-gray-500 shrink-0">{fmtPrice(s.price)}</span>
            </label>
          ))}
        </div>

        <div className="flex justify-between items-center pt-1">
          <span className="text-xs text-gray-400">Выбрано: {selected.size}</span>
          <div className="flex gap-2">
            <button
              onClick={onClose}
              className="px-3 py-1.5 text-sm text-gray-600 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Отмена
            </button>
            <button
              onClick={handleSave}
              className="px-4 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
            >
              Сохранить
            </button>
          </div>
        </div>
      </div>
    </Modal>
  )
}

// ─── DoctorServicesTab ────────────────────────────────────────────────────────

export default function DoctorServicesTab({ doctorId }) {
  const qc = useQueryClient()
  const [pickerOpen, setPickerOpen] = useState(false)
  const [unassignTarget, setUnassignTarget] = useState(null)

  const { data: assigned = [], isLoading } = useQuery({
    queryKey: ['assigned-services', doctorId],
    queryFn: () => getAssignedServices(doctorId),
  })

  const { data: catalog = [] } = useQuery({
    queryKey: ['catalog-services', false],
    queryFn: () => getAllServices(true),
    enabled: pickerOpen,
  })

  const currentIds = useMemo(() => assigned.map((s) => s.id), [assigned])

  const bulkSetMut = useMutation({
    mutationFn: (ids) => setDoctorServices(doctorId, ids),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['assigned-services', doctorId] })
      toast.success('Услуги врача обновлены')
    },
    onError: () => toast.error('Не удалось обновить услуги'),
  })

  const handleSave = (ids) => bulkSetMut.mutate(ids)

  const handleUnassign = (id) => {
    const remaining = currentIds.filter((x) => x !== id)
    bulkSetMut.mutate(remaining, {
      onSuccess: () => setUnassignTarget(null),
    })
  }

  const PATIENT_TYPE_LABEL = { adult: 'Взрослые', child: 'Дети', both: 'Все' }

  const columns = [
    {
      key: 'category',
      label: 'Категория',
      render: (row) => row.category ?? <span className="text-gray-400">—</span>,
    },
    { key: 'name', label: 'Название' },
    {
      key: 'duration_minutes',
      label: 'Длит.',
      render: (row) => `${row.duration_minutes} мин`,
    },
    {
      key: 'price',
      label: 'Цена',
      render: (row) => fmtPrice(row.price),
    },
    {
      key: 'patient_type',
      label: 'Пациенты',
      render: (row) => {
        const label = PATIENT_TYPE_LABEL[row.patient_type] ?? row.patient_type ?? '—'
        return <span className="text-xs text-gray-500">{label}</span>
      },
    },
    {
      key: 'actions',
      label: '',
      render: (row) => (
        <div className="flex justify-end">
          <button
            onClick={() => setUnassignTarget(row)}
            className="p-1.5 text-gray-400 hover:text-red-600 rounded transition-colors"
            title="Убрать у врача"
          >
            <X size={15} />
          </button>
        </div>
      ),
    },
  ]

  return (
    <div>
      <div className="flex justify-end mb-4">
        <button
          onClick={() => setPickerOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
        >
          <Plus size={16} />
          Назначить из каталога
        </button>
      </div>

      <DataTable
        columns={columns}
        data={assigned}
        loading={isLoading}
        emptyText="Врачу не назначено ни одной услуги"
      />

      <CatalogPickerModal
        isOpen={pickerOpen}
        onClose={() => setPickerOpen(false)}
        onSave={handleSave}
        currentIds={currentIds}
        allServices={catalog}
      />

      <ConfirmDialog
        isOpen={!!unassignTarget}
        onClose={() => setUnassignTarget(null)}
        onConfirm={() => handleUnassign(unassignTarget.id)}
        title="Убрать услугу"
        message={`Услуга «${unassignTarget?.name}» будет убрана у врача. В каталоге она останется.`}
        confirmLabel="Убрать"
        isLoading={bulkSetMut.isPending}
      />
    </div>
  )
}
