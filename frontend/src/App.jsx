import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import LoginPage from './pages/LoginPage'
import RequireAuth from './components/RequireAuth'
import Layout from './components/Layout'
import DirectionsPage from './pages/admin/DirectionsPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />

        {/* Admin routes */}
        <Route element={<RequireAuth allowedRoles={['admin']} />}>
          <Route element={<Layout />}>
            <Route path="/admin/directions" element={<DirectionsPage />} />
            <Route path="/admin" element={<Navigate to="/admin/directions" replace />} />
          </Route>
        </Route>

        {/* Doctor routes — placeholder until 8.3+ */}
        <Route element={<RequireAuth allowedRoles={['doctor']} />}>
          <Route
            path="/doctor/*"
            element={<div className="p-8 text-gray-500">Doctor panel — coming soon</div>}
          />
        </Route>

        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
