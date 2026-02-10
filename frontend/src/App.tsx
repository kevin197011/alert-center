import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useAuthStore } from './store/auth';
import Layout from './components/Layout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import AlertRules from './pages/AlertRules';
import AlertChannels from './pages/AlertChannels';
import AlertTemplates from './pages/AlertTemplates';
import AlertHistory from './pages/AlertHistory';
import UserManagement from './pages/UserManagement';
import AuditLogs from './pages/AuditLogs';
import DataSources from './pages/DataSources';
import Statistics from './pages/Statistics';
import Settings from './pages/Settings';
import AlertSilences from './pages/AlertSilences';
import SLAConfigs from './pages/SLAConfigs';
import OnCallSchedules from './pages/OnCallSchedules';
import AlertCorrelation from './pages/AlertCorrelation';
import SLABreaches from './pages/SLABreaches';
import OnCallReport from './pages/OnCallReport';
import EscalationHistory from './pages/EscalationHistory';
import TicketManagement from './pages/TicketManagement';
import { ConfigProvider, theme } from 'antd';
import { useState, useEffect } from 'react';

const PrivateRoute = ({ children }: { children: React.ReactNode }) => {
  const { token } = useAuthStore();
  if (!token) {
    return <Navigate to="/login" replace />;
  }
  return <>{children}</>;
};

function App() {
  const [darkMode, setDarkMode] = useState(() => {
    const saved = localStorage.getItem('darkMode');
    return saved ? JSON.parse(saved) : false;
  });

  useEffect(() => {
    localStorage.setItem('darkMode', JSON.stringify(darkMode));
  }, [darkMode]);

  return (
    <ConfigProvider
      theme={{
        algorithm: darkMode ? theme.darkAlgorithm : theme.defaultAlgorithm,
        token: {
          colorPrimary: '#1890ff',
        },
      }}
    >
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            path="/*"
            element={
              <PrivateRoute>
                <Layout darkMode={darkMode} onToggleDark={() => setDarkMode(!darkMode)}>
                  <Routes>
                    <Route path="/" element={<Dashboard />} />
                    <Route path="/rules" element={<AlertRules />} />
                    <Route path="/channels" element={<AlertChannels />} />
                    <Route path="/templates" element={<AlertTemplates />} />
                    <Route path="/history" element={<AlertHistory />} />
                    <Route path="/users" element={<UserManagement />} />
                    <Route path="/audit-logs" element={<AuditLogs />} />
                    <Route path="/data-sources" element={<DataSources />} />
                    <Route path="/statistics" element={<Statistics />} />
                    <Route path="/silences" element={<AlertSilences />} />
                    <Route path="/sla" element={<SLAConfigs />} />
                    <Route path="/oncall" element={<OnCallSchedules />} />
                    <Route path="/correlation" element={<AlertCorrelation />} />
                    <Route path="/sla-breaches" element={<SLABreaches />} />
                    <Route path="/oncall/report" element={<OnCallReport />} />
                    <Route path="/escalations" element={<EscalationHistory />} />
                    <Route path="/tickets" element={<TicketManagement />} />
                    <Route path="/settings" element={<Settings />} />
                  </Routes>
                </Layout>
              </PrivateRoute>
            }
          />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
}

export default App;
