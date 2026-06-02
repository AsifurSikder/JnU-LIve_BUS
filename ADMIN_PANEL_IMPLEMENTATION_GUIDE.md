# Admin Panel Implementation Guide (React + TypeScript)

Complete implementation guide for the JnU Live Bus Tracker Admin Panel.

---

## 📦 Step 1: Install Dependencies

```bash
cd admin-panel
npm install axios react-router-dom @tanstack/react-query leaflet react-leaflet
npm install -D @types/leaflet
```

---

## 🔧 Step 2: Create API Configuration

Create `src/config/api.ts`:

```typescript
export const API_BASE_URL = 'http://localhost:8080';

export const API_ENDPOINTS = {
  LOGIN: '/auth/login',
  ROUTES: '/routes',
  LOCATION_BUSES: '/location/buses',
  WS_LOCATION: 'ws://localhost:8080/ws/location',
};

export const getAuthHeaders = (): Record<string, string> => {
  const token = localStorage.getItem('jwt_token');
  return token ? { 'Authorization': `Bearer ${token}` } : {};
};
```

---

## 🔐 Step 3: Create API Service

Create `src/services/apiService.ts`:

```typescript
import axios from 'axios';
import { API_BASE_URL, API_ENDPOINTS, getAuthHeaders } from '../config/api';

export interface LoginResponse {
  token: string;
  expiresAt: string;
  role: string;
  userId: string;
}

export interface Route {
  id: string;
  name: string;
  stops: Stop[];
  assignedBusId?: string;
  assignedDriverName?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Stop {
  id: string;
  name: string;
  latitude: number;
  longitude: number;
  stopOrder: number;
}

export interface CreateRouteRequest {
  name: string;
  stops: Array<{
    name: string;
    latitude: number;
    longitude: number;
  }>;
}

export interface BusLocation {
  busId: string;
  latitude: number;
  longitude: number;
  timestamp: string;
  routeId?: string;
  routeName?: string;
  driverName?: string;
}

class ApiService {
  async login(username: string, password: string): Promise<LoginResponse> {
    const response = await axios.post(`${API_BASE_URL}${API_ENDPOINTS.LOGIN}`, {
      username,
      password,
      role: 'admin',
    });
    return response.data;
  }

  async getRoutes(): Promise<Route[]> {
    const response = await axios.get(`${API_BASE_URL}${API_ENDPOINTS.ROUTES}`, {
      headers: getAuthHeaders(),
    });
    return response.data.routes || [];
  }

  async createRoute(route: CreateRouteRequest): Promise<Route> {
    const response = await axios.post(
      `${API_BASE_URL}${API_ENDPOINTS.ROUTES}`,
      route,
      { headers: getAuthHeaders() }
    );
    return response.data;
  }

  async updateRoute(id: string, route: CreateRouteRequest): Promise<Route> {
    const response = await axios.put(
      `${API_BASE_URL}${API_ENDPOINTS.ROUTES}/${id}`,
      route,
      { headers: getAuthHeaders() }
    );
    return response.data;
  }

  async deleteRoute(id: string): Promise<void> {
    await axios.delete(`${API_BASE_URL}${API_ENDPOINTS.ROUTES}/${id}`, {
      headers: getAuthHeaders(),
    });
  }

  async assignBus(
    routeId: string,
    busId: string,
    driverId: string
  ): Promise<void> {
    await axios.post(
      `${API_BASE_URL}${API_ENDPOINTS.ROUTES}/${routeId}/assign`,
      { busId, driverId },
      { headers: getAuthHeaders() }
    );
  }

  async getAllBuses(): Promise<BusLocation[]> {
    const response = await axios.get(
      `${API_BASE_URL}${API_ENDPOINTS.LOCATION_BUSES}`,
      { headers: getAuthHeaders() }
    );
    return response.data.buses || [];
  }
}

export const apiService = new ApiService();
```

---

## 🔐 Step 4: Create Login Page

Create `src/pages/LoginPage.tsx`:

```typescript
import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { apiService } from '../services/apiService';
import './LoginPage.css';

export const LoginPage: React.FC = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      const response = await apiService.login(username, password);
      localStorage.setItem('jwt_token', response.token);
      localStorage.setItem('user_id', response.userId);
      navigate('/dashboard');
    } catch (err: any) {
      if (err.response?.status === 401) {
        setError('Invalid credentials');
      } else if (err.response?.status === 429) {
        setError('Too many attempts. Please try again later');
      } else {
        setError('Login failed. Please try again');
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <div className="login-header">
          <h1>🚌 JnU Bus Tracker</h1>
          <p>Admin Panel</p>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="username">Username</label>
            <input
              type="text"
              id="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="password">Password</label>
            <input
              type="password"
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>

          {error && <div className="error-message">{error}</div>}

          <button type="submit" disabled={isLoading} className="btn-primary">
            {isLoading ? 'Logging in...' : 'Login'}
          </button>
        </form>
      </div>
    </div>
  );
};
```

Create `src/pages/LoginPage.css`:

```css
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
  background: white;
  padding: 2rem;
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
  width: 100%;
  max-width: 400px;
}

.login-header {
  text-align: center;
  margin-bottom: 2rem;
}

.login-header h1 {
  margin: 0 0 0.5rem 0;
  color: #333;
}

.login-header p {
  margin: 0;
  color: #666;
  font-size: 0.9rem;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  color: #333;
}

.form-group input {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 1rem;
}

.form-group input:focus {
  outline: none;
  border-color: #667eea;
}

.error-message {
  background: #fee;
  color: #c33;
  padding: 0.75rem;
  border-radius: 6px;
  margin-bottom: 1rem;
  text-align: center;
}

.btn-primary {
  width: 100%;
  padding: 0.75rem;
  background: #667eea;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 1rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary:hover:not(:disabled) {
  background: #5568d3;
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
```

---

## 🛤️ Step 5: Create Routes Page

Create `src/pages/RoutesPage.tsx`:

```typescript
import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiService, Route, CreateRouteRequest } from '../services/apiService';
import { RouteForm } from '../components/RouteForm';
import './RoutesPage.css';

export const RoutesPage: React.FC = () => {
  const [showForm, setShowForm] = useState(false);
  const [editingRoute, setEditingRoute] = useState<Route | null>(null);
  const queryClient = useQueryClient();

  const { data: routes, isLoading } = useQuery({
    queryKey: ['routes'],
    queryFn: () => apiService.getRoutes(),
  });

  const createMutation = useMutation({
    mutationFn: (route: CreateRouteRequest) => apiService.createRoute(route),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['routes'] });
      setShowForm(false);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, route }: { id: string; route: CreateRouteRequest }) =>
      apiService.updateRoute(id, route),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['routes'] });
      setEditingRoute(null);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => apiService.deleteRoute(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['routes'] });
    },
  });

  const handleDelete = async (id: string, name: string) => {
    if (window.confirm(`Delete route "${name}"?`)) {
      try {
        await deleteMutation.mutateAsync(id);
      } catch (err: any) {
        if (err.response?.status === 409) {
          alert('Cannot delete route with active bus assignment');
        } else {
          alert('Failed to delete route');
        }
      }
    }
  };

  if (isLoading) {
    return <div className="loading">Loading routes...</div>;
  }

  return (
    <div className="routes-page">
      <div className="page-header">
        <h1>Route Management</h1>
        <button
          className="btn-primary"
          onClick={() => setShowForm(true)}
        >
          + Create Route
        </button>
      </div>

      <div className="routes-grid">
        {routes?.map((route) => (
          <div key={route.id} className="route-card">
            <div className="route-header">
              <h3>{route.name}</h3>
              <div className="route-actions">
                <button
                  className="btn-icon"
                  onClick={() => setEditingRoute(route)}
                  title="Edit"
                >
                  ✏️
                </button>
                <button
                  className="btn-icon"
                  onClick={() => handleDelete(route.id, route.name)}
                  title="Delete"
                >
                  🗑️
                </button>
              </div>
            </div>

            <div className="route-info">
              <p>
                <strong>Stops:</strong> {route.stops.length}
              </p>
              {route.assignedBusId && (
                <p>
                  <strong>Assigned Bus:</strong> {route.assignedBusId}
                </p>
              )}
              {route.assignedDriverName && (
                <p>
                  <strong>Driver:</strong> {route.assignedDriverName}
                </p>
              )}
            </div>

            <details className="route-stops">
              <summary>View Stops</summary>
              <ol>
                {route.stops.map((stop) => (
                  <li key={stop.id}>
                    {stop.name} ({stop.latitude.toFixed(4)}, {stop.longitude.toFixed(4)})
                  </li>
                ))}
              </ol>
            </details>
          </div>
        ))}
      </div>

      {showForm && (
        <RouteForm
          onSubmit={(route) => createMutation.mutate(route)}
          onCancel={() => setShowForm(false)}
        />
      )}

      {editingRoute && (
        <RouteForm
          route={editingRoute}
          onSubmit={(route) =>
            updateMutation.mutate({ id: editingRoute.id, route })
          }
          onCancel={() => setEditingRoute(null)}
        />
      )}
    </div>
  );
};
```

---

## 🗺️ Step 6: Create Live Map Page

Create `src/pages/LiveMapPage.tsx`:

```typescript
import React, { useEffect, useState } from 'react';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet';
import { apiService, BusLocation } from '../services/apiService';
import { API_ENDPOINTS } from '../config/api';
import 'leaflet/dist/leaflet.css';
import './LiveMapPage.css';

export const LiveMapPage: React.FC = () => {
  const [buses, setBuses] = useState<Map<string, BusLocation>>(new Map());
  const [isOnline, setIsOnline] = useState(true);
  const [isReconnecting, setIsReconnecting] = useState(false);
  const wsRef = React.useRef<WebSocket | null>(null);

  const connectWebSocket = () => {
    const ws = new WebSocket(API_ENDPOINTS.WS_LOCATION);

    ws.onopen = () => {
      console.log('WebSocket connected');
      setIsOnline(true);
      setIsReconnecting(false);
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === 'location_update') {
          setBuses((prev) => {
            const updated = new Map(prev);
            updated.set(data.busId, data);
            return updated;
          });
        }
      } catch (err) {
        console.error('Error parsing WebSocket message:', err);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
      console.log('WebSocket closed');
      setIsOnline(false);
      // Attempt to reconnect after 5 seconds
      setTimeout(() => {
        setIsReconnecting(true);
        connectWebSocket();
      }, 5000);
    };

    wsRef.current = ws;
  };

  const loadInitialBuses = async () => {
    try {
      const busLocations = await apiService.getAllBuses();
      const busMap = new Map<string, BusLocation>();
      busLocations.forEach((bus) => busMap.set(bus.busId, bus));
      setBuses(busMap);
    } catch (err) {
      console.error('Error loading buses:', err);
    }
  };

  useEffect(() => {
    loadInitialBuses();
    connectWebSocket();

    return () => {
      wsRef.current?.close();
    };
  }, []);

  const dhaka = { lat: 23.7104, lng: 90.4074 };

  return (
    <div className="live-map-page">
      <div className="map-header">
        <h1>Live Bus Tracking</h1>
        <div className="map-stats">
          <span>Active Buses: {buses.size}</span>
        </div>
      </div>

      {!isOnline && (
        <div className="status-banner offline">
          📡 Offline — showing last known location
        </div>
      )}

      {isReconnecting && (
        <div className="status-banner reconnecting">
          🔄 Reconnecting...
        </div>
      )}

      <MapContainer
        center={[dhaka.lat, dhaka.lng]}
        zoom={14}
        style={{ height: 'calc(100vh - 120px)', width: '100%' }}
      >
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />

        {Array.from(buses.values()).map((bus) => (
          <Marker
            key={bus.busId}
            position={[bus.latitude, bus.longitude]}
          >
            <Popup>
              <div>
                <strong>{bus.routeName || 'No Route'}</strong>
                <br />
                Bus: {bus.busId}
                <br />
                Driver: {bus.driverName || 'Unassigned'}
                <br />
                Last Update: {new Date(bus.timestamp).toLocaleTimeString()}
              </div>
            </Popup>
          </Marker>
        ))}
      </MapContainer>
    </div>
  );
};
```

---

## 🎨 Step 7: Update App.tsx

```typescript
import React from 'react';
import {
  BrowserRouter,
  Routes,
  Route,
  Navigate,
  Link,
} from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { LoginPage } from './pages/LoginPage';
import { RoutesPage } from './pages/RoutesPage';
import { LiveMapPage } from './pages/LiveMapPage';
import './App.css';

const queryClient = new QueryClient();

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
    <QueryClientProvider client={queryClient}>
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
    </QueryClientProvider>
  );
}
```

---

## 🚀 Step 8: Run the Admin Panel

```bash
cd admin-panel
npm run dev
```

Open http://localhost:5173 in your browser.

**Default Admin Credentials** (if you seeded the database):
- Username: `admin`
- Password: `admin123`

---

## ✅ Summary

You now have complete implementation guides for:

1. **Driver App (Flutter)** - Login, GPS tracking, offline queue
2. **Rider App (Flutter)** - Live map, WebSocket updates, offline indicator
3. **Admin Panel (React)** - Login, route management, live map

All apps connect to your production-ready backend at `http://localhost:8080`!

