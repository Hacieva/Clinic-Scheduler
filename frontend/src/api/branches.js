import client from './client'

export const getBranches = () =>
  client.get('/branches').then((r) => r.data)
