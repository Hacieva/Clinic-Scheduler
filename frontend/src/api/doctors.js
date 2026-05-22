import client from './client'

export const getDoctors = (params) =>
  client.get('/doctors', { params }).then((r) => r.data)

export const getDoctorById = (id) =>
  client.get(`/doctors/${id}`).then((r) => r.data)

export const createDoctor = (data) =>
  client.post('/doctors', data).then((r) => r.data)

export const updateDoctor = (id, data) =>
  client.patch(`/doctors/${id}`, data).then((r) => r.data)

export const deleteDoctor = (id) =>
  client.delete(`/doctors/${id}`)

export const setDoctorDirections = (id, directionIds) =>
  client.put(`/doctors/${id}/directions`, { direction_ids: directionIds })

export const createDoctorAccount = (doctorId, email, password) =>
  client
    .post(`/doctors/${doctorId}/account`, { email, password })
    .then((r) => r.data)
