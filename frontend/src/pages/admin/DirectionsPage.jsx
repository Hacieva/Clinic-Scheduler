import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Pencil, Trash2 } from 'lucide-react'
import toast from 'react-hot-toast'
import {
  getDirections,
  createDirection,
  updateDirection,
  deleteDirection,
} from '../../api/directions'
import DataTable from '../../components/DataTable'
import Modal from '../../components/Modal'
import ConfirmDialog from '../../components/ConfirmDialog'
import Badge from '../../components/Badge'

const schema = z.object({
  name: z.string().min(1, 'Введите название'),
  description: z.string().optional(),
})

function DirectionForm({ defaultValues, onSubmit, isLoading }) {
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
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Описание
        </label>
        <textarea
          rows={3}
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

export default function DirectionsPage() {
  const qc = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)

  const { data: directions = [], isLoading } = useQuery({
    queryKey: ['directions'],
    queryFn: getDirections,
  })

  const createMut = useMutation({
    mutationFn: (data) => createDirection(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['directions'] })
      setCreateOpen(false)
      toast.success('Направление создано')
    },
    onError: () => toast.error('Не удалось создать направление'),
  })

  const updateMut = useMutation({
    mutationFn: ({ id, data }) => updateDirection(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['directions'] })
      setEditTarget(null)
      toast.success('Направление обновлено')
    },
    onError: () => toast.error('Не удалось обновить направление'),
  })

  const deleteMut = useMutation({
    mutationFn: (id) => deleteDirection(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['directions'] })
      setDeleteTarget(null)
      toast.success('Направление деактивировано')
    },
    onError: () => toast.error('Не удалось деактивировать направление'),
  })

  const columns = [
    { key: 'id', label: 'ID' },
    { key: 'name', label: 'Название' },
    {
      key: 'description',
      label: 'Описание',
      render: (row) => (
        <span className="text-gray-500">{row.description ?? '—'}</span>
      ),
    },
    {
      key: 'is_active',
      label: 'Статус',
      render: (row) => (
        <Badge variant={row.is_active ? 'active' : 'inactive'}>
          {row.is_active ? 'Активно' : 'Неактивно'}
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
            title="Редактировать"
          >
            <Pencil size={15} />
          </button>
          <button
            onClick={() => setDeleteTarget(row)}
            className="p-1.5 text-gray-400 hover:text-red-600 rounded transition-colors"
            title="Деактивировать"
          >
            <Trash2 size={15} />
          </button>
        </div>
      ),
    },
  ]

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Направления</h1>
        <button
          onClick={() => setCreateOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
        >
          <Plus size={16} />
          Добавить
        </button>
      </div>

      <DataTable
        columns={columns}
        data={directions}
        loading={isLoading}
        emptyText="Направлений пока нет"
      />

      <Modal
        isOpen={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Новое направление"
      >
        <DirectionForm
          onSubmit={(data) => createMut.mutate(data)}
          isLoading={createMut.isPending}
        />
      </Modal>

      <Modal
        isOpen={!!editTarget}
        onClose={() => setEditTarget(null)}
        title="Редактировать направление"
      >
        {editTarget && (
          <DirectionForm
            defaultValues={{
              name: editTarget.name,
              description: editTarget.description ?? '',
            }}
            onSubmit={(data) => updateMut.mutate({ id: editTarget.id, data })}
            isLoading={updateMut.isPending}
          />
        )}
      </Modal>

      <ConfirmDialog
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteMut.mutate(deleteTarget.id)}
        title="Деактивировать направление"
        message={`Направление "${deleteTarget?.name}" будет деактивировано. Врачи с этим направлением сохранятся.`}
        confirmLabel="Деактивировать"
        isLoading={deleteMut.isPending}
      />
    </div>
  )
}
