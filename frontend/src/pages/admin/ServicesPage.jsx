import { useState, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Pencil, Trash2, ChevronRight, ChevronDown, Search, Tag } from 'lucide-react'
import toast from 'react-hot-toast'
import { getAllServices, createService, updateService, deleteService } from '../../api/services'
import { getDirections } from '../../api/directions'
import Modal from '../../components/Modal'
import ConfirmDialog from '../../components/ConfirmDialog'
import Badge from '../../components/Badge'

const schema = z.object({
  direction_id: z
    .number({ invalid_type_error: 'Выберите направление' })
    .positive('Выберите направление'),
  category: z.string().optional(),
  name: z.string().min(1, 'Введите название'),
  description: z.string().optional(),
  duration_minutes: z
    .number({ invalid_type_error: 'Введите число минут' })
    .min(30, 'Минимум 30 минут')
    .refine((v) => v % 30 === 0, 'Должно быть кратно 30 минутам'),
  price_rub: z.string().optional(),
})

function toKopecks(rub) {
  if (!rub || rub === '') return undefined
  const n = parseFloat(rub)
  if (isNaN(n) || n < 0) return undefined
  return Math.round(n * 100)
}

function fmtPrice(kopecks) {
  if (kopecks == null) return '—'
  return `${(kopecks / 100).toLocaleString('ru-RU', { minimumFractionDigits: 0 })} ₽`
}

function ServiceForm({ directions, defaultValues, onSubmit, isLoading }) {
  const { register, handleSubmit, formState: { errors } } = useForm({
    resolver: zodResolver(schema),
    defaultValues,
  })

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Направление <span className="text-red-500">*</span>
          </label>
          <select
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            {...register('direction_id', { valueAsNumber: true })}
          >
            <option value="">Выберите</option>
            {directions.map((d) => (
              <option key={d.id} value={d.id}>{d.name}</option>
            ))}
          </select>
          {errors.direction_id && (
            <p className="mt-1 text-xs text-red-600">{errors.direction_id.message}</p>
          )}
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Категория</label>
          <input
            type="text"
            placeholder="Диагностика"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            {...register('category')}
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Название <span className="text-red-500">*</span>
        </label>
        <input
          type="text"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          {...register('name')}
        />
        {errors.name && (
          <p className="mt-1 text-xs text-red-600">{errors.name.message}</p>
        )}
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Длительность (мин) <span className="text-red-500">*</span>
          </label>
          <input
            type="number"
            step="30"
            min="30"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            {...register('duration_minutes', { valueAsNumber: true })}
          />
          {errors.duration_minutes && (
            <p className="mt-1 text-xs text-red-600">{errors.duration_minutes.message}</p>
          )}
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Цена (₽)</label>
          <input
            type="text"
            placeholder="1500"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            {...register('price_rub')}
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Описание</label>
        <textarea
          rows={2}
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
          {...register('description')}
        />
      </div>

      <div className="flex justify-end pt-2">
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

function CategoryRow({ category, services, onEdit, onDelete }) {
  const [open, setOpen] = useState(true)
  const Chevron = open ? ChevronDown : ChevronRight

  return (
    <>
      <tr
        className="bg-gray-50 cursor-pointer select-none hover:bg-gray-100 transition-colors"
        onClick={() => setOpen((v) => !v)}
      >
        <td colSpan={6} className="px-4 py-2.5">
          <div className="flex items-center gap-2">
            <Chevron size={13} className="text-gray-500 shrink-0" />
            <Tag size={12} className="text-gray-400 shrink-0" />
            <span className="text-sm font-semibold text-gray-700">{category}</span>
            <span className="text-xs text-gray-400 font-normal">({services.length})</span>
          </div>
        </td>
      </tr>
      {open &&
        services.map((svc) => (
          <tr key={svc.id} className="hover:bg-gray-50 border-t border-gray-100 transition-colors">
            <td className="pl-10 pr-4 py-2.5 text-xs text-gray-400 font-mono w-12">{svc.id}</td>
            <td className="px-4 py-2.5">
              <div>
                <p className="text-sm font-medium text-gray-900">{svc.name}</p>
                {svc.description && (
                  <p className="text-xs text-gray-400 mt-0.5 max-w-xs truncate">{svc.description}</p>
                )}
              </div>
            </td>
            <td className="px-4 py-2.5 text-sm text-gray-600 whitespace-nowrap">{fmtPrice(svc.price)}</td>
            <td className="px-4 py-2.5 text-sm text-gray-600 whitespace-nowrap">
              {svc.duration_minutes} мин
            </td>
            <td className="px-4 py-2.5">
              <Badge variant={svc.is_active ? 'active' : 'inactive'}>
                {svc.is_active ? 'Активна' : 'Неактивна'}
              </Badge>
            </td>
            <td className="px-4 py-2.5">
              <div className="flex items-center gap-1 justify-end">
                <button
                  onClick={() => onEdit(svc)}
                  className="p-1.5 text-gray-400 hover:text-blue-600 rounded transition-colors"
                >
                  <Pencil size={14} />
                </button>
                <button
                  onClick={() => onDelete(svc)}
                  disabled={!svc.is_active}
                  className="p-1.5 text-gray-400 hover:text-red-600 rounded transition-colors disabled:opacity-30"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            </td>
          </tr>
        ))}
    </>
  )
}

export default function ServicesPage() {
  const qc = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [showInactive, setShowInactive] = useState(false)
  const [search, setSearch] = useState('')

  const { data: services = [], isLoading } = useQuery({
    queryKey: ['catalog-services', showInactive],
    queryFn: () => getAllServices(!showInactive),
  })

  const { data: directions = [] } = useQuery({
    queryKey: ['directions'],
    queryFn: getDirections,
  })

  const invalidate = () => qc.invalidateQueries({ queryKey: ['catalog-services'] })

  const createMut = useMutation({
    mutationFn: ({ price_rub, category, ...rest }) =>
      createService({
        ...rest,
        ...(category ? { category } : {}),
        ...(price_rub ? { price: toKopecks(price_rub) } : {}),
      }),
    onSuccess: () => {
      invalidate()
      setCreateOpen(false)
      toast.success('Услуга добавлена')
    },
    onError: () => toast.error('Не удалось добавить услугу'),
  })

  const updateMut = useMutation({
    mutationFn: ({ price_rub, category, ...rest }) =>
      updateService(editTarget.id, {
        ...rest,
        ...(category ? { category } : {}),
        ...(price_rub ? { price: toKopecks(price_rub) } : {}),
      }),
    onSuccess: () => {
      invalidate()
      setEditTarget(null)
      toast.success('Услуга обновлена')
    },
    onError: () => toast.error('Не удалось обновить услугу'),
  })

  const deleteMut = useMutation({
    mutationFn: (id) => deleteService(id),
    onSuccess: () => {
      invalidate()
      setDeleteTarget(null)
      toast.success('Услуга деактивирована')
    },
    onError: () => toast.error('Не удалось деактивировать услугу'),
  })

  const filtered = useMemo(() => {
    if (!search.trim()) return services
    const q = search.toLowerCase()
    return services.filter(
      (s) =>
        s.name.toLowerCase().includes(q) ||
        (s.category ?? '').toLowerCase().includes(q) ||
        (s.description ?? '').toLowerCase().includes(q),
    )
  }, [services, search])

  const grouped = useMemo(() => {
    const map = {}
    filtered.forEach((s) => {
      const cat = s.category || 'Без категории'
      if (!map[cat]) map[cat] = []
      map[cat].push(s)
    })
    return Object.entries(map).sort(([a], [b]) => {
      if (a === 'Без категории') return 1
      if (b === 'Без категории') return -1
      return a.localeCompare(b, 'ru')
    })
  }, [filtered])

  const editDefaults = editTarget
    ? {
        direction_id: editTarget.direction_id,
        category: editTarget.category ?? '',
        name: editTarget.name,
        description: editTarget.description ?? '',
        duration_minutes: editTarget.duration_minutes,
        price_rub: editTarget.price != null ? String(editTarget.price / 100) : '',
      }
    : undefined

  return (
    <div className="p-6 lg:p-8">
      <div className="mb-5 flex items-start justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Услуги</h1>
          <p className="text-sm text-gray-500 mt-0.5">Глобальный каталог услуг клиники</p>
        </div>
        <div className="flex items-center gap-3 flex-wrap">
          <label className="flex items-center gap-2 text-sm text-gray-600 cursor-pointer select-none">
            <input
              type="checkbox"
              checked={showInactive}
              onChange={(e) => setShowInactive(e.target.checked)}
              className="rounded"
            />
            Показать неактивные
          </label>
          <button
            onClick={() => setCreateOpen(true)}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
          >
            <Plus size={16} />
            Добавить услугу
          </button>
        </div>
      </div>

      <div className="relative mb-4 max-w-sm">
        <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none" />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Поиск по названию или категории…"
          className="w-full pl-9 pr-4 py-2 text-sm border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        {isLoading ? (
          <div className="p-8 text-center text-sm text-gray-400">Загрузка...</div>
        ) : grouped.length === 0 ? (
          <div className="p-8 text-center text-sm text-gray-400">
            {search ? 'Ничего не найдено' : 'Услуг пока нет'}
          </div>
        ) : (
          <table className="w-full text-left">
            <thead>
              <tr className="border-b border-gray-200 bg-gray-50/80">
                <th className="pl-10 pr-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide w-12">
                  Код
                </th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide">
                  Название
                </th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide">
                  Цена
                </th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide">
                  Длительность
                </th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide">
                  Статус
                </th>
                <th className="px-4 py-2.5 w-20" />
              </tr>
            </thead>
            <tbody>
              {grouped.map(([category, svcs]) => (
                <CategoryRow
                  key={category}
                  category={category}
                  services={svcs}
                  onEdit={setEditTarget}
                  onDelete={setDeleteTarget}
                />
              ))}
            </tbody>
          </table>
        )}
      </div>

      <Modal isOpen={createOpen} onClose={() => setCreateOpen(false)} title="Новая услуга">
        <ServiceForm
          directions={directions}
          onSubmit={(data) => createMut.mutate(data)}
          isLoading={createMut.isPending}
        />
      </Modal>

      <Modal isOpen={!!editTarget} onClose={() => setEditTarget(null)} title="Редактировать услугу">
        {editTarget && (
          <ServiceForm
            key={editTarget.id}
            directions={directions}
            defaultValues={editDefaults}
            onSubmit={(data) => updateMut.mutate(data)}
            isLoading={updateMut.isPending}
          />
        )}
      </Modal>

      <ConfirmDialog
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteMut.mutate(deleteTarget.id)}
        title="Деактивировать услугу"
        message={`Услуга «${deleteTarget?.name}» будет деактивирована. Назначения врачей останутся.`}
        confirmLabel="Деактивировать"
        isLoading={deleteMut.isPending}
      />
    </div>
  )
}
