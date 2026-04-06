import { useEffect } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AppShell } from './components/layout/AppShell';
import { ProtectedRoute } from './components/shared/ProtectedRoute';
import { useAuthStore } from './store/authStore';
import { LoginPage } from './pages/LoginPage';
import { DashboardPage } from './pages/DashboardPage';
import { CatalogPage } from './pages/CatalogPage';
import { CollectiblePage } from './pages/CollectiblePage';
import { CollectibleFormPage } from './pages/CollectibleFormPage';
import { OrdersPage } from './pages/OrdersPage';
import { OrderDetailPage } from './pages/OrderDetailPage';
import { MessagesPage } from './pages/MessagesPage';
import { NotificationsPage } from './pages/NotificationsPage';
import { NotificationPrefsPage } from './pages/NotificationPrefsPage';
import { AnalyticsPage } from './pages/AnalyticsPage';
import { ABTestPage } from './pages/ABTestPage';
import { AdminUsersPage } from './pages/AdminUsersPage';
import { AdminModerationPage } from './pages/AdminModerationPage';
import { AnomalyAlertsPage } from './pages/AnomalyAlertsPage';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30000,
      retry: 1,
    },
  },
});

export function App() {
  const bootstrap = useAuthStore((s) => s.bootstrap);

  useEffect(() => {
    bootstrap();
  }, [bootstrap]);

  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            element={
              <ProtectedRoute>
                <AppShell />
              </ProtectedRoute>
            }
          >
            <Route path="/" element={<DashboardPage />} />
            <Route path="/catalog" element={<CatalogPage />} />
            <Route path="/catalog/:id" element={<CollectiblePage />} />
            <Route path="/orders" element={<OrdersPage />} />
            <Route path="/orders/:id" element={<OrderDetailPage />} />
            <Route path="/orders/:id/messages" element={<MessagesPage />} />
            <Route path="/notifications" element={<NotificationsPage />} />
            <Route path="/notifications/prefs" element={<NotificationPrefsPage />} />
            <Route
              path="/sell/new"
              element={
                <ProtectedRoute roles={['seller']}>
                  <CollectibleFormPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/sell/:id/edit"
              element={
                <ProtectedRoute roles={['seller']}>
                  <CollectibleFormPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/analytics"
              element={
                <ProtectedRoute roles={['administrator', 'compliance_analyst']}>
                  <AnalyticsPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/analytics/ab-tests"
              element={
                <ProtectedRoute roles={['administrator', 'compliance_analyst']}>
                  <ABTestPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/users"
              element={
                <ProtectedRoute roles={['administrator']}>
                  <AdminUsersPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/moderation"
              element={
                <ProtectedRoute roles={['administrator', 'compliance_analyst']}>
                  <AdminModerationPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/anomalies"
              element={
                <ProtectedRoute roles={['compliance_analyst', 'administrator']}>
                  <AnomalyAlertsPage />
                </ProtectedRoute>
              }
            />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
