import React, { useEffect, useState } from 'react';
import { apiService, BusLocation } from '../services/apiService';
import { API_ENDPOINTS } from '../config/api';
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

      <div className="map-placeholder">
        <div className="map-note">
          <h3>🗺️ Map Integration</h3>
          <p>To use Google Maps or Leaflet, install the required packages:</p>
          <pre>npm install leaflet react-leaflet</pre>
          <p>Or</p>
          <pre>npm install @react-google-maps/api</pre>
          <p>Then replace this placeholder with the map component.</p>
        </div>

        <div className="bus-list">
          <h3>Active Buses</h3>
          {Array.from(buses.values()).length === 0 ? (
            <p className="empty-message">No buses currently active</p>
          ) : (
            <div className="bus-cards">
              {Array.from(buses.values()).map((bus) => (
                <div key={bus.busId} className="bus-card">
                  <h4>{bus.routeName || 'No Route'}</h4>
                  <p>Bus ID: {bus.busId}</p>
                  <p>Driver: {bus.driverName || 'Unassigned'}</p>
                  <p>
                    Location: {bus.latitude.toFixed(4)}, {bus.longitude.toFixed(4)}
                  </p>
                  <p className="timestamp">
                    Last Update: {new Date(bus.timestamp).toLocaleTimeString()}
                  </p>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
