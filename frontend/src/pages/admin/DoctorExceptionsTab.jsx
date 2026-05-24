import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { format, subDays, addDays } from 'date-fns'
import toast from 'react-hot-toast'
import {
  getExceptions,
  createException,
  updateException,
  deleteException,
  createExceptionRange,
} from '../../api/schedule'
import ExceptionsManager from '../../components/ExceptionsManager'

function toApiPayload(data) {
  return {
    date: data.date,
    type: data.type,
    ...(data.type === 'custom_working_hours' && {
      start_time: data.start_time || undefined,
      end_time: data.end_time || undefined,
    }),
    ...(data.comment && { comment: data.comment }),
  }
}

function VacationRangeForm({ doctorId, onCreated }) {
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [busy, setBusy] = useState(false)

  const handleAdd = async () => {
    if (!from || !to) {
      toast.error('Укажите период')
      return
    }
    if (from > to) {
      toast.error('Дата начала должна быть раньше даты окончания')
      return
    }
    setBusy(true)
    try {
      const result = await createExceptionRange(doctorId, { from, to, type: 'day_off' })
      const count = result?.created ?? 0
      toast.success(`Добавлено выходных: ${count}`)
      setFrom('')
      setTo('')
      onCreated()
    } catch {
      toast.error('Не удалось добавить выходные')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="mb-5 p-4 bg-amber-50 border border-amber-200 rounded-xl">
      <p className="text-sm font-medium text-amber-800 mb-3">Добавить отпуск / выходные дни</p>
      <div className="flex items-end gap-3 flex-wrap">
        <div>
          <label className="block text-xs text-gray-600 mb-1">С</label>
          <input
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            className="border border-gray-300 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-amber-400"
          />
        </div>
        <div>
          <label className="block text-xs text-gray-600 mb-1">По</label>
          <input
            type="date"
            value={to}
            min={from}
            onChange={(e) => setTo(e.target.value)}
            className="border border-gray-300 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-amber-400"
          />
        </div>
        <button
          onClick={handleAdd}
          disabled={busy || !from || !to}
          className="px-4 py-1.5 text-sm font-medium text-white bg-amber-500 hover:bg-amber-600 rounded-lg disabled:opacity-50 transition-colors"
        >
          {busy ? 'Добавляю...' : 'Добавить выходные'}
        </button>
      </div>
      <p className="mt-2 text-xs text-amber-600">
        Все даты в диапазоне будут отмечены как выходной день. Уже существующие пропускаются.
      </p>
    </div>
  )
}

export default function DoctorExceptionsTab({ doctorId }) {
  const qc = useQueryClient()
  const today = new Date()
  const [from] = useState(() => format(subDays(today, 30), 'yyyy-MM-dd'))
  const [to] = useState(() => format(addDays(today, 90), 'yyyy-MM-dd'))

  const { data: exceptions = [], isLoading } = useQuery({
    queryKey: ['exceptions', doctorId, from, to],
    queryFn: () => getExceptions(doctorId, from, to),
  })

  const invalidate = () => qc.invalidateQueries({ queryKey: ['exceptions', doctorId] })

  const createMut = useMutation({
    mutationFn: (data) => createException(doctorId, toApiPayload(data)),
    onSuccess: () => { invalidate(); toast.success('Исключение добавлено') },
    onError: (err) => {
      if (err?.response?.status === 409) {
        toast.error('На эту дату уже есть исключение')
      } else {
        toast.error('Не удалось добавить исключение')
      }
    },
  })

  const updateMut = useMutation({
    mutationFn: ({ id, data }) => updateException(doctorId, id, toApiPayload(data)),
    onSuccess: () => { invalidate(); toast.success('Исключение обновлено') },
    onError: () => toast.error('Не удалось обновить исключение'),
  })

  const deleteMut = useMutation({
    mutationFn: (id) => deleteException(doctorId, id),
    onSuccess: () => { invalidate(); toast.success('Исключение удалено') },
    onError: () => toast.error('Не удалось удалить исключение'),
  })

  if (isLoading) {
    return <p className="text-sm text-gray-500">Загрузка...</p>
  }

  return (
    <div>
      <VacationRangeForm doctorId={doctorId} onCreated={invalidate} />

      <p className="text-sm text-gray-500 mb-4">
        Исключения за период {from} — {to} (выходные дни и особое расписание).
      </p>
      <ExceptionsManager
        exceptions={exceptions}
        onAdd={(data, onDone) => createMut.mutate(data, { onSuccess: onDone })}
        onEdit={(id, data, onDone) => updateMut.mutate({ id, data }, { onSuccess: onDone })}
        onDelete={(id, onDone) => deleteMut.mutate(id, { onSuccess: onDone })}
        adding={createMut.isPending}
        editing={updateMut.isPending}
        deleting={deleteMut.isPending}
      />
    </div>
  )
}
