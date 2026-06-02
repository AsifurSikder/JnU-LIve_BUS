import React from 'react';
import {
  BrowserRouter,
  Routes,
  Route,
  Navigate,
  Link,
} from 'react-router-dom';
import { LoginPage } from './pages/LoginPage';
import { RoutesPage } from './pages/RoutesPage';
import { LiveMapPage } from './pages/LiveMapPage';
import './App.css';

const PrivateRoute: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const token = localStorage.getItem('jwt_token');
  return token ? <>{children}</> : <Navigate to="/login" />;
};

const Layout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const handleLogout = () => {
    localStorage.removeItem('jwt_token');
    localStorage.removeItem('user_id');
    window.location.href = '/login';
  };

  return (
    <div className="app-layout">
      <nav className="sidebar">
        <div className="sidebar-header">
          <h2>🚌 JnU Bus Tracker</h2>
        </div>
        <ul className="nav-links">
          <li>
            <Link to="/dashboard/routes">🛤️ Routes</Link>
          </li>
          <li>
            <Link to="/dashboard/map">🗺️ Live Map</Link>
          </li>
        </ul>
        <button className="btn-logout" onClick={handleLogout}>
          🚪 Logout
        </button>
      </nav>
      <main className="main-content">{children}</main>
    </div>
  );
};

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/dashboard"
          element={
            <PrivateRoute>
              <Layout>
                <Navigate to="/dashboard/routes" />
              </Layout>
            </PrivateRoute>
          }
        />
        <Route
          path="/dashboard/routes"
          element={
            <PrivateRoute>
              <Layout>
                <RoutesPage />
              </Layout>
            </PrivateRoute>
          }
        />
        <Route
          path="/dashboard/map"
          element={
            <PrivateRoute>
              <Layout>
                <LiveMapPage />
              </Layout>
            </PrivateRoute>
          }
        />
        <Route path="/" element={<Navigate to="/dashboard" />} />
      </Routes>
    </BrowserRouter>
  );
}
