import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { format, subDays, addDays } from 'date-fns'
import toast from 'react-hot-toast'
import {
  getExceptions,
  createException,
  updateException,
  deleteException,
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

export default function DoctorExceptionsTab({ doctorId }) {
  const qc = useQueryClient()
  const today = new Date()
  const [from] = useState(() => format(subDays(today, 30), 'yyyy-MM-dd'))
  const [to] = useState(() => format(addDays(today, 90), 'yyyy-MM-dd'))

  const { data: exceptions = [], isLoading } = useQuery({
    queryKey: ['exceptions', doctorId, from, to],
    queryFn: () => getExceptions(doctorId, from, to),
  })

  const createMut = useMutation({
    mutationFn: (data) => createException(doctorId, toApiPayload(data)),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['exceptions', doctorId] })
      toast.success('Исключение добавлено')
    },
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
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['exceptions', doctorId] })
      toast.success('Исключение обновлено')
    },
    onError: () => toast.error('Не удалось обновить исключение'),
  })

  const deleteMut = useMutation({
    mutationFn: (id) => deleteException(doctorId, id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['exceptions', doctorId] })
      toast.success('Исключение удалено')
    },
    onError: () => toast.error('Не удалось удалить исключение'),
  })

  if (isLoading) {
    return <p className="text-sm text-gray-500">Загрузка...</p>
  }

  return (
    <div>
      <p className="text-sm text-gray-500 mb-4">
        Исключения за период {from} — {to} (выходные дни и особое расписание).
      </p>
      <ExceptionsManager
        exceptions={exceptions}
        onAdd={(data, onDone) =>
          createMut.mutate(data, { onSuccess: onDone })
        }
        onEdit={(id, data, onDone) =>
          updateMut.mutate({ id, data }, { onSuccess: onDone })
        }
        onDelete={(id, onDone) =>
          deleteMut.mutate(id, { onSuccess: onDone })
        }
        adding={createMut.isPending}
        editing={updateMut.isPending}
        deleting={deleteMut.isPending}
      />
    </div>
  )
}
