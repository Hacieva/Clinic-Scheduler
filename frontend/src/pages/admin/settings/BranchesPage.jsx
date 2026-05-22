import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Pencil, Building2, ToggleLeft, ToggleRight } from 'lucide-react'
import toast from 'react-hot-toast'
import { getBranches, createBranch, updateBranch } from '../../../api/branches'
import Modal from '../../../components/Modal'
import ConfirmDialog from '../../../components/ConfirmDialog'
import Badge from '../../../components/Badge'

// ─── Schema ───────────────────────────────────────────────────────────────────

const schema = z.object({
  name: z.string().min(1, 'Введите название'),
  address: z.string().optional(),
  phone: z.string().optional(),
})

// ─── BranchForm ───────────────────────────────────────────────────────────────

function BranchForm({ defaultValues, onSubmit, isLoading }) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(schema),
    defaultValues: defaultValues ?? { name: '', address: '', phone: '' },
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
          placeholder="Главный филиал"
          {...register('name')}
        />
        {errors.name && <p className="mt-1 text-xs text-red-600">{errors.name.message}</p>}
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Адрес</label>
        <input
          type="text"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          placeholder="ул. Ленина, 5"
          {...register('address')}
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Телефон</label>
        <input
          type="tel"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          placeholder="+7 (999) 000-00-00"
          {...register('phone')}
        />
      </div>

      <div className="flex justify-end pt-1">
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-60 transition-colors"
        >
          {isLoading ? 'Сохранение…' : 'Сохранить'}
        </button>
      </div>
    </form>
  )
}

// ─── BranchRow ────────────────────────────────────────────────────────────────

function BranchRow({ branch, onEdit, onToggle }) {
  return (
    <div className="flex items-center gap-4 px-5 py-4 border-b border-gray-100 last:border-0 hover:bg-gray-50/60 transition-colors">
      <div className="w-8 h-8 rounded-lg bg-blue-50 flex items-center justify-center shrink-0">
        <Building2 size={15} className="text-blue-600" />
      </div>

      <div className="flex-1 min-w-0">
        <p className="text-sm font-semibold text-gray-900 truncate">{branch.name}</p>
        {branch.address && (
          <p className="text-xs text-gray-400 truncate mt-0.5">{branch.address}</p>
        )}
      </div>

      {branch.phone && (
        <span className="text-xs text-gray-500 hidden sm:block shrink-0">{branch.phone}</span>
      )}

      <Badge variant={branch.is_active ? 'active' : 'inactive'}>
        {branch.is_active ? 'Активен' : 'Неактивен'}
      </Badge>

      <div className="flex items-center gap-1 shrink-0">
        <button
          onClick={() => onEdit(branch)}
          className="p-1.5 text-gray-400 hover:text-blue-600 rounded-lg hover:bg-blue-50 transition-colors"
          title="Редактировать"
        >
          <Pencil size={14} />
        </button>
        <button
          onClick={() => onToggle(branch)}
          className={`p-1.5 rounded-lg transition-colors ${
            branch.is_active
              ? 'text-gray-400 hover:text-amber-600 hover:bg-amber-50'
              : 'text-gray-400 hover:text-green-600 hover:bg-green-50'
          }`}
          title={branch.is_active ? 'Деактивировать' : 'Активировать'}
        >
          {branch.is_active ? <ToggleRight size={16} /> : <ToggleLeft size={16} />}
        </button>
      </div>
    </div>
  )
}

// ─── BranchesPage ─────────────────────────────────────────────────────────────

export default function BranchesPage() {
  const qc = useQueryClient()
  const [editTarget, setEditTarget] = useState(null)   // null | branch obj
  const [createOpen, setCreateOpen] = useState(false)
  const [toggleTarget, setToggleTarget] = useState(null)

  const { data: branches = [], isLoading } = useQuery({
    queryKey: ['branches'],
    queryFn: getBranches,
  })

  const invalidate = () => qc.invalidateQueries({ queryKey: ['branches'] })

  const createMut = useMutation({
    mutationFn: createBranch,
    onSuccess: () => { invalidate(); setCreateOpen(false); toast.success('Филиал добавлен') },
    onError: () => toast.error('Не удалось добавить филиал'),
  })

  const updateMut = useMutation({
    mutationFn: ({ id, data }) => updateBranch(id, data),
    onSuccess: () => { invalidate(); setEditTarget(null); toast.success('Филиал обновлён') },
    onError: () => toast.error('Не удалось обновить филиал'),
  })

  const handleToggleConfirm = () => {
    if (!toggleTarget) return
    updateMut.mutate(
      { id: toggleTarget.id, data: { is_active: !toggleTarget.is_active } },
      { onSuccess: () => { invalidate(); setToggleTarget(null) } },
    )
  }

  return (
    <div className="p-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Филиалы</h1>
          <p className="text-sm text-gray-500 mt-0.5">Подразделения клиники</p>
        </div>
        <button
          onClick={() => setCreateOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
        >
          <Plus size={16} />
          Добавить филиал
        </button>
      </div>

      {/* List */}
      <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
        {isLoading ? (
          <div className="py-12 text-center text-sm text-gray-400">Загрузка…</div>
        ) : branches.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 gap-3 text-gray-400">
            <Building2 size={36} strokeWidth={1.25} />
            <p className="text-sm font-medium text-gray-500">Филиалы не добавлены</p>
            <p className="text-xs">Добавьте первый филиал клиники</p>
          </div>
        ) : (
          branches.map((b) => (
            <BranchRow
              key={b.id}
              branch={b}
              onEdit={setEditTarget}
              onToggle={setToggleTarget}
            />
          ))
        )}
      </div>

      {/* Create modal */}
      <Modal isOpen={createOpen} onClose={() => setCreateOpen(false)} title="Новый филиал">
        <BranchForm
          onSubmit={(data) => createMut.mutate(data)}
          isLoading={createMut.isPending}
        />
      </Modal>

      {/* Edit modal */}
      <Modal isOpen={!!editTarget} onClose={() => setEditTarget(null)} title="Редактировать филиал">
        {editTarget && (
          <BranchForm
            defaultValues={{ name: editTarget.name, address: editTarget.address ?? '', phone: editTarget.phone ?? '' }}
            onSubmit={(data) => updateMut.mutate({ id: editTarget.id, data })}
            isLoading={updateMut.isPending}
          />
        )}
      </Modal>

      {/* Toggle confirm */}
      <ConfirmDialog
        isOpen={!!toggleTarget}
        onClose={() => setToggleTarget(null)}
        onConfirm={handleToggleConfirm}
        title={toggleTarget?.is_active ? 'Деактивировать филиал' : 'Активировать филиал'}
        message={
          toggleTarget?.is_active
            ? `Филиал «${toggleTarget?.name}» будет деактивирован. Врачи этого филиала перестанут отображаться.`
            : `Филиал «${toggleTarget?.name}» будет активирован.`
        }
        confirmLabel={toggleTarget?.is_active ? 'Деактивировать' : 'Активировать'}
        confirmVariant={toggleTarget?.is_active ? 'warning' : 'primary'}
        isLoading={updateMut.isPending}
      />
    </div>
  )
}
