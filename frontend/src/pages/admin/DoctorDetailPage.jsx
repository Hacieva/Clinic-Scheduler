import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ChevronLeft } from 'lucide-react'
import { getDoctorById } from '../../api/doctors'
import { getDirections } from '../../api/directions'
import Badge from '../../components/Badge'
import DoctorInfoTab from './DoctorInfoTab'
import DoctorServicesTab from './DoctorServicesTab'
import DoctorScheduleTab from './DoctorScheduleTab'
import DoctorExceptionsTab from './DoctorExceptionsTab'
import DoctorAccountTab from './DoctorAccountTab'

const TABS = [
  { id: 'info', label: 'Информация' },
  { id: 'services', label: 'Услуги' },
  { id: 'schedule', label: 'Расписание' },
  { id: 'exceptions', label: 'Исключения' },
  { id: 'account', label: 'Аккаунт' },
]

function fullName(d) {
  return [d.last_name, d.first_name, d.middle_name].filter(Boolean).join(' ')
}

export default function DoctorDetailPage() {
  const { id } = useParams()
  const doctorId = Number(id)
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState('info')

  const { data: doctor, isLoading, isError } = useQuery({
    queryKey: ['doctor', doctorId],
    queryFn: () => getDoctorById(doctorId),
  })

  const { data: allDirections = [] } = useQuery({
    queryKey: ['directions'],
    queryFn: getDirections,
  })

  if (isLoading) {
    return (
      <div className="p-8 text-sm text-gray-500">Загрузка...</div>
    )
  }

  if (isError || !doctor) {
    return (
      <div className="p-8 text-sm text-red-600">Врач не найден.</div>
    )
  }

  return (
    <div className="p-8">
      {/* Header */}
      <div className="flex items-start gap-3 mb-6">
        <button
          onClick={() => navigate(-1)}
          className="mt-0.5 p-1.5 text-gray-400 hover:text-gray-700 rounded-lg hover:bg-gray-100 transition-colors shrink-0"
          title="Назад"
        >
          <ChevronLeft size={20} />
        </button>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">{fullName(doctor)}</h1>
          <div className="flex items-center gap-2 mt-1">
            <Badge variant={doctor.is_active ? 'active' : 'inactive'}>
              {doctor.is_active ? 'Активен' : 'Неактивен'}
            </Badge>
            {doctor.cabinet && (
              <span className="text-sm text-gray-500">Кабинет {doctor.cabinet}</span>
            )}
          </div>
        </div>
      </div>

      {/* Tab nav */}
      <div className="border-b border-gray-200 mb-6">
        <nav className="-mb-px flex gap-0">
          {TABS.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab.id
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab content */}
      {activeTab === 'info' && (
        <DoctorInfoTab
          doctor={doctor}
          doctorId={doctorId}
          allDirections={allDirections}
        />
      )}
      {activeTab === 'services' && (
        <DoctorServicesTab
          doctorId={doctorId}
          doctorDirections={doctor.directions ?? []}
        />
      )}
      {activeTab === 'schedule' && <DoctorScheduleTab doctorId={doctorId} />}
      {activeTab === 'exceptions' && <DoctorExceptionsTab doctorId={doctorId} />}
      {activeTab === 'account' && (
        <DoctorAccountTab doctor={doctor} doctorId={doctorId} />
      )}
    </div>
  )
}
