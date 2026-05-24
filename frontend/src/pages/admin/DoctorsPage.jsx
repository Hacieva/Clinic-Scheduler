import { useState, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Pencil, Trash2, ExternalLink, Users, Search } from 'lucide-react'
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
    register, handleSubmit, watch, setValue, formState: { errors },
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
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Фамилия <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
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
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
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
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          {...register('middle_name')}
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Телефон</label>
          <input
            type="text"
            placeholder="+7 999 000 00 00"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            {...register('phone')}
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Кабинет</label>
          <input
            type="text"
            className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
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

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Описание</label>
        <textarea
          rows={2}
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
          {...register('description')}
        />
      </div>

      {!isEdit && (
        <div className="border-t border-gray-100 pt-4">
          <label className="flex items-center gap-2 cursor-pointer mb-3">
            <input type="checkbox" className="rounded" {...register('create_account')} />
            <span className="text-sm font-medium text-gray-700">Создать аккаунт сейчас</span>
          </label>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
              <input
                type="email"
                autoComplete="off"
                disabled={!createAccount}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-50 disabled:text-gray-400"
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
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-50 disabled:text-gray-400"
                {...register('password')}
              />
              {errors.password && (
                <p className="mt-1 text-xs text-red-600">{errors.password.message}</p>
              )}
            </div>
          </div>
          {createAccount && (
            <p className="mt-1.5 text-xs text-gray-500">
              Пароль будет показан на странице врача. Минимум 8 символов.
            </p>
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

// ─── Compact staff row ────────────────────────────────────────────────────────

function StaffRow({ doctor, allBranches, onNavigate, onEdit, onDelete }) {
  const name = fullName(doctor)
  const initial = (doctor.last_name ?? doctor.first_name ?? '?')[0]
  const dirs = doctor.directions ?? []
  const branch = allBranches.find((b) => b.id === doctor.branch_id)

  return (
    <tr
      className={`border-b border-gray-100 last:border-0 hover:bg-gray-50/60 transition-colors ${
        !doctor.is_active ? 'opacity-60' : ''
      }`}
    >
      {/* Avatar + name + directions */}
      <td className="px-4 py-3">
        <div className="flex items-center gap-3">
          <div
            className={`w-8 h-8 rounded-full ${avatarBg(doctor.id)} flex items-center justify-center text-white text-xs font-bold shrink-0`}
          >
            {initial}
          </div>
          <div className="min-w-0">
            <p className="text-sm font-semibold text-gray-900 leading-tight">{name}</p>
            {dirs.length > 0 && (
              <div className="flex gap-1 mt-0.5 flex-wrap">
                {dirs.slice(0, 2).map((d) => (
                  <span
                    key={d.id}
                    className="text-[10px] px-1.5 py-0.5 rounded-full bg-blue-50 text-blue-600 border border-blue-100 leading-none whitespace-nowrap"
                  >
                    {d.name}
                  </span>
                ))}
                {dirs.length > 2 && (
                  <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-gray-100 text-gray-500 leading-none">
                    +{dirs.length - 2}
                  </span>
                )}
              </div>
            )}
          </div>
        </div>
      </td>

      {/* Phone */}
      <td className="px-4 py-3 text-sm text-gray-600 whitespace-nowrap">
        {doctor.phone ?? <span className="text-gray-300">—</span>}
      </td>

      {/* Cabinet */}
      <td className="px-4 py-3 text-sm text-gray-600 whitespace-nowrap">
        {doctor.cabinet ? `Каб. ${doctor.cabinet}` : <span className="text-gray-300">—</span>}
      </td>

      {/* Branch */}
      <td className="px-4 py-3 text-sm text-gray-600 whitespace-nowrap">
        {branch ? branch.name : <span className="text-gray-300">—</span>}
      </td>

      {/* Status */}
      <td className="px-4 py-3">
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
            doctor.is_active
              ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
              : 'bg-rose-50 text-rose-600 border border-rose-200'
          }`}
        >
          {doctor.is_active ? 'Активен' : 'Неактивен'}
        </span>
      </td>

      {/* Actions */}
      <td className="px-4 py-3">
        <div className="flex items-center gap-1 justify-end">
          <button
            onClick={() => onNavigate(doctor.id)}
            className="p-1.5 text-gray-400 hover:text-blue-600 rounded transition-colors"
            title="Открыть"
          >
            <ExternalLink size={14} />
          </button>
          <button
            onClick={() => onEdit(doctor)}
            className="p-1.5 text-gray-400 hover:text-blue-600 rounded transition-colors"
            title="Изменить"
          >
            <Pencil size={14} />
          </button>
          <button
            onClick={() => onDelete(doctor)}
            disabled={!doctor.is_active}
            className="p-1.5 text-gray-400 hover:text-red-600 rounded transition-colors disabled:opacity-30"
            title="Деактивировать"
          >
            <Trash2 size={14} />
          </button>
        </div>
      </td>
    </tr>
  )
}

// ─── DoctorsPage ──────────────────────────────────────────────────────────────

export default function DoctorsPage() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [search, setSearch] = useState('')
  const [filterDirId, setFilterDirId] = useState('')

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

  const filtered = useMemo(() => {
    let result = doctors
    if (search.trim()) {
      const q = search.toLowerCase()
      result = result.filter((d) => fullName(d).toLowerCase().includes(q))
    }
    if (filterDirId) {
      result = result.filter((d) =>
        (d.directions ?? []).some((dir) => String(dir.id) === filterDirId),
      )
    }
    return result
  }, [doctors, search, filterDirId])

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
      toast.success('Сотрудник добавлен')
      navigate(`/admin/doctors/${doctor.id}`)
    },
    onError: (err) => {
      const status = err?.response?.status
      if (status === 409) toast.error('Email уже занят')
      else if (status === 422) toast.error('Пароль слишком слабый')
      else toast.error('Не удалось добавить сотрудника')
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
      toast.success('Сотрудник обновлён')
    },
    onError: () => toast.error('Не удалось обновить сотрудника'),
  })

  const deleteMut = useMutation({
    mutationFn: (id) => deleteDoctor(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['doctors'] })
      setDeleteTarget(null)
      toast.success('Сотрудник деактивирован')
    },
    onError: () => toast.error('Не удалось деактивировать сотрудника'),
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

  const activeDirections = allDirections.filter((d) => d.is_active)

  return (
    <div className="p-6 lg:p-8">
      <div className="flex items-center justify-between mb-5 flex-wrap gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Сотрудники</h1>
          <p className="text-sm text-gray-500 mt-0.5">{doctors.length} врачей в системе</p>
        </div>
        <button
          onClick={() => setCreateOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
        >
          <Plus size={16} />
          Добавить сотрудника
        </button>
      </div>

      {/* Filters row */}
      <div className="flex items-center gap-3 mb-4 flex-wrap">
        <div className="relative flex-1 max-w-xs">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 pointer-events-none" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Поиск по ФИО…"
            className="w-full pl-9 pr-4 py-2 text-sm border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        {activeDirections.length > 0 && (
          <select
            value={filterDirId}
            onChange={(e) => setFilterDirId(e.target.value)}
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 text-gray-700"
          >
            <option value="">Все специальности</option>
            {activeDirections.map((d) => (
              <option key={d.id} value={String(d.id)}>
                {d.name}
              </option>
            ))}
          </select>
        )}
      </div>

      {/* Staff table */}
      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        {isLoading ? (
          <div className="divide-y divide-gray-100">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="px-4 py-3 flex items-center gap-3 animate-pulse">
                <div className="w-8 h-8 rounded-full bg-gray-200 shrink-0" />
                <div className="flex-1 space-y-1.5">
                  <div className="h-3.5 bg-gray-200 rounded w-40" />
                  <div className="h-3 bg-gray-100 rounded w-24" />
                </div>
              </div>
            ))}
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-gray-400 gap-2">
            <Users size={36} strokeWidth={1.25} className="text-gray-300" />
            <p className="text-sm">
              {doctors.length === 0 ? 'Сотрудников пока нет' : 'Ничего не найдено'}
            </p>
          </div>
        ) : (
          <table className="w-full text-left">
            <thead>
              <tr className="border-b border-gray-200 bg-gray-50/80">
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide">
                  Сотрудник
                </th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide">
                  Телефон
                </th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide">
                  Кабинет
                </th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide">
                  Филиал
                </th>
                <th className="px-4 py-2.5 text-xs font-semibold text-gray-500 uppercase tracking-wide">
                  Статус
                </th>
                <th className="px-4 py-2.5 w-24" />
              </tr>
            </thead>
            <tbody>
              {filtered.map((d) => (
                <StaffRow
                  key={d.id}
                  doctor={d}
                  allBranches={allBranches}
                  onNavigate={(id) => navigate(`/admin/doctors/${id}`)}
                  onEdit={setEditTarget}
                  onDelete={setDeleteTarget}
                />
              ))}
            </tbody>
          </table>
        )}
      </div>

      <Modal isOpen={createOpen} onClose={() => setCreateOpen(false)} title="Новый сотрудник" maxWidth="max-w-lg">
        <DoctorForm
          allDirections={allDirections}
          allBranches={allBranches}
          onSubmit={(data) => createMut.mutate(data)}
          isLoading={createMut.isPending}
          isEdit={false}
        />
      </Modal>

      <Modal isOpen={!!editTarget} onClose={() => setEditTarget(null)} title="Редактировать сотрудника" maxWidth="max-w-lg">
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
        title="Деактивировать сотрудника"
        message={`Сотрудник ${deleteTarget ? fullName(deleteTarget) : ''} будет деактивирован. Записи сохранятся.`}
        confirmLabel="Деактивировать"
        isLoading={deleteMut.isPending}
      />
    </div>
  )
}
