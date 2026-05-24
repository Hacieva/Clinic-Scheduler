import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { format, addDays } from 'date-fns'
import toast from 'react-hot-toast'
import { getWorkingHours, replaceWorkingHours, getExceptions } from '../../api/schedule'
import ScheduleEditor from '../../components/ScheduleEditor'
import SchedulePreview from '../../components/SchedulePreview'

export default function DoctorScheduleTab({ doctorId }) {
  const qc = useQueryClient()

  const today = new Date()
  const previewFrom = format(today, 'yyyy-MM-dd')
  const previewTo = format(addDays(today, 27), 'yyyy-MM-dd')

  const { data: schedule, isLoading } = useQuery({
    queryKey: ['working-hours', doctorId],
    queryFn: () => getWorkingHours(doctorId),
  })

  const { data: exceptions = [] } = useQuery({
    queryKey: ['exceptions', doctorId, previewFrom, previewTo],
    queryFn: () => getExceptions(doctorId, previewFrom, previewTo),
  })

  const mut = useMutation({
    mutationFn: (items) => replaceWorkingHours(doctorId, items),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['working-hours', doctorId] })
      toast.success('Расписание сохранено')
    },
    onError: (err) => {
      if (err?.response?.status === 422) {
        toast.error('Неверные параметры расписания')
      } else {
        toast.error('Не удалось сохранить расписание')
      }
    },
  })

  if (isLoading) {
    return <p className="text-sm text-gray-500">Загрузка...</p>
  }

  return (
    <div className="max-w-2xl">
      <p className="text-sm text-gray-500 mb-4">
        Настройте рабочие часы врача. Только выбранные дни будут доступны для записи.
        Используйте кнопку «обед» для добавления перерыва.
      </p>
      <ScheduleEditor
        schedule={schedule ?? []}
        onSubmit={(items) => mut.mutate(items)}
        isLoading={mut.isPending}
      />
      <SchedulePreview schedule={schedule ?? []} exceptions={exceptions} />
    </div>
  )
}
