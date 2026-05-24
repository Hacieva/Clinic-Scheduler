import client from './client'

// ─── Global catalog ───────────────────────────────────────────────────────────

export const getAllServices = (activeOnly = true) =>
  client.get('/services', { params: { active_only: activeOnly } }).then((r) => r.data)

export const createService = (data) =>
  client.post('/services', data).then((r) => r.data)

export const updateService = (id, data) =>
  client.put(`/services/${id}`, data).then((r) => r.data)

export const deleteService = (id) =>
  client.delete(`/services/${id}`)

// ─── Doctor assignments (junction-based) ──────────────────────────────────────

export const getAssignedServices = (doctorId) =>
  client.get(`/doctors/${doctorId}/assigned-services`).then((r) => r.data)

export const setDoctorServices = (doctorId, serviceIds) =>
  client.put(`/doctors/${doctorId}/assigned-services`, { service_ids: serviceIds })

export const assignDoctorService = (doctorId, serviceId) =>
  client.post(`/doctors/${doctorId}/assigned-services/${serviceId}`)

export const unassignDoctorService = (doctorId, serviceId) =>
  client.delete(`/doctors/${doctorId}/assigned-services/${serviceId}`)

// ─── Legacy (bot backward compat — do not use in new code) ───────────────────

/** @deprecated Use getAssignedServices instead */
export const getDoctorServices = (doctorId) =>
  client.get(`/doctors/${doctorId}/services`).then((r) => r.data)

/** @deprecated Use catalog + assignDoctorService flow instead */
export const createDoctorService = (doctorId, data) =>
  client.post(`/doctors/${doctorId}/services`, data).then((r) => r.data)

/** @deprecated Use updateService instead */
export const updateDoctorService = (doctorId, serviceId, data) =>
  client.put(`/doctors/${doctorId}/services/${serviceId}`, data).then((r) => r.data)

/** @deprecated Use deleteService instead */
export const deleteDoctorService = (doctorId, serviceId) =>
  client.delete(`/doctors/${doctorId}/services/${serviceId}`)
