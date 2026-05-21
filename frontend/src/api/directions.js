import client from './client'

export const getDirections = () =>
  client.get('/directions').then((r) => r.data)

export const createDirection = (data) =>
  client.post('/directions', data).then((r) => r.data)

export const updateDirection = (id, data) =>
  client.put(`/directions/${id}`, data).then((r) => r.data)

export const deleteDirection = (id) =>
  client.delete(`/directions/${id}`)
