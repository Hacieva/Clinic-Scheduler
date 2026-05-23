import { useState, useEffect } from 'react'
import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  BookOpen, Users, Stethoscope, ClipboardList, LogOut, CalendarDays, UserRound,
  LayoutGrid, Settings, Building2, UserCog, Plug2, Tag, FlaskConical, ChevronDown, BarChart3,
} from 'lucide-react'
import useAuthStore from '../stores/auth'
import useBranchStore from '../stores/branch'
import { logout } from '../api/auth'
import { getBranches } from '../api/branches'

// ─── Nav definitions ──────────────────────────────────────────────────────────

const MAIN_NAV = [
  { to: '/admin/dashboard',     label: 'Сводка',        icon: BarChart3 },
  { to: '/admin/schedule-grid', label: 'Журнал записи', icon: LayoutGrid },
  { to: '/admin/appointments',  label: 'Записи',        icon: ClipboardList },
  { to: '/admin/patients',      label: 'Пациенты',      icon: UserRound },
  { to: '/admin/doctors',       label: 'Врачи',         icon: Users },
  { to: '/admin/directions',    label: 'Направления',   icon: BookOpen },
]

const SETTINGS_NAV = [
  { to: '/admin/settings/branches',     label: 'Филиалы',       icon: Building2 },
  { to: '/admin/settings/users',        label: 'Пользователи',  icon: UserCog },
  { to: '/admin/settings/integrations', label: 'Интеграции',    icon: Plug2 },
  { to: '/admin/settings/prices',       label: 'Прайсы',        icon: Tag },
  { to: '/admin/settings/lab',          label: 'Лаборатория',   icon: FlaskConical },
]

const DOCTOR_NAV = [
  { to: '/doctor/schedule', label: 'Расписание', icon: CalendarDays },
]

// ─── NavItem ──────────────────────────────────────────────────────────────────

function NavItem({ to, label, icon: Icon, small = false }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `flex items-center gap-3 rounded-lg font-medium transition-colors ${
          small ? 'px-2.5 py-1.5 text-[13px]' : 'px-3 py-2 text-sm'
        } ${
          isActive
            ? 'bg-blue-50 text-blue-700'
            : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
        }`
      }
    >
      <Icon size={small ? 15 : 17} />
      {label}
    </NavLink>
  )
}

// ─── Layout ───────────────────────────────────────────────────────────────────

export default function Layout() {
  const navigate = useNavigate()
  const location = useLocation()
  const user = useAuthStore((s) => s.user)
  const clearTokens = useAuthStore((s) => s.clearTokens)
  const activeBranchId = useBranchStore((s) => s.activeBranchId)
  const setActiveBranchId = useBranchStore((s) => s.setActiveBranchId)

  const isDoctor = user?.role === 'doctor'
  const isOwner = user?.role === 'owner'

  // Settings accordion — auto-open when on a settings route
  const onSettingsRoute = location.pathname.startsWith('/admin/settings')
  const [settingsOpen, setSettingsOpen] = useState(onSettingsRoute)
  useEffect(() => {
    if (onSettingsRoute) setSettingsOpen(true)
  }, [onSettingsRoute])

  // Branches for switcher — only fetched for owner
  const { data: branches = [] } = useQuery({
    queryKey: ['branches'],
    queryFn: getBranches,
    enabled: isOwner,
  })

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

        {/* ── Sidebar header ── */}
        <div className="p-5 border-b border-gray-200 shrink-0">
          <div className="flex items-center gap-3 mb-2">
            <div className="flex items-center justify-center w-8 h-8 bg-blue-600 rounded-lg shrink-0">
              <Stethoscope size={16} className="text-white" />
            </div>
            <h1 className="text-base font-semibold text-gray-900 leading-tight">Clinic Scheduler</h1>
          </div>
          <p className="text-xs text-gray-400 truncate">{user?.email}</p>

          {/* Branch switcher — owner only, when branches exist */}
          {isOwner && branches.length > 0 && (
            <div className="flex items-center gap-1.5 mt-2.5">
              <Building2 size={12} className="text-gray-400 shrink-0" />
              <select
                value={activeBranchId ?? ''}
                onChange={(e) =>
                  setActiveBranchId(e.target.value ? Number(e.target.value) : null)
                }
                className="flex-1 min-w-0 text-xs text-gray-600 bg-gray-50 border border-gray-200 rounded-md px-2 py-1 focus:outline-none focus:ring-1 focus:ring-blue-400"
              >
                <option value="">Все филиалы</option>
                {branches.map((b) => (
                  <option key={b.id} value={b.id}>
                    {b.name}
                  </option>
                ))}
              </select>
            </div>
          )}
        </div>

        {/* ── Navigation ── */}
        <nav className="flex-1 overflow-y-auto py-3 px-3 space-y-1">
          {isDoctor ? (
            // Doctor — minimal nav
            DOCTOR_NAV.map(({ to, label, icon: Icon }) => (
              <NavItem key={to} to={to} label={label} icon={Icon} />
            ))
          ) : (
            <>
              {/* Main nav */}
              {MAIN_NAV.map(({ to, label, icon: Icon }) => (
                <NavItem key={to} to={to} label={label} icon={Icon} />
              ))}

              {/* Settings section */}
              <div className="pt-2 mt-1">
                <div className="mx-1 mb-2 border-t border-gray-100" />

                {/* Accordion toggle */}
                <button
                  onClick={() => setSettingsOpen((v) => !v)}
                  className={`flex items-center justify-between w-full px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                    onSettingsRoute
                      ? 'bg-blue-50 text-blue-700'
                      : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
                  }`}
                >
                  <span className="flex items-center gap-3">
                    <Settings size={17} />
                    Настройки
                  </span>
                  <ChevronDown
                    size={14}
                    className={`transition-transform duration-200 text-gray-400 ${
                      settingsOpen ? 'rotate-180' : 'rotate-0'
                    }`}
                  />
                </button>

                {/* Settings sub-items */}
                {settingsOpen && (
                  <div className="mt-1 ml-3 pl-3 border-l border-gray-100 space-y-0.5">
                    {SETTINGS_NAV.map(({ to, label, icon: Icon }) => (
                      <NavItem key={to} to={to} label={label} icon={Icon} small />
                    ))}
                  </div>
                )}
              </div>
            </>
          )}
        </nav>

        {/* ── Footer ── */}
        <div className="p-4 border-t border-gray-200 shrink-0">
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
