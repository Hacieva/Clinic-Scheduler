import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import toast from 'react-hot-toast'
import { getWorkingHours, replaceWorkingHours } from '../../api/schedule'
import ScheduleEditor from '../../components/ScheduleEditor'

export default function DoctorScheduleTab({ doctorId }) {
  const qc = useQueryClient()

  const { data: schedule, isLoading } = useQuery({
    queryKey: ['working-hours', doctorId],
    queryFn: () => getWorkingHours(doctorId),
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
    <div className="max-w-lg">
      <p className="text-sm text-gray-500 mb-4">
        Настройте рабочие часы врача. Только выбранные дни будут доступны для записи.
      </p>
      <ScheduleEditor
        schedule={schedule ?? []}
        onSubmit={(items) => mut.mutate(items)}
        isLoading={mut.isPending}
      />
    </div>
  )
}
