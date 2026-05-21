import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Copy, CheckCircle } from 'lucide-react'
import toast from 'react-hot-toast'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { createDoctorAccount } from '../../api/doctors'

const schema = z.object({
  email: z.string().email('Введите корректный email'),
  password: z.string().min(8, 'Минимум 8 символов'),
})

function AccountCreatedBanner({ password, onAck }) {
  const [copied, setCopied] = useState(false)

  const copy = () => {
    navigator.clipboard.writeText(password).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <div className="bg-green-50 border border-green-200 rounded-xl p-5 max-w-md">
      <div className="flex items-center gap-2 mb-3">
        <CheckCircle className="text-green-600" size={18} />
        <span className="text-sm font-semibold text-green-800">Аккаунт создан</span>
      </div>
      <p className="text-sm text-green-700 mb-3">
        Сохраните пароль — он больше не будет показан.
      </p>
      <div className="flex items-center gap-2 bg-white border border-green-300 rounded-lg px-3 py-2 font-mono text-sm text-gray-800">
        <span className="flex-1 select-all">{password}</span>
        <button
          onClick={copy}
          className="text-gray-400 hover:text-gray-700 transition-colors shrink-0"
          title="Копировать"
        >
          {copied ? <CheckCircle size={16} className="text-green-500" /> : <Copy size={16} />}
        </button>
      </div>
      <button
        onClick={onAck}
        className="mt-4 px-4 py-2 text-sm font-medium text-green-800 bg-green-100 hover:bg-green-200 rounded-lg transition-colors"
      >
        Понял, скрыть
      </button>
    </div>
  )
}

function CreateAccountForm({ doctorId, onCreated }) {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm({ resolver: zodResolver(schema) })

  const mut = useMutation({
    mutationFn: ({ email, password }) => createDoctorAccount(doctorId, email, password),
    onSuccess: (_, variables) => {
      onCreated(variables.password)
    },
    onError: (err) => {
      const status = err?.response?.status
      if (status === 409) {
        toast.error('Email уже занят или аккаунт уже существует')
      } else if (status === 422) {
        toast.error('Пароль слишком слабый')
      } else {
        toast.error('Не удалось создать аккаунт')
      }
    },
  })

  return (
    <form
      onSubmit={handleSubmit((data) => mut.mutate(data))}
      className="space-y-4 max-w-sm"
    >
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Email <span className="text-red-500">*</span>
        </label>
        <input
          type="email"
          autoComplete="off"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          {...register('email')}
        />
        {errors.email && (
          <p className="mt-1 text-xs text-red-600">{errors.email.message}</p>
        )}
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Пароль <span className="text-red-500">*</span>
        </label>
        <input
          type="password"
          autoComplete="new-password"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          {...register('password')}
        />
        {errors.password && (
          <p className="mt-1 text-xs text-red-600">{errors.password.message}</p>
        )}
        <p className="mt-1 text-xs text-gray-500">Минимум 8 символов</p>
      </div>
      <button
        type="submit"
        disabled={isSubmitting || mut.isPending}
        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-60 transition-colors"
      >
        {mut.isPending ? 'Создание...' : 'Создать аккаунт'}
      </button>
    </form>
  )
}

export default function DoctorAccountTab({ doctor, doctorId }) {
  const qc = useQueryClient()
  const [shownPassword, setShownPassword] = useState(null)

  const handleCreated = (password) => {
    setShownPassword(password)
    qc.invalidateQueries({ queryKey: ['doctor', doctorId] })
  }

  const handleAck = () => {
    setShownPassword(null)
  }

  if (shownPassword) {
    return <AccountCreatedBanner password={shownPassword} onAck={handleAck} />
  }

  if (doctor.user_id != null) {
    return (
      <div className="max-w-sm">
        <div className="flex items-center gap-2 text-green-700 bg-green-50 border border-green-200 rounded-xl px-4 py-3">
          <CheckCircle size={16} />
          <span className="text-sm font-medium">Аккаунт уже создан</span>
        </div>
        <p className="mt-3 text-sm text-gray-500">
          Для сброса пароля используйте функцию смены пароля на странице аккаунта.
        </p>
      </div>
    )
  }

  return (
    <div>
      <p className="text-sm text-gray-500 mb-4">
        Создайте аккаунт для входа врача в систему. Пароль будет показан один раз.
      </p>
      <CreateAccountForm doctorId={doctorId} onCreated={handleCreated} />
    </div>
  )
}
