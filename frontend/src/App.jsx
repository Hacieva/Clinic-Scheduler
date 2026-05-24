import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import LoginPage from './pages/LoginPage'
import RequireAuth from './components/RequireAuth'
import Layout from './components/Layout'
import ScheduleGridPage from './pages/admin/ScheduleGridPage'
import DirectionsPage from './pages/admin/DirectionsPage'
import DoctorsPage from './pages/admin/DoctorsPage'
import DoctorDetailPage from './pages/admin/DoctorDetailPage'
import AppointmentsPage from './pages/admin/AppointmentsPage'
import PatientsPage from './pages/admin/PatientsPage'
import PatientDetailPage from './pages/admin/PatientDetailPage'
import SchedulePage from './pages/doctor/SchedulePage'
import BranchesPage from './pages/admin/settings/BranchesPage'
import UsersPage from './pages/admin/settings/UsersPage'
import IntegrationsPage from './pages/admin/settings/IntegrationsPage'
import PricesPage from './pages/admin/settings/PricesPage'
import ServicesSettingsPage from './pages/admin/settings/ServicesPage'
import LabPage from './pages/admin/settings/LabPage'
import ClinicPage from './pages/admin/settings/ClinicPage'
import SettingsDashboard from './pages/admin/settings/SettingsDashboard'
import DashboardPage from './pages/admin/DashboardPage'
import CashboxPage from './pages/admin/cashbox/CashboxPage'
import WalkInPage from './pages/admin/cashbox/WalkInPage'
import ReportsPage from './pages/admin/ReportsPage'
import CrmPage from './pages/admin/CrmPage'
import ServicesPage from './pages/admin/ServicesPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />

        {/* Admin + Owner routes */}
        <Route element={<RequireAuth allowedRoles={['admin', 'owner']} />}>
          <Route element={<Layout />}>
            {/* Main */}
            <Route path="/admin/dashboard" element={<DashboardPage />} />
            <Route path="/admin/schedule-grid" element={<ScheduleGridPage />} />
            <Route path="/admin/appointments" element={<AppointmentsPage />} />
            <Route path="/admin/patients" element={<PatientsPage />} />
            <Route path="/admin/patients/:id" element={<PatientDetailPage />} />
            <Route path="/admin/doctors" element={<DoctorsPage />} />
            <Route path="/admin/doctors/:id" element={<DoctorDetailPage />} />
            <Route path="/admin/reports" element={<ReportsPage />} />
            <Route path="/admin/crm" element={<CrmPage />} />
            <Route path="/admin/services" element={<ServicesPage />} />

            {/* Redirect old bookmarks */}
            <Route path="/admin/directions" element={<Navigate to="/admin/settings/directions" replace />} />

            {/* Cashbox — routes kept, removed from main nav */}
            <Route path="/admin/cashbox" element={<CashboxPage />} />
            <Route path="/admin/cashbox/walk-in" element={<WalkInPage />} />

            {/* Settings */}
            <Route path="/admin/settings" element={<SettingsDashboard />} />
            <Route path="/admin/settings/clinic" element={<ClinicPage />} />
            <Route path="/admin/settings/branches" element={<BranchesPage />} />
            <Route path="/admin/settings/users" element={<UsersPage />} />
            <Route path="/admin/settings/directions" element={<DirectionsPage />} />
            <Route path="/admin/settings/integrations" element={<IntegrationsPage />} />
            <Route path="/admin/settings/services" element={<ServicesSettingsPage />} />
            <Route path="/admin/settings/prices" element={<PricesPage />} />
            <Route path="/admin/settings/lab" element={<LabPage />} />

            {/* Default admin redirect */}
            <Route path="/admin" element={<Navigate to="/admin/schedule-grid" replace />} />
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
