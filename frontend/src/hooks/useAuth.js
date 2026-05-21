import useAuthStore from '../stores/auth'

export default function useAuth() {
  const user = useAuthStore((s) => s.user)
  const accessToken = useAuthStore((s) => s.accessToken)
  return {
    user,
    isAuthenticated: !!accessToken,
    isAdmin: user?.role === 'admin',
    isDoctor: user?.role === 'doctor',
  }
}
