import { useState, useEffect } from 'react'
import { Mail, Phone, Building2, Save, Pencil, X } from 'lucide-react'
import { clinic as defaultClinic } from '../../../lib/clinic.config'

const STORAGE_KEY = 'clinic_profile'

function loadProfile() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return { ...defaultClinic, ...JSON.parse(raw) }
  } catch {}
  return { ...defaultClinic }
}

function saveProfile(data) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
}

export default function ClinicPage() {
  const [editing, setEditing] = useState(false)
  const [saved, setSaved] = useState(loadProfile)
  const [form, setForm] = useState(saved)

  useEffect(() => {
    setSaved(loadProfile())
  }, [])

  const handleEdit = () => {
    setForm({ ...saved })
    setEditing(true)
  }

  const handleCancel = () => {
    setForm({ ...saved })
    setEditing(false)
  }

  const handleSave = () => {
    saveProfile(form)
    setSaved({ ...form })
    setEditing(false)
  }

  const set = (key) => (e) => setForm((f) => ({ ...f, [key]: e.target.value }))

  const data = editing ? form : saved

  return (
    <div className="p-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Профиль клиники</h1>
          <p className="text-sm text-gray-500 mt-0.5">Основная информация об организации</p>
        </div>
        {!editing ? (
          <button
            onClick={handleEdit}
            className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-gray-700 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
          >
            <Pencil size={14} />
            Редактировать
          </button>
        ) : (
          <div className="flex items-center gap-2">
            <button
              onClick={handleCancel}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-600 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              <X size={14} />
              Отмена
            </button>
            <button
              onClick={handleSave}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
            >
              <Save size={14} />
              Сохранить
            </button>
          </div>
        )}
      </div>

      <div className="bg-white rounded-xl border border-gray-200 p-6 max-w-xl space-y-5">
        {/* Logo placeholder */}
        <div className="flex items-center gap-4">
          <div className="w-14 h-14 rounded-xl bg-blue-600 flex items-center justify-center shrink-0">
            <span className="text-white font-bold text-sm">{(data.shortName ?? data.name?.slice(0, 2) ?? 'МП').toUpperCase()}</span>
          </div>
          {editing ? (
            <p className="text-xs text-gray-400">Загрузка логотипа — в следующей версии</p>
          ) : (
            <div>
              <p className="text-base font-semibold text-gray-900">{data.name}</p>
              <p className="text-sm text-gray-500">{data.tagline}</p>
            </div>
          )}
        </div>

        {/* Fields */}
        <div className="space-y-4">
          <Field
            label="Название клиники"
            value={data.name}
            editing={editing}
            onChange={set('name')}
            placeholder="МЕДИК-ПРОФИ"
          />
          <Field
            label="Юридическое / отображаемое имя"
            value={data.tagline}
            editing={editing}
            onChange={set('tagline')}
            placeholder="Медицинский центр"
          />
          <div className="grid grid-cols-2 gap-4">
            <FieldWithIcon
              label="Телефон"
              icon={Phone}
              value={data.phone}
              editing={editing}
              onChange={set('phone')}
              placeholder="+7 (999) 000-00-00"
              type="tel"
            />
            <FieldWithIcon
              label="Email"
              icon={Mail}
              value={data.supportEmail}
              editing={editing}
              onChange={set('supportEmail')}
              placeholder="info@clinic.ru"
              type="email"
            />
          </div>
          <FieldWithIcon
            label="Главный филиал"
            icon={Building2}
            value={data.mainBranch}
            editing={editing}
            onChange={set('mainBranch')}
            placeholder="ул. Ленина, 1"
          />
        </div>

        {editing && (
          <p className="text-xs text-gray-400 pt-1">
            Данные сохраняются локально в браузере. Синхронизация с сервером — в v0.4.
          </p>
        )}
      </div>
    </div>
  )
}

function Field({ label, value, editing, onChange, placeholder }) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-500 mb-1">{label}</label>
      {editing ? (
        <input
          type="text"
          value={value ?? ''}
          onChange={onChange}
          placeholder={placeholder}
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      ) : (
        <p className="text-sm text-gray-900">{value || <span className="text-gray-400">—</span>}</p>
      )}
    </div>
  )
}

function FieldWithIcon({ label, value, editing, onChange, placeholder, icon: Icon, type = 'text' }) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-500 mb-1">{label}</label>
      {editing ? (
        <input
          type={type}
          value={value ?? ''}
          onChange={onChange}
          placeholder={placeholder}
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      ) : (
        <div className="flex items-center gap-2 text-sm text-gray-700">
          <Icon size={14} className="text-gray-400 shrink-0" />
          {value || <span className="text-gray-400">—</span>}
        </div>
      )}
    </div>
  )
}
