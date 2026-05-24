import { ClipboardList, Plus } from 'lucide-react'

export default function CrmPage() {
  return (
    <div className="p-6 lg:p-8 max-w-5xl mx-auto">
      <div className="mb-6 flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">CRM / Задачи</h1>
          <p className="text-sm text-gray-500 mt-0.5">Задачи и воронка по пациентам</p>
        </div>
        <button
          disabled
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg opacity-40 cursor-not-allowed"
        >
          <Plus size={16} />
          Новая задача
        </button>
      </div>

      <div className="bg-white rounded-xl border border-dashed border-gray-300 p-12 flex flex-col items-center justify-center text-center gap-3 min-h-80">
        <ClipboardList size={44} strokeWidth={1.25} className="text-gray-300" />
        <div>
          <p className="text-sm font-semibold text-gray-500">Модуль CRM в разработке</p>
          <p className="text-xs text-gray-400 mt-1 max-w-sm">
            Воронка, задачи по пациентам и напоминания появятся в версии v0.4
          </p>
        </div>
      </div>
    </div>
  )
}
