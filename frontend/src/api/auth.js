import client from './client'

export const login = (email, password) =>
  client.post('/auth/login', { email, password }).then((r) => r.data)

export const logout = () =>
  client.post('/auth/logout').then((r) => r.data)

export const me = (config) =>
  client.get('/auth/me', config).then((r) => r.data)

export const refresh = (refreshToken) =>
  client.post('/auth/refresh', { refresh_token: refreshToken }).then((r) => r.data)

export const changePassword = (currentPassword, newPassword) =>
  client
    .post('/auth/change-password', {
      current_password: currentPassword,
      new_password: newPassword,
    })
    .then((r) => r.data)
