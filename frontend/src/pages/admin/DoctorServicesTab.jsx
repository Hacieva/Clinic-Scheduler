import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Pencil, Trash2 } from 'lucide-react'
import toast from 'react-hot-toast'
import {
  getDoctorServices,
  createDoctorService,
  updateDoctorService,
  deleteDoctorService,
} from '../../api/services'
import DataTable from '../../components/DataTable'
import Modal from '../../components/Modal'
import ConfirmDialog from '../../components/ConfirmDialog'
import Badge from '../../components/Badge'

const schema = z.object({
  direction_id: z
    .number({ invalid_type_error: 'Выберите направление' })
    .positive('Выберите направление'),
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

function ServiceForm({ doctorDirections, defaultValues, onSubmit, isLoading }) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(schema),
    defaultValues,
  })

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Направление <span className="text-red-500">*</span>
        </label>
        <select
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          {...register('direction_id', { valueAsNumber: true })}
        >
          <option value="">Выберите направление</option>
          {doctorDirections.map((d) => (
            <option key={d.id} value={d.id}>
              {d.name}
            </option>
          ))}
        </select>
        {errors.direction_id && (
          <p className="mt-1 text-xs text-red-600">{errors.direction_id.message}</p>
        )}
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Название <span className="text-red-500">*</span>
        </label>
        <input
          type="text"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
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
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            {...register('duration_minutes', { valueAsNumber: true })}
          />
          {errors.duration_minutes && (
            <p className="mt-1 text-xs text-red-600">{errors.duration_minutes.message}</p>
          )}
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Цена (₽)
          </label>
          <input
            type="text"
            placeholder="1500"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            {...register('price_rub')}
          />
        </div>
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Описание</label>
        <textarea
          rows={2}
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
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

export default function DoctorServicesTab({ doctorId, doctorDirections }) {
  const qc = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)

  const dirMap = Object.fromEntries(doctorDirections.map((d) => [d.id, d.name]))

  const { data: services = [], isLoading } = useQuery({
    queryKey: ['services', doctorId],
    queryFn: () => getDoctorServices(doctorId),
  })

  const createMut = useMutation({
    mutationFn: ({ price_rub, ...rest }) =>
      createDoctorService(doctorId, {
        ...rest,
        ...(price_rub ? { price: toKopecks(price_rub) } : {}),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['services', doctorId] })
      setCreateOpen(false)
      toast.success('Услуга создана')
    },
    onError: (err) => {
      if (err?.response?.status === 422) {
        toast.error('Направление не принадлежит врачу')
      } else {
        toast.error('Не удалось создать услугу')
      }
    },
  })

  const updateMut = useMutation({
    mutationFn: ({ price_rub, ...rest }) =>
      updateDoctorService(doctorId, editTarget.id, {
        ...rest,
        ...(price_rub ? { price: toKopecks(price_rub) } : {}),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['services', doctorId] })
      setEditTarget(null)
      toast.success('Услуга обновлена')
    },
    onError: () => toast.error('Не удалось обновить услугу'),
  })

  const deleteMut = useMutation({
    mutationFn: (serviceId) => deleteDoctorService(doctorId, serviceId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['services', doctorId] })
      setDeleteTarget(null)
      toast.success('Услуга деактивирована')
    },
    onError: () => toast.error('Не удалось деактивировать услугу'),
  })

  const columns = [
    {
      key: 'direction',
      label: 'Направление',
      render: (row) => dirMap[row.direction_id] ?? '—',
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
      key: 'is_active',
      label: 'Статус',
      render: (row) => (
        <Badge variant={row.is_active ? 'active' : 'inactive'}>
          {row.is_active ? 'Активна' : 'Неактивна'}
        </Badge>
      ),
    },
    {
      key: 'actions',
      label: '',
      render: (row) => (
        <div className="flex items-center gap-2 justify-end">
          <button
            onClick={() => setEditTarget(row)}
            className="p-1.5 text-gray-400 hover:text-blue-600 rounded transition-colors"
          >
            <Pencil size={15} />
          </button>
          <button
            onClick={() => setDeleteTarget(row)}
            className="p-1.5 text-gray-400 hover:text-red-600 rounded transition-colors"
          >
            <Trash2 size={15} />
          </button>
        </div>
      ),
    },
  ]

  const editDefaults = editTarget
    ? {
        direction_id: editTarget.direction_id,
        name: editTarget.name,
        description: editTarget.description ?? '',
        duration_minutes: editTarget.duration_minutes,
        price_rub: editTarget.price != null ? String(editTarget.price / 100) : '',
      }
    : undefined

  return (
    <div>
      <div className="flex justify-end mb-4">
        <button
          onClick={() => setCreateOpen(true)}
          disabled={doctorDirections.length === 0}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition-colors"
          title={doctorDirections.length === 0 ? 'Сначала добавьте направления врачу' : undefined}
        >
          <Plus size={16} />
          Добавить услугу
        </button>
      </div>

      {doctorDirections.length === 0 && (
        <p className="text-sm text-amber-600 bg-amber-50 border border-amber-200 rounded-lg px-4 py-3 mb-4">
          У врача нет направлений. Сначала добавьте направления на вкладке «Информация».
        </p>
      )}

      <DataTable
        columns={columns}
        data={services}
        loading={isLoading}
        emptyText="Услуг пока нет"
      />

      <Modal
        isOpen={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Новая услуга"
      >
        <ServiceForm
          doctorDirections={doctorDirections}
          onSubmit={(data) => createMut.mutate(data)}
          isLoading={createMut.isPending}
        />
      </Modal>

      <Modal
        isOpen={!!editTarget}
        onClose={() => setEditTarget(null)}
        title="Редактировать услугу"
      >
        {editTarget && (
          <ServiceForm
            key={editTarget.id}
            doctorDirections={doctorDirections}
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
        message={`Услуга «${deleteTarget?.name}» будет деактивирована.`}
        confirmLabel="Деактивировать"
        isLoading={deleteMut.isPending}
      />
    </div>
  )
}
