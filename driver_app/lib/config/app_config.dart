class AppConfig {
  // API Configuration
  static const String apiBaseUrl = 'http://localhost:8080'; // Change to production URL
  
  // Endpoints
  static const String loginEndpoint = '/auth/login';
  static const String routesEndpoint = '/routes';
  static const String locationUpdateEndpoint = '/location/update';
  
  // GPS Configuration
  static const int gpsUpdateIntervalSeconds = 5;
  static const int maxOfflineQueueSize = 500;
  static const int maxTimestampAgeMinutes = 30;
  
  // Storage Keys
  static const String jwtTokenKey = 'jwt_token';
  static const String userIdKey = 'user_id';
  static const String usernameKey = 'username';
  static const String selectedRouteKey = 'selected_route';
  static const String busIdKey = 'bus_id';
  
  // App Info
  static const String appName = 'JnU Bus Tracker - Driver';
  static const String appVersion = '1.0.0';
}
