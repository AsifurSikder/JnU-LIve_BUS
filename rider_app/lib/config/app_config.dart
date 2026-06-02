class AppConfig {
  // API Configuration
  static const String apiBaseUrl = 'http://localhost:8080';
  static const String wsUrl = 'ws://localhost:8080/ws/location';
  
  // Endpoints
  static const String locationBusesEndpoint = '/location/buses';
  
  // Map Configuration
  static const double defaultLatitude = 23.7104; // JnU coordinates
  static const double defaultLongitude = 90.4074;
  static const double defaultZoom = 14.0;
  
  // Reconnection Configuration
  static const int maxReconnectAttempts = 3;
  static const int reconnectIntervalSeconds = 5;
  
  // App Info
  static const String appName = 'JnU Bus Tracker - Rider';
  static const String appVersion = '1.0.0';
}
