import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Plus, Pencil, Trash2 } from 'lucide-react'
import { format, parseISO } from 'date-fns'
import Modal from './Modal'
import ConfirmDialog from './ConfirmDialog'
import Badge from './Badge'

const schema = z
  .object({
    date: z.string().min(1, 'Укажите дату'),
    type: z.enum(['day_off', 'custom_working_hours']),
    start_time: z.string().optional(),
    end_time: z.string().optional(),
    comment: z.string().optional(),
  })
  .refine(
    (v) =>
      v.type === 'day_off' ||
      (v.type === 'custom_working_hours' && v.start_time && v.end_time),
    { message: 'Для особого расписания укажите время', path: ['start_time'] },
  )

function ExceptionForm({ defaultValues, onSubmit, isLoading }) {
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm({
    resolver: zodResolver(schema),
    defaultValues: defaultValues ?? { type: 'day_off' },
  })

  const type = watch('type')

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Дата <span className="text-red-500">*</span>
        </label>
        <input
          type="date"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          {...register('date')}
        />
        {errors.date && (
          <p className="mt-1 text-xs text-red-600">{errors.date.message}</p>
        )}
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Тип</label>
        <select
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          {...register('type')}
        >
          <option value="day_off">Выходной</option>
          <option value="custom_working_hours">Особое расписание</option>
        </select>
      </div>
      {type === 'custom_working_hours' && (
        <div className="flex items-center gap-3">
          <div className="flex-1">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              С <span className="text-red-500">*</span>
            </label>
            <input
              type="time"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              {...register('start_time')}
            />
          </div>
          <div className="flex-1">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              До <span className="text-red-500">*</span>
            </label>
            <input
              type="time"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              {...register('end_time')}
            />
          </div>
        </div>
      )}
      {errors.start_time && (
        <p className="text-xs text-red-600">{errors.start_time.message}</p>
      )}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Комментарий
        </label>
        <input
          type="text"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          {...register('comment')}
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

function fmtDate(isoDate) {
  try {
    return format(parseISO(isoDate), 'dd.MM.yyyy')
  } catch {
    return isoDate
  }
}

function fmtTime(isoTime) {
  if (!isoTime) return null
  return isoTime.split('T')[1]?.slice(0, 5) ?? isoTime
}

export default function ExceptionsManager({
  exceptions,
  onAdd,
  onEdit,
  onDelete,
  adding,
  editing,
  deleting,
}) {
  const [createOpen, setCreateOpen] = useState(false)
  const [editTarget, setEditTarget] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)

  const handleAdd = (data) => {
    onAdd(data, () => setCreateOpen(false))
  }

  const handleEdit = (data) => {
    onEdit(editTarget.id, data, () => setEditTarget(null))
  }

  const editDefaults = editTarget
    ? {
        date: editTarget.date.split('T')[0],
        type: editTarget.type,
        start_time: editTarget.start_time ? fmtTime(editTarget.start_time) : '',
        end_time: editTarget.end_time ? fmtTime(editTarget.end_time) : '',
        comment: editTarget.comment ?? '',
      }
    : undefined

  return (
    <div>
      <div className="flex justify-end mb-4">
        <button
          onClick={() => setCreateOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
        >
          <Plus size={16} />
          Добавить исключение
        </button>
      </div>

      {exceptions.length === 0 ? (
        <p className="text-sm text-gray-500 text-center py-8">Исключений нет</p>
      ) : (
        <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                {['Дата', 'Тип', 'Время', 'Комментарий', ''].map((h) => (
                  <th
                    key={h}
                    className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wide"
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {exceptions.map((ex) => (
                <tr key={ex.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 text-sm font-medium text-gray-900">
                    {fmtDate(ex.date)}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <Badge variant={ex.type === 'day_off' ? 'inactive' : 'pending'}>
                      {ex.type === 'day_off' ? 'Выходной' : 'Особое расписание'}
                    </Badge>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">
                    {ex.type === 'day_off'
                      ? '—'
                      : `${fmtTime(ex.start_time)} – ${fmtTime(ex.end_time)}`}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    {ex.comment ?? '—'}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <div className="flex items-center gap-2 justify-end">
                      <button
                        onClick={() => setEditTarget(ex)}
                        className="p-1.5 text-gray-400 hover:text-blue-600 rounded transition-colors"
                        title="Редактировать"
                      >
                        <Pencil size={15} />
                      </button>
                      <button
                        onClick={() => setDeleteTarget(ex)}
                        className="p-1.5 text-gray-400 hover:text-red-600 rounded transition-colors"
                        title="Удалить"
                      >
                        <Trash2 size={15} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal
        isOpen={createOpen}
        onClose={() => setCreateOpen(false)}
        title="Добавить исключение"
      >
        <ExceptionForm onSubmit={handleAdd} isLoading={adding} />
      </Modal>

      <Modal
        isOpen={!!editTarget}
        onClose={() => setEditTarget(null)}
        title="Редактировать исключение"
      >
        {editTarget && (
          <ExceptionForm
            key={editTarget.id}
            defaultValues={editDefaults}
            onSubmit={handleEdit}
            isLoading={editing}
          />
        )}
      </Modal>

      <ConfirmDialog
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => {
          onDelete(deleteTarget.id, () => setDeleteTarget(null))
        }}
        title="Удалить исключение"
        message={`Исключение на ${deleteTarget ? fmtDate(deleteTarget.date) : ''} будет удалено.`}
        confirmLabel="Удалить"
        isLoading={deleting}
      />
    </div>
  )
}
