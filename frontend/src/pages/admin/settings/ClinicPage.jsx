import { Mail, Phone } from 'lucide-react'
import { clinic } from '../../../lib/clinic.config'

export default function ClinicPage() {
  return (
    <div className="p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Профиль клиники</h1>
        <p className="text-sm text-gray-500 mt-0.5">Основная информация об организации</p>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 p-6 max-w-xl">
        <div className="flex items-center gap-4 mb-6">
          <div className="w-12 h-12 rounded-xl bg-blue-600 flex items-center justify-center shrink-0">
            <span className="text-white font-bold text-sm">{clinic.shortName}</span>
          </div>
          <div>
            <h2 className="text-lg font-semibold text-gray-900">{clinic.name}</h2>
            <p className="text-sm text-gray-500">{clinic.tagline}</p>
          </div>
        </div>

        <div className="space-y-3 text-sm">
          {clinic.phone && (
            <div className="flex items-center gap-3 text-gray-600">
              <Phone size={15} className="text-gray-400 shrink-0" />
              {clinic.phone}
            </div>
          )}
          {clinic.supportEmail && (
            <div className="flex items-center gap-3 text-gray-600">
              <Mail size={15} className="text-gray-400 shrink-0" />
              {clinic.supportEmail}
            </div>
          )}
        </div>

        <p className="mt-6 text-xs text-gray-400">
          Для изменения данных обратитесь к администратору системы.
        </p>
      </div>
    </div>
  )
}
