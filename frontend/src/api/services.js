import client from './client'

export const getDoctorServices = (doctorId) =>
  client.get(`/doctors/${doctorId}/services`).then((r) => r.data)

export const createDoctorService = (doctorId, data) =>
  client.post(`/doctors/${doctorId}/services`, data).then((r) => r.data)

export const updateDoctorService = (doctorId, serviceId, data) =>
  client.put(`/doctors/${doctorId}/services/${serviceId}`, data).then((r) => r.data)

export const deleteDoctorService = (doctorId, serviceId) =>
  client.delete(`/doctors/${doctorId}/services/${serviceId}`)
