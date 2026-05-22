import { Tag, Lock } from 'lucide-react'

export default function PricesPage() {
  return (
    <div className="p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Прайсы</h1>
        <p className="text-sm text-gray-500 mt-0.5">Тарифы и стоимость услуг по филиалам</p>
      </div>
      <div className="flex flex-col items-center justify-center py-24 gap-4 text-gray-400">
        <div className="w-16 h-16 rounded-2xl bg-gray-100 flex items-center justify-center">
          <Tag size={28} strokeWidth={1.25} className="text-gray-400" />
        </div>
        <div className="text-center">
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 bg-gray-100 text-gray-500 rounded-full text-xs font-medium mb-3">
            <Lock size={11} /> Скоро
          </span>
          <p className="text-sm font-medium text-gray-600">Управление прайсами</p>
          <p className="text-xs text-gray-400 mt-1 max-w-xs leading-relaxed">
            Гибкое ценообразование: базовые тарифы, надбавки по филиалу, сезонные акции.
          </p>
        </div>
      </div>
    </div>
  )
}
