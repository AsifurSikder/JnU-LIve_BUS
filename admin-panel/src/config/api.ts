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
