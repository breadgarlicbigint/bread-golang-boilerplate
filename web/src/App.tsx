import { BrowserRouter, Route, Routes } from "react-router-dom";
import { AuthProvider } from "./context/AuthContext";
import { SettingsProvider } from "./context/SettingsContext";
import { ToastProvider } from "./context/ToastContext";
import { Layout } from "./components/Layout";
import { ProtectedRoute, AdminRoute } from "./components/ProtectedRoute";
import { SessionWatcher } from "./components/SessionWatcher";

import { LoginPage } from "./pages/LoginPage";
import { RegisterPage } from "./pages/RegisterPage";
import { DashboardPage } from "./pages/DashboardPage";
import { ProfilePage } from "./pages/ProfilePage";
import { TwoFAPage } from "./pages/TwoFAPage";
import { OAuthPage } from "./pages/OAuthPage";
import { PasskeysPage } from "./pages/PasskeysPage";
import { MobilePage } from "./pages/MobilePage";
import { NotificationsPage } from "./pages/NotificationsPage";
import { AdminUsersPage } from "./pages/AdminUsersPage";
import { AdminAppVersionsPage } from "./pages/AdminAppVersionsPage";
import { AdminAnalyticsPage } from "./pages/AdminAnalyticsPage";
import { AppVersionCheckPage } from "./pages/AppVersionCheckPage";
import { HealthPage } from "./pages/HealthPage";
import { ApiConsolePage } from "./pages/ApiConsolePage";
import { I18nComparePage } from "./pages/I18nComparePage";
import { ActivityLogPage } from "./pages/ActivityLogPage";
import { SettingsPage } from "./pages/SettingsPage";
import { NotFoundPage } from "./pages/NotFoundPage";

export function App() {
  return (
    <SettingsProvider>
      <AuthProvider>
        <ToastProvider>
          <BrowserRouter>
            <SessionWatcher />
            <Routes>
              <Route element={<Layout />}>
                <Route path="/login" element={<LoginPage />} />
                <Route path="/register" element={<RegisterPage />} />
                <Route path="/app-version-check" element={<AppVersionCheckPage />} />
                <Route path="/health" element={<HealthPage />} />
                <Route path="/console" element={<ApiConsolePage />} />
                <Route path="/i18n-compare" element={<I18nComparePage />} />
                <Route path="/activity" element={<ActivityLogPage />} />
                <Route path="/settings" element={<SettingsPage />} />
                <Route path="/" element={<DashboardPage />} />

                <Route element={<ProtectedRoute />}>
                  <Route path="/profile" element={<ProfilePage />} />
                  <Route path="/2fa" element={<TwoFAPage />} />
                  <Route path="/passkeys" element={<PasskeysPage />} />
                  <Route path="/mobile" element={<MobilePage />} />
                  <Route path="/notifications" element={<NotificationsPage />} />
                  <Route path="/oauth" element={<OAuthPage />} />

                  <Route element={<AdminRoute />}>
                    <Route path="/admin/users" element={<AdminUsersPage />} />
                    <Route path="/admin/app-versions" element={<AdminAppVersionsPage />} />
                    <Route path="/admin/analytics" element={<AdminAnalyticsPage />} />
                  </Route>
                </Route>

                <Route path="*" element={<NotFoundPage />} />
              </Route>
            </Routes>
          </BrowserRouter>
        </ToastProvider>
      </AuthProvider>
    </SettingsProvider>
  );
}
