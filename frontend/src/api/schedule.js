import client from './client'

export const getWorkingHours = (doctorId) =>
  client.get(`/doctors/${doctorId}/working-hours`).then((r) => r.data)

export const replaceWorkingHours = (doctorId, items) =>
  client.put(`/doctors/${doctorId}/working-hours`, { items })

export const getExceptions = (doctorId, from, to) =>
  client
    .get(`/doctors/${doctorId}/exceptions`, { params: { from, to } })
    .then((r) => r.data)

export const createException = (doctorId, data) =>
  client.post(`/doctors/${doctorId}/exceptions`, data).then((r) => r.data)

export const updateException = (doctorId, exId, data) =>
  client.put(`/doctors/${doctorId}/exceptions/${exId}`, data).then((r) => r.data)

export const deleteException = (doctorId, exId) =>
  client.delete(`/doctors/${doctorId}/exceptions/${exId}`)

export const createExceptionRange = (doctorId, data) =>
  client.post(`/doctors/${doctorId}/exceptions/range`, data).then((r) => r.data)
