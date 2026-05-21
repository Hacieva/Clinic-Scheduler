import client from './client'

export const getAppointments = (params) =>
  client.get('/appointments', { params }).then((r) => r.data)

export const createAppointment = (data) =>
  client.post('/appointments', data).then((r) => r.data)

export const confirmAppointment = (id) =>
  client.post(`/appointments/${id}/confirm`)

export const cancelAppointment = (id, comment) =>
  client.post(`/appointments/${id}/cancel`, comment ? { comment } : {})

export const completeAppointment = (id) =>
  client.post(`/appointments/${id}/complete`)

export const noShowAppointment = (id) =>
  client.post(`/appointments/${id}/no-show`)
