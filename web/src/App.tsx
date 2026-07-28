import { Navigate, Route, Routes } from 'react-router-dom';
import { useAuth } from './auth';
import Layout from './components/Layout';
import AuthPage from './pages/AuthPage';
import SetupPage from './pages/SetupPage';
import Dashboard from './pages/Dashboard';
import Portal from './pages/Portal';
import Profile from './pages/Profile';
import Catalog from './pages/Catalog';
import Collections from './pages/Collections';
import CollectionDetail from './pages/CollectionDetail';
import ToolFormPage from './pages/ToolFormPage';
import ToolDetail from './pages/ToolDetail';
import Hosts from './pages/Hosts';
import HostInventory from './pages/HostInventory';
import Requests from './pages/Requests';
import AdminUsers from './pages/AdminUsers';
import AdminGroups from './pages/AdminGroups';
import AdminWebhooks from './pages/AdminWebhooks';
import AdminAlerts from './pages/AdminAlerts';
import AdminAudit from './pages/AdminAudit';
import AdminServerInfo from './pages/AdminServerInfo';
import AdminSettings from './pages/AdminSettings';
import { ForgotPassword, ResetPassword } from './pages/ForgotPassword';
import type { ReactNode } from 'react';

function Protected({ children, admin }: { children: ReactNode; admin?: boolean }) {
  const { user, loading, status } = useAuth();
  if (loading) {
    return <div className="flex min-h-screen items-center justify-center text-muted">Loading…</div>;
  }
  if (!user) {
    return <Navigate to={status.setup_required ? '/setup' : '/login'} replace />;
  }
  if (admin && user.role !== 'admin') {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}

export default function App() {
  const { user, status } = useAuth();
  // First run: no accounts exist yet — force the admin setup screen.
  const gate = (page: ReactNode) => {
    if (user) {
      return <Navigate to="/" replace />;
    }
    if (status.setup_required) {
      return <Navigate to="/setup" replace />;
    }
    return page;
  };
  return (
    <Routes>
      <Route
        path="/setup"
        element={
          user || !status.setup_required ? <Navigate to="/" replace /> : <SetupPage />
        }
      />
      <Route path="/login" element={gate(<AuthPage mode="login" />)} />
      <Route path="/forgot" element={gate(<ForgotPassword />)} />
      <Route path="/reset" element={gate(<ResetPassword />)} />
      <Route path="/signup" element={gate(<AuthPage mode="signup" />)} />
      <Route path="/" element={<Portal />} />
      <Route
        element={
          <Protected>
            <Layout />
          </Protected>
        }
      >
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/profile" element={<Profile />} />
        <Route path="/services" element={<Catalog />} />
        <Route path="/services/new" element={<ToolFormPage />} />
        <Route path="/services/:id" element={<ToolDetail />} />
        <Route path="/services/:id/edit" element={<ToolFormPage />} />
        <Route path="/collections" element={<Collections />} />
        <Route path="/collections/:id" element={<CollectionDetail />} />
        <Route path="/hosts" element={<Hosts />} />
        <Route path="/hosts/:id" element={<HostInventory />} />
        <Route path="/requests" element={<Requests />} />
        <Route
          path="/admin/users"
          element={
            <Protected admin>
              <AdminUsers />
            </Protected>
          }
        />
        {/* Not admin-gated: a group's moderator manages its members from this page,
            and the API scopes both the list and every write to what they may touch. */}
        <Route path="/admin/groups" element={<AdminGroups />} />
        <Route
          path="/admin/webhooks"
          element={
            <Protected admin>
              <AdminWebhooks />
            </Protected>
          }
        />
        <Route
          path="/admin/alerts"
          element={
            <Protected admin>
              <AdminAlerts />
            </Protected>
          }
        />
        <Route
          path="/admin/audit"
          element={
            <Protected admin>
              <AdminAudit />
            </Protected>
          }
        />
        <Route
          path="/admin/settings"
          element={
            <Protected admin>
              <AdminSettings />
            </Protected>
          }
        />
        <Route
          path="/admin/server"
          element={
            <Protected admin>
              <AdminServerInfo />
            </Protected>
          }
        />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
