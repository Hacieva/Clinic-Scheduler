import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Pencil, Trash2, ExternalLink } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import toast from 'react-hot-toast'
import {
  getDoctors,
  createDoctor,
  updateDoctor,
  deleteDoctor,
  setDoctorDirections,
} from '../../api/doctors'
import { getDirections } from '../../api/directions'
import DataTable from '../../components/DataTable'
import Modal from '../../components/Modal'
import ConfirmDialog from '../../components/ConfirmDialog'
import Badge from '../../components/Badge'

const schema = z.object({
  last_name: z.string().min(1, 'Введите фамилию'),
  first_name: z.string().min(1, 'Введите имя'),
  middle_name: z.string().optional(),
  cabinet: z.string().optional(),
  branch_address: z.string().optional(),
  description: z.string().optional(),
  direction_ids: z.array(z.number()).default([]),
})

function toPayload(data) {
  return {
    first_name: data.first_name,
    last_name: data.last_name,
    ...(data.middle_name && { middle_name: data.middle_name }),
    ...(data.cabinet && { cabinet: data.cabinet }),
    ...(data.branch_address && { branch_address: data.branch_address }),
    ...(data.description && { description: data.description }),
  }
}

function DoctorForm({ defaultValues, allDirections, onSubmit, isLoading }) {
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(schema),
    defaultValues: defaultValues ?? { direction_ids: [] },
  })

  const selectedIds = watch('direction_ids') ?? []

  const toggleDirection = (id) => {
    if (selectedIds.includes(id)) {
      setValue(
        'direction_ids',
        selectedIds.filter((d) => d !== id),
        { shouldValidate: true },
      )
    } else {
      setValue('direction_ids', [...selectedIds, id], { shouldValidate: true })
    }
  }

  const activeDirections = allDirections.filter((d) => d.is_active)

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Фамилия <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            {...register('last_name')}
          />
          {errors.last_name && (
            <p className="mt-1 text-xs text-red-600">{errors.last_name.message}</p>
          )}
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Имя <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            {...register('first_name')}
          />
          {errors.first_name && (
            <p className="mt-1 text-xs text-red-600">{errors.first_name.message}</p>
          )}
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Отчество</label>
        <input
          type="text"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          {...register('middle_name')}
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Кабинет</label>
          <input
            type="text"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            {...register('cabinet')}
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Адрес филиала
          </label>
          <input
            type="text"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            {...register('branch_address')}
          />
        </div>
      </div>

      {activeDirections.length > 0 && (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Направления
          </label>
          <div className="space-y-1 max-h-32 overflow-y-auto border border-gray-200 rounded-lg p-2">
            {activeDirections.map((d) => (
              <label
                key={d.id}
                className="flex items-center gap-2 cursor-pointer py-0.5 hover:bg-gray-50 px-1 rounded"
              >
                <input
                  type="checkbox"
                  className="rounded"
                  checked={selectedIds.includes(d.id)}
                  onChange={() => toggleDirection(d.id)}
                />
                <span className="text-sm text-gray-700">{d.name}</span>
              </label>
            ))}
          </div>
        </div>
      )}

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Описание</label>
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

function fullName(row) {
  return [row.last_name, row.first_name, row.middle_name].filter(Boolean).join(' ')
}

export default function DoctorsPage() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)

  const { data: doctors = [], isLoading } = useQuery({
    queryKey: ['doctors'],
    queryFn: getDoctors,
  })

  const { data: allDirections = [] } = useQuery({
    queryKey: ['directions'],
    queryFn: getDirections,
  })

  const createMut = useMutation({
    mutationFn: async ({ direction_ids, ...rest }) => {
      const doctor = await createDoctor(toPayload(rest))
      if (direction_ids?.length > 0) {
        await setDoctorDirections(doctor.id, direction_ids)
      }
      return doctor
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['doctors'] })
      setCreateOpen(false)
      toast.success('Врач создан')
    },
    onError: () => toast.error('Не удалось создать врача'),
  })

  const updateMut = useMutation({
    mutationFn: async ({ id, direction_ids, ...rest }) => {
      await updateDoctor(id, toPayload(rest))
      await setDoctorDirections(id, direction_ids ?? [])
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['doctors'] })
      setEditTarget(null)
      toast.success('Врач обновлён')
    },
    onError: () => toast.error('Не удалось обновить врача'),
  })

  const deleteMut = useMutation({
    mutationFn: (id) => deleteDoctor(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['doctors'] })
      setDeleteTarget(null)
      toast.success('Врач деактивирован')
    },
    onError: () => toast.error('Не удалось деактивировать врача'),
  })

  const columns = [
    {
      key: 'name',
      label: 'Врач',
      render: (row) => <span className="font-medium">{fullName(row)}</span>,
    },
    {
      key: 'directions',
      label: 'Направления',
      render: (row) => {
        const dirs = row.directions ?? []
        if (dirs.length === 0) return <span className="text-gray-400">—</span>
        const shown = dirs.slice(0, 2)
        const rest = dirs.length - 2
        return (
          <div className="flex flex-wrap gap-1">
            {shown.map((d) => (
              <Badge key={d.id} variant="active">
                {d.name}
              </Badge>
            ))}
            {rest > 0 && <Badge variant="inactive">+{rest}</Badge>}
          </div>
        )
      },
    },
    {
      key: 'cabinet',
      label: 'Кабинет',
      render: (row) => row.cabinet ?? '—',
    },
    {
      key: 'is_active',
      label: 'Статус',
      render: (row) => (
        <Badge variant={row.is_active ? 'active' : 'inactive'}>
          {row.is_active ? 'Активен' : 'Неактивен'}
        </Badge>
      ),
    },
    {
      key: 'actions',
      label: '',
      render: (row) => (
        <div className="flex items-center gap-2 justify-end">
          <button
            onClick={() => navigate(`/admin/doctors/${row.id}`)}
            className="p-1.5 text-gray-400 hover:text-gray-700 rounded transition-colors"
            title="Открыть"
          >
            <ExternalLink size={15} />
          </button>
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

  const editDefaultValues = editTarget
    ? {
        last_name: editTarget.last_name,
        first_name: editTarget.first_name,
        middle_name: editTarget.middle_name ?? '',
        cabinet: editTarget.cabinet ?? '',
        branch_address: editTarget.branch_address ?? '',
        description: editTarget.description ?? '',
        direction_ids: editTarget.directions?.map((d) => d.id) ?? [],
      }
    : undefined

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Врачи</h1>
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
        data={doctors}
        loading={isLoading}
        emptyText="Врачей пока нет"
      />

      <Modal
        isOpen={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Новый врач"
        maxWidth="max-w-lg"
      >
        <DoctorForm
          allDirections={allDirections}
          onSubmit={(data) => createMut.mutate(data)}
          isLoading={createMut.isPending}
        />
      </Modal>

      <Modal
        isOpen={!!editTarget}
        onClose={() => setEditTarget(null)}
        title="Редактировать врача"
        maxWidth="max-w-lg"
      >
        {editTarget && (
          <DoctorForm
            defaultValues={editDefaultValues}
            allDirections={allDirections}
            onSubmit={(data) => updateMut.mutate({ id: editTarget.id, ...data })}
            isLoading={updateMut.isPending}
          />
        )}
      </Modal>

      <ConfirmDialog
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => deleteMut.mutate(deleteTarget.id)}
        title="Деактивировать врача"
        message={`Врач ${deleteTarget ? fullName(deleteTarget) : ''} будет деактивирован. Записи сохранятся.`}
        confirmLabel="Деактивировать"
        isLoading={deleteMut.isPending}
      />
    </div>
  )
}
