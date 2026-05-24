import { Navigate, Outlet } from 'react-router-dom'
import useAuthStore from '../stores/auth'

export default function RequireAuth({ allowedRoles }) {
  const accessToken = useAuthStore((s) => s.accessToken)
  const user = useAuthStore((s) => s.user)

  if (!accessToken || !user) {
    return <Navigate to="/login" replace />
  }

  if (allowedRoles && !allowedRoles.includes(user.role)) {
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}
