import client from './client'

export const getBranches = () =>
  client.get('/branches').then((r) => r.data)

export const createBranch = (data) =>
  client.post('/branches', data).then((r) => r.data)

export const updateBranch = (id, data) =>
  client.patch(`/branches/${id}`, data).then((r) => r.data)

export const deleteBranch = (id) =>
  client.delete(`/branches/${id}`)
