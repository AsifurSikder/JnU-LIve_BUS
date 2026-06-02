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
