import axios from 'axios'
import useAuthStore from '../stores/auth'

const client = axios.create({
  baseURL: '/api/v1',
})

client.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

let refreshPromise = null

client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const original = error.config

    if (
      error.response?.status === 401 &&
      !original._retry &&
      !original.url?.includes('/auth/refresh')
    ) {
      original._retry = true

      if (!refreshPromise) {
        const store = useAuthStore.getState()
        refreshPromise = axios
          .post('/api/v1/auth/refresh', { refresh_token: store.refreshToken })
          .then(async (res) => {
            const { access_token, refresh_token } = res.data
            store.setTokens(access_token, refresh_token)
            const meRes = await client.get('/auth/me', { _retry: true })
            useAuthStore.getState().setUser(meRes.data)
          })
          .catch((err) => {
            useAuthStore.getState().clearTokens()
            throw err
          })
          .finally(() => {
            refreshPromise = null
          })
      }

      try {
        await refreshPromise
        return client(original)
      } catch {
        return Promise.reject(error)
      }
    }

    return Promise.reject(error)
  },
)

export default client
