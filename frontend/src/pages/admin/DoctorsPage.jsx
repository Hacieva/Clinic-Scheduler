import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Pencil, Trash2, ExternalLink, Users } from 'lucide-react'
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
import { getBranches } from '../../api/branches'
import Modal from '../../components/Modal'
import ConfirmDialog from '../../components/ConfirmDialog'

const schema = z.object({
  last_name: z.string().min(1, 'Введите фамилию'),
  first_name: z.string().min(1, 'Введите имя'),
  middle_name: z.string().optional(),
  phone: z.string().optional(),
  cabinet: z.string().optional(),
  branch_id: z.number().nullable().optional(),
  description: z.string().optional(),
  direction_ids: z.array(z.number()).default([]),
  create_account: z.boolean().default(false),
  email: z.string().optional(),
  password: z.string().optional(),
}).superRefine((data, ctx) => {
  if (data.create_account) {
    if (!data.email || !z.string().email().safeParse(data.email).success) {
      ctx.addIssue({ code: 'custom', path: ['email'], message: 'Введите корректный email' })
    }
    if (!data.password || data.password.length < 8) {
      ctx.addIssue({ code: 'custom', path: ['password'], message: 'Минимум 8 символов' })
    }
  }
})

function toPayload(data) {
  const payload = {
    first_name: data.first_name,
    last_name: data.last_name,
    ...(data.middle_name && { middle_name: data.middle_name }),
    ...(data.phone && { phone: data.phone }),
    ...(data.cabinet && { cabinet: data.cabinet }),
    ...(data.branch_id && { branch_id: data.branch_id }),
    ...(data.description && { description: data.description }),
  }
  if (data.create_account && data.email && data.password) {
    payload.account = { email: data.email, password: data.password }
  }
  return payload
}

function DoctorForm({ defaultValues, allDirections, allBranches, onSubmit, isLoading, isEdit }) {
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(schema),
    defaultValues: defaultValues ?? { direction_ids: [], create_account: false },
  })

  const selectedIds = watch('direction_ids') ?? []
  const createAccount = watch('create_account')

  const toggleDirection = (id) => {
    if (selectedIds.includes(id)) {
      setValue('direction_ids', selectedIds.filter((d) => d !== id), { shouldValidate: true })
    } else {
      setValue('direction_ids', [...selectedIds, id], { shouldValidate: true })
    }
  }

  const activeDirections = allDirections.filter((d) => d.is_active)

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      {/* Name */}
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

      {/* Phone + cabinet */}
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

      {/* Branch */}
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

      {/* Directions */}
      {activeDirections.length > 0 && (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">Направления</label>
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

      {/* Description */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Описание</label>
        <textarea
          rows={2}
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
          {...register('description')}
        />
      </div>

      {/* Account section — only on create */}
      {!isEdit && (
        <div className="border-t border-gray-100 pt-4">
          <label className="flex items-center gap-2 cursor-pointer mb-3">
            <input
              type="checkbox"
              className="rounded"
              {...register('create_account')}
            />
            <span className="text-sm font-medium text-gray-700">Создать аккаунт сейчас</span>
          </label>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
              <input
                type="email"
                autoComplete="off"
                disabled={!createAccount}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-50 disabled:text-gray-400"
                {...register('email')}
              />
              {errors.email && (
                <p className="mt-1 text-xs text-red-600">{errors.email.message}</p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Пароль</label>
              <input
                type="password"
                autoComplete="new-password"
                disabled={!createAccount}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-50 disabled:text-gray-400"
                {...register('password')}
              />
              {errors.password && (
                <p className="mt-1 text-xs text-red-600">{errors.password.message}</p>
              )}
            </div>
          </div>
          {createAccount && (
            <p className="mt-1.5 text-xs text-gray-500">Пароль будет показан на странице врача. Минимум 8 символов.</p>
          )}
        </div>
      )}

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

const AVATAR_COLORS = [
  'bg-blue-500', 'bg-emerald-500', 'bg-violet-500',
  'bg-amber-500', 'bg-rose-500', 'bg-cyan-500',
]
function avatarBg(id) { return AVATAR_COLORS[id % AVATAR_COLORS.length] }

function DoctorCard({ doctor, allBranches, onEdit, onDelete, onNavigate }) {
  const name = fullName(doctor)
  const initial = (doctor.last_name ?? doctor.first_name ?? '?')[0]
  const dirs = doctor.directions ?? []
  const branch = allBranches.find((b) => b.id === doctor.branch_id)
  return (
    <div className={`bg-white rounded-xl border border-gray-200 p-4 flex flex-col gap-3 ${!doctor.is_active ? 'opacity-60' : ''}`}>
      <div className="flex items-start gap-3">
        <div className={`w-10 h-10 rounded-full ${avatarBg(doctor.id)} flex items-center justify-center text-white text-sm font-bold shrink-0`}>
          {initial}
        </div>
        <div className="flex-1 min-w-0">
          <p className="font-semibold text-gray-900 text-sm leading-tight truncate">{name}</p>
          <div className="flex items-center gap-2 mt-0.5 flex-wrap">
            {doctor.cabinet && <span className="text-xs text-gray-400">Каб. {doctor.cabinet}</span>}
            {branch && <span className="text-xs text-gray-400">{branch.name}</span>}
            {!doctor.is_active && <span className="text-xs text-rose-500 font-medium">Неактивен</span>}
          </div>
        </div>
      </div>
      {dirs.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {dirs.slice(0, 3).map((d) => (
            <span key={d.id} className="text-[11px] px-2 py-0.5 rounded-full bg-blue-50 text-blue-600 border border-blue-100 leading-none">
              {d.name}
            </span>
          ))}
          {dirs.length > 3 && (
            <span className="text-[11px] px-2 py-0.5 rounded-full bg-gray-100 text-gray-500 leading-none">
              +{dirs.length - 3}
            </span>
          )}
        </div>
      )}
      <div className="flex items-center pt-1 border-t border-gray-100 -mx-1">
        <button
          onClick={() => onNavigate(doctor.id)}
          className="flex-1 flex items-center justify-center gap-1.5 py-1.5 text-xs text-gray-500 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
        >
          <ExternalLink size={12} />
          Открыть
        </button>
        <button
          onClick={() => onEdit(doctor)}
          className="flex-1 flex items-center justify-center gap-1.5 py-1.5 text-xs text-gray-500 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
        >
          <Pencil size={12} />
          Изменить
        </button>
        <button
          onClick={() => onDelete(doctor)}
          className="flex items-center justify-center px-3 py-1.5 text-xs text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
        >
          <Trash2 size={12} />
        </button>
      </div>
    </div>
  )
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

  const { data: allBranches = [] } = useQuery({
    queryKey: ['branches'],
    queryFn: getBranches,
  })

  const createMut = useMutation({
    mutationFn: async ({ direction_ids, create_account, email, password, ...rest }) => {
      const payload = toPayload({ direction_ids, create_account, email, password, ...rest })
      const doctor = await createDoctor(payload)
      if (direction_ids?.length > 0) {
        await setDoctorDirections(doctor.id, direction_ids)
      }
      return doctor
    },
    onSuccess: (doctor) => {
      qc.invalidateQueries({ queryKey: ['doctors'] })
      setCreateOpen(false)
      toast.success('Врач создан')
      navigate(`/admin/doctors/${doctor.id}`)
    },
    onError: (err) => {
      const status = err?.response?.status
      if (status === 409) {
        toast.error('Email уже занят')
      } else if (status === 422) {
        toast.error('Пароль слишком слабый')
      } else {
        toast.error('Не удалось создать врача')
      }
    },
  })

  const updateMut = useMutation({
    mutationFn: async ({ id, direction_ids, create_account, email, password, ...rest }) => {
      await updateDoctor(id, toPayload({ direction_ids, create_account: false, email, password, ...rest }))
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

  const editDefaultValues = editTarget
    ? {
        last_name: editTarget.last_name,
        first_name: editTarget.first_name,
        middle_name: editTarget.middle_name ?? '',
        phone: editTarget.phone ?? '',
        cabinet: editTarget.cabinet ?? '',
        branch_id: editTarget.branch_id ?? null,
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

      {isLoading ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="bg-white rounded-xl border border-gray-200 p-4 animate-pulse">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-10 h-10 rounded-full bg-gray-200 shrink-0" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 bg-gray-200 rounded w-3/4" />
                  <div className="h-3 bg-gray-100 rounded w-1/2" />
                </div>
              </div>
              <div className="flex gap-1.5">
                <div className="h-5 bg-gray-100 rounded-full w-16" />
                <div className="h-5 bg-gray-100 rounded-full w-20" />
              </div>
            </div>
          ))}
        </div>
      ) : doctors.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-gray-400">
          <Users size={40} strokeWidth={1.25} className="mb-3 text-gray-300" />
          <p className="text-sm">Врачей пока нет</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {doctors.map((d) => (
            <DoctorCard
              key={d.id}
              doctor={d}
              allBranches={allBranches}
              onNavigate={(id) => navigate(`/admin/doctors/${id}`)}
              onEdit={setEditTarget}
              onDelete={setDeleteTarget}
            />
          ))}
        </div>
      )}

      <Modal
        isOpen={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Новый врач"
        maxWidth="max-w-lg"
      >
        <DoctorForm
          allDirections={allDirections}
          allBranches={allBranches}
          onSubmit={(data) => createMut.mutate(data)}
          isLoading={createMut.isPending}
          isEdit={false}
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
            allBranches={allBranches}
            onSubmit={(data) => updateMut.mutate({ id: editTarget.id, ...data })}
            isLoading={updateMut.isPending}
            isEdit={true}
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
