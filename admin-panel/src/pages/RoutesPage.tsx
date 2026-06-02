import React, { useState, useEffect } from 'react';
import { apiService, Route } from '../services/apiService';
import './RoutesPage.css';

export const RoutesPage: React.FC = () => {
  const [routes, setRoutes] = useState<Route[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadRoutes();
  }, []);

  const loadRoutes = async () => {
    setIsLoading(true);
    setError('');
    try {
      const data = await apiService.getRoutes();
      setRoutes(data);
    } catch (err: any) {
      setError('Failed to load routes');
    } finally {
      setIsLoading(false);
    }
  };

  const handleDelete = async (id: string, name: string) => {
    if (!window.confirm(`Delete route "${name}"?`)) return;

    try {
      await apiService.deleteRoute(id);
      await loadRoutes();
    } catch (err: any) {
      if (err.response?.status === 409) {
        alert('Cannot delete route with active bus assignment');
      } else {
        alert('Failed to delete route');
      }
    }
  };

  if (isLoading) {
    return (
      <div className="routes-page">
        <div className="loading">Loading routes...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="routes-page">
        <div className="error-container">
          <p>{error}</p>
          <button onClick={loadRoutes} className="btn-primary">
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="routes-page">
      <div className="page-header">
        <h1>Route Management</h1>
        <button className="btn-primary" onClick={() => alert('Create route form - implement as needed')}>
          + Create Route
        </button>
      </div>

      {routes.length === 0 ? (
        <div className="empty-state">
          <p>No routes found</p>
          <button className="btn-primary" onClick={() => alert('Create route')}>
            Create First Route
          </button>
        </div>
      ) : (
        <div className="routes-grid">
          {routes.map((route) => (
            <div key={route.id} className="route-card">
              <div className="route-header">
                <h3>{route.name}</h3>
                <div className="route-actions">
                  <button
                    className="btn-icon"
                    onClick={() => alert('Edit route')}
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
      )}
    </div>
  );
};
