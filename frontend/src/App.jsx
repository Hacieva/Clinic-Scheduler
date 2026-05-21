import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import LoginPage from './pages/LoginPage'
import RequireAuth from './components/RequireAuth'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<RequireAuth allowedRoles={['admin']} />}>
          <Route
            path="/admin/*"
            element={<div className="p-8 text-gray-500">Admin panel — coming soon</div>}
          />
        </Route>
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
