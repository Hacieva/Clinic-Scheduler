import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useNavigate } from 'react-router-dom'
import toast from 'react-hot-toast'
import { Stethoscope } from 'lucide-react'
import { login } from '../api/auth'
import useAuthStore from '../stores/auth'

const schema = z.object({
  email: z.string().email('Введите корректный email'),
  password: z.string().min(1, 'Введите пароль'),
})

function getErrorMessage(err) {
  const status = err?.response?.status
  if (status === 401) return 'Неверный email или пароль'
  if (status === 403) return 'Аккаунт деактивирован'
  if (status === 429) return 'Слишком много попыток, попробуйте позже'
  if (status >= 500) return 'Ошибка сервера, попробуйте позже'
  return 'Не удалось войти, попробуйте позже'
}

export default function LoginPage() {
  const navigate = useNavigate()
  const { setTokens, setUser } = useAuthStore()

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm({ resolver: zodResolver(schema) })

  const onSubmit = async ({ email, password }) => {
    try {
      const data = await login(email, password)
      setTokens(data.access_token, data.refresh_token)
      setUser(data.user)
      if (data.user.role === 'admin') {
        navigate('/admin/directions', { replace: true })
      } else {
        navigate('/doctor/schedule', { replace: true })
      }
    } catch (err) {
      toast.error(getErrorMessage(err))
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-slate-100 flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <div className="flex flex-col items-center mb-8">
          <div className="flex items-center justify-center w-12 h-12 bg-blue-600 rounded-2xl mb-4 shadow-md">
            <Stethoscope size={24} className="text-white" />
          </div>
          <h1 className="text-2xl font-semibold text-gray-900">Clinic Scheduler</h1>
          <p className="text-sm text-gray-500 mt-1">Панель управления</p>
        </div>
      <div className="bg-white rounded-2xl shadow-lg border border-gray-100 p-8">
        <h2 className="text-lg font-semibold text-gray-900 mb-6">Вход в систему</h2>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Email
            </label>
            <input
              type="email"
              autoComplete="email"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              {...register('email')}
            />
            {errors.email && (
              <p className="mt-1 text-xs text-red-600">{errors.email.message}</p>
            )}
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Пароль
            </label>
            <input
              type="password"
              autoComplete="current-password"
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              {...register('password')}
            />
            {errors.password && (
              <p className="mt-1 text-xs text-red-600">{errors.password.message}</p>
            )}
          </div>
          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white font-medium py-2 rounded-lg text-sm transition-colors"
          >
            {isSubmitting ? 'Вход...' : 'Войти'}
          </button>
        </form>
      </div>
      </div>
    </div>
  )
}
