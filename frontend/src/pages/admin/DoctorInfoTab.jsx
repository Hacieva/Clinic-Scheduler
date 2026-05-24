import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Pencil } from 'lucide-react'
import toast from 'react-hot-toast'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { updateDoctor, setDoctorDirections } from '../../api/doctors'
import { getBranches } from '../../api/branches'
import Badge from '../../components/Badge'

const schema = z.object({
  last_name: z.string().min(1, 'Введите фамилию'),
  first_name: z.string().min(1, 'Введите имя'),
  middle_name: z.string().optional(),
  phone: z.string().optional(),
  cabinet: z.string().optional(),
  branch_id: z.number().nullable().optional(),
  description: z.string().optional(),
  direction_ids: z.array(z.number()).default([]),
})

function toPayload(data) {
  return {
    first_name: data.first_name,
    last_name: data.last_name,
    ...(data.middle_name && { middle_name: data.middle_name }),
    ...(data.phone && { phone: data.phone }),
    ...(data.cabinet && { cabinet: data.cabinet }),
    ...(data.branch_id && { branch_id: data.branch_id }),
    ...(data.description && { description: data.description }),
  }
}

function EditForm({ doctor, allDirections, allBranches, onCancel, onSaved }) {
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(schema),
    defaultValues: {
      last_name: doctor.last_name,
      first_name: doctor.first_name,
      middle_name: doctor.middle_name ?? '',
      phone: doctor.phone ?? '',
      cabinet: doctor.cabinet ?? '',
      branch_id: doctor.branch_id ?? null,
      description: doctor.description ?? '',
      direction_ids: doctor.directions?.map((d) => d.id) ?? [],
    },
  })

  const selectedIds = watch('direction_ids') ?? []

  const toggle = (id) => {
    if (selectedIds.includes(id)) {
      setValue('direction_ids', selectedIds.filter((d) => d !== id))
    } else {
      setValue('direction_ids', [...selectedIds, id])
    }
  }

  return (
    <form onSubmit={handleSubmit(onSaved)} className="space-y-4 max-w-lg">
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
          <label className="block text-sm font-medium text-gray-700 mb-1">Телефон</label>
          <input
            type="text"
            placeholder="+7 999 000 00 00"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            {...register('phone')}
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Кабинет</label>
          <input
            type="text"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            {...register('cabinet')}
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Филиал</label>
        <select
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          {...register('branch_id', { valueAsNumber: true })}
        >
          <option value="">Не выбрано</option>
          {allBranches.filter((b) => b.is_active).map((b) => (
            <option key={b.id} value={b.id}>{b.name}</option>
          ))}
        </select>
      </div>

      {allDirections.filter((d) => d.is_active).length > 0 && (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">Направления</label>
          <div className="space-y-1 max-h-32 overflow-y-auto border border-gray-200 rounded-lg p-2">
            {allDirections
              .filter((d) => d.is_active)
              .map((d) => (
                <label
                  key={d.id}
                  className="flex items-center gap-2 cursor-pointer py-0.5 hover:bg-gray-50 px-1 rounded"
                >
                  <input
                    type="checkbox"
                    className="rounded"
                    checked={selectedIds.includes(d.id)}
                    onChange={() => toggle(d.id)}
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

      <div className="flex gap-3">
        <button
          type="submit"
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
        >
          Сохранить
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
        >
          Отмена
        </button>
      </div>
    </form>
  )
}

function InfoRow({ label, value }) {
  return (
    <div>
      <dt className="text-xs font-medium text-gray-500 uppercase tracking-wide">{label}</dt>
      <dd className="mt-0.5 text-sm text-gray-900">{value || '—'}</dd>
    </div>
  )
}

export default function DoctorInfoTab({ doctor, doctorId, allDirections }) {
  const [editing, setEditing] = useState(false)
  const qc = useQueryClient()

  const { data: allBranches = [] } = useQuery({
    queryKey: ['branches'],
    queryFn: getBranches,
  })

  const branch = allBranches.find((b) => b.id === doctor.branch_id)

  const mut = useMutation({
    mutationFn: async ({ direction_ids, ...rest }) => {
      await updateDoctor(doctorId, toPayload(rest))
      await setDoctorDirections(doctorId, direction_ids ?? [])
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['doctor', doctorId] })
      setEditing(false)
      toast.success('Данные сохранены')
    },
    onError: () => toast.error('Не удалось сохранить изменения'),
  })

  if (editing) {
    return (
      <EditForm
        doctor={doctor}
        allDirections={allDirections}
        allBranches={allBranches}
        onCancel={() => setEditing(false)}
        onSaved={(data) => mut.mutate(data)}
      />
    )
  }

  return (
    <div className="max-w-lg">
      <div className="flex justify-end mb-4">
        <button
          onClick={() => setEditing(true)}
          className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
        >
          <Pencil size={14} />
          Редактировать
        </button>
      </div>
      <dl className="space-y-4">
        <InfoRow label="Фамилия" value={doctor.last_name} />
        <InfoRow label="Имя" value={doctor.first_name} />
        <InfoRow label="Отчество" value={doctor.middle_name} />
        <InfoRow label="Телефон" value={doctor.phone} />
        <InfoRow label="Кабинет" value={doctor.cabinet} />
        <InfoRow label="Филиал" value={branch?.name} />
        <div>
          <dt className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-1">
            Направления
          </dt>
          <dd>
            {doctor.directions?.length ? (
              <div className="flex flex-wrap gap-1">
                {doctor.directions.map((d) => (
                  <Badge key={d.id} variant="active">
                    {d.name}
                  </Badge>
                ))}
              </div>
            ) : (
              <span className="text-sm text-gray-400">—</span>
            )}
          </dd>
        </div>
        {doctor.description && (
          <div>
            <dt className="text-xs font-medium text-gray-500 uppercase tracking-wide">Описание</dt>
            <dd className="mt-0.5 text-sm text-gray-900 whitespace-pre-wrap">{doctor.description}</dd>
          </div>
        )}
      </dl>
    </div>
  )
}
