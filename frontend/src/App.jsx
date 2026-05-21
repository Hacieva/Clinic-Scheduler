import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import LoginPage from './pages/LoginPage'
import RequireAuth from './components/RequireAuth'
import Layout from './components/Layout'
import DirectionsPage from './pages/admin/DirectionsPage'
import DoctorsPage from './pages/admin/DoctorsPage'
import DoctorDetailPage from './pages/admin/DoctorDetailPage'
import AppointmentsPage from './pages/admin/AppointmentsPage'
import SchedulePage from './pages/doctor/SchedulePage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />

        {/* Admin routes */}
        <Route element={<RequireAuth allowedRoles={['admin']} />}>
          <Route element={<Layout />}>
            <Route path="/admin/directions" element={<DirectionsPage />} />
            <Route path="/admin/doctors" element={<DoctorsPage />} />
            <Route path="/admin/doctors/:id" element={<DoctorDetailPage />} />
            <Route path="/admin/appointments" element={<AppointmentsPage />} />
            <Route path="/admin" element={<Navigate to="/admin/directions" replace />} />
          </Route>
        </Route>

        {/* Doctor routes */}
        <Route element={<RequireAuth allowedRoles={['doctor']} />}>
          <Route element={<Layout />}>
            <Route path="/doctor/schedule" element={<SchedulePage />} />
            <Route path="/doctor" element={<Navigate to="/doctor/schedule" replace />} />
          </Route>
        </Route>

        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
