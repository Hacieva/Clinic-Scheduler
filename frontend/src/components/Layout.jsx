import { Outlet, NavLink, useNavigate } from 'react-router-dom'
import { BookOpen, Users, Stethoscope, ClipboardList, LogOut, CalendarDays, UserRound } from 'lucide-react'
import toast from 'react-hot-toast'
import useAuthStore from '../stores/auth'
import { logout } from '../api/auth'

const adminNav = [
  { to: '/admin/directions', label: 'Направления', icon: BookOpen },
  { to: '/admin/doctors', label: 'Врачи', icon: Users },
  { to: '/admin/appointments', label: 'Записи', icon: ClipboardList },
  { to: '/admin/patients', label: 'Пациенты', icon: UserRound },
]

const doctorNav = [
  { to: '/doctor/schedule', label: 'Расписание', icon: CalendarDays },
]

export default function Layout() {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const clearTokens = useAuthStore((s) => s.clearTokens)
  const nav = user?.role === 'doctor' ? doctorNav : adminNav

  const handleLogout = async () => {
    try {
      await logout()
    } catch {
      // ignore logout errors — clear locally regardless
    } finally {
      clearTokens()
      navigate('/login', { replace: true })
    }
  }

  return (
    <div className="flex h-screen bg-gray-50">
      <aside className="w-64 bg-white border-r border-gray-200 flex flex-col shrink-0">
        <div className="p-5 border-b border-gray-200">
          <div className="flex items-center gap-3 mb-3">
            <div className="flex items-center justify-center w-8 h-8 bg-blue-600 rounded-lg shrink-0">
              <Stethoscope size={16} className="text-white" />
            </div>
            <h1 className="text-base font-semibold text-gray-900 leading-tight">Clinic Scheduler</h1>
          </div>
          <p className="text-xs text-gray-400 truncate">{user?.email}</p>
        </div>
        <nav className="flex-1 p-4 space-y-1">
          {nav.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-blue-50 text-blue-700'
                    : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
                }`
              }
            >
              <Icon size={18} />
              {label}
            </NavLink>
          ))}
        </nav>
        <div className="p-4 border-t border-gray-200">
          <button
            onClick={handleLogout}
            className="flex items-center gap-3 w-full px-3 py-2 rounded-lg text-sm font-medium text-gray-700 hover:bg-gray-100 transition-colors"
          >
            <LogOut size={18} />
            Выйти
          </button>
        </div>
      </aside>
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
