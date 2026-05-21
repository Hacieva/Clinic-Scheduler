import client from './client'

export const getDirections = () =>
  client.get('/admin/directions').then((r) => r.data)

export const createDirection = (data) =>
  client.post('/admin/directions', data).then((r) => r.data)

export const updateDirection = (id, data) =>
  client.put(`/admin/directions/${id}`, data).then((r) => r.data)

export const deleteDirection = (id) =>
  client.delete(`/admin/directions/${id}`)
