# Mobile Apps Implementation Guide

This guide provides complete implementation instructions for the Flutter Driver App and Rider App.

---

## 🚗 Driver App Implementation

### Step 1: Install Dependencies

```bash
cd driver_app
flutter pub get
```

### Step 2: Update Android Permissions

Edit `android/app/src/main/AndroidManifest.xml`:

```xml
<manifest xmlns:android="http://schemas.android.com/apk/res/android">
    <!-- Permissions -->
    <uses-permission android:name="android.permission.INTERNET"/>
    <uses-permission android:name="android.permission.ACCESS_FINE_LOCATION"/>
    <uses-permission android:name="android.permission.ACCESS_COARSE_LOCATION"/>
    <uses-permission android:name="android.permission.ACCESS_BACKGROUND_LOCATION"/>
    <uses-permission android:name="android.permission.FOREGROUND_SERVICE"/>
    <uses-permission android:name="android.permission.WAKE_LOCK"/>
    
    <application
        android:label="JnU Bus Tracker Driver"
        android:name="${applicationName}"
        android:icon="@mipmap/ic_launcher">
        <!-- Rest of the config -->
    </application>
</manifest>
```

### Step 3: Create API Service

Create `lib/services/api_service.dart`:

```dart
import 'dart:convert';
import 'package:http/http.dart' as http;
import '../config/app_config.dart';
import '../models/user.dart';
import '../models/route.dart';

class ApiService {
  String? _jwtToken;
  
  void setToken(String token) {
    _jwtToken = token;
  }
  
  Map<String, String> _getHeaders() {
    final headers = {
      'Content-Type': 'application/json',
    };
    if (_jwtToken != null) {
      headers['Authorization'] = 'Bearer $_jwtToken';
    }
    return headers;
  }
  
  Future<LoginResponse> login(String username, String password) async {
    final url = Uri.parse('${AppConfig.apiBaseUrl}${AppConfig.loginEndpoint}');
    final response = await http.post(
      url,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'username': username,
        'password': password,
        'role': 'driver',
      }),
    );
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return LoginResponse.fromJson(data);
    } else if (response.statusCode == 401) {
      throw Exception('Invalid credentials');
    } else if (response.statusCode == 429) {
      throw Exception('Too many attempts. Please try again later');
    } else {
      throw Exception('Login failed: ${response.statusCode}');
    }
  }
  
  Future<List<BusRoute>> getRoutes() async {
    final url = Uri.parse('${AppConfig.apiBaseUrl}${AppConfig.routesEndpoint}');
    final response = await http.get(url, headers: _getHeaders());
    
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      final routes = (data['routes'] as List)
          .map((r) => BusRoute.fromJson(r))
          .toList();
      return routes;
    } else {
      throw Exception('Failed to load routes');
    }
  }
  
  Future<void> sendLocationUpdate(LocationUpdate update) async {
    final url = Uri.parse('${AppConfig.apiBaseUrl}${AppConfig.locationUpdateEndpoint}');
    final response = await http.post(
      url,
      headers: _getHeaders(),
      body: jsonEncode(update.toJson()),
    );
    
    if (response.statusCode != 200) {
      throw Exception('Failed to send location: ${response.statusCode}');
    }
  }
}
```

### Step 4: Create Storage Service

Create `lib/services/storage_service.dart`:

```dart
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../config/app_config.dart';

class StorageService {
  final _secureStorage = const FlutterSecureStorage();
  
  Future<void> saveToken(String token) async {
    await _secureStorage.write(key: AppConfig.jwtTokenKey, value: token);
  }
  
  Future<String?> getToken() async {
    return await _secureStorage.read(key: AppConfig.jwtTokenKey);
  }
  
  Future<void> clearToken() async {
    await _secureStorage.delete(key: AppConfig.jwtTokenKey);
  }
  
  Future<void> saveUserId(String userId) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(AppConfig.userIdKey, userId);
  }
  
  Future<String?> getUserId() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(AppConfig.userIdKey);
  }
  
  Future<void> saveSelectedRoute(String routeId, String routeName) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('selected_route_id', routeId);
    await prefs.setString('selected_route_name', routeName);
  }
  
  Future<Map<String, String>?> getSelectedRoute() async {
    final prefs = await SharedPreferences.getInstance();
    final routeId = prefs.getString('selected_route_id');
    final routeName = prefs.getString('selected_route_name');
    if (routeId != null && routeName != null) {
      return {'id': routeId, 'name': routeName};
    }
    return null;
  }
}
```

### Step 5: Create Location Service

Create `lib/services/location_service.dart`:

```dart
import 'dart:async';
import 'package:geolocator/geolocator.dart';
import 'package:permission_handler/permission_handler.dart';
import '../models/location_update.dart';
import 'api_service.dart';
import 'offline_queue_service.dart';

class LocationService {
  final ApiService _apiService;
  final OfflineQueueService _queueService;
  StreamSubscription<Position>? _positionStream;
  Timer? _updateTimer;
  
  LocationService(this._apiService, this._queueService);
  
  Future<bool> requestPermissions() async {
    final status = await Permission.location.request();
    if (status.isGranted) {
      // Request background permission for Android 10+
      await Permission.locationAlways.request();
      return true;
    }
    return false;
  }
  
  Future<void> startTracking({
    required String busId,
    required String routeId,
    required String routeName,
    required String driverId,
    required String driverName,
  }) async {
    // Check permissions
    final hasPermission = await Geolocator.checkPermission();
    if (hasPermission == LocationPermission.denied ||
        hasPermission == LocationPermission.deniedForever) {
      throw Exception('Location permission not granted');
    }
    
    // Start listening to position changes
    _positionStream = Geolocator.getPositionStream(
      locationSettings: const LocationSettings(
        accuracy: LocationAccuracy.high,
        distanceFilter: 10, // meters
      ),
    ).listen((Position position) async {
      final update = LocationUpdate(
        busId: busId,
        routeId: routeId,
        routeName: routeName,
        driverId: driverId,
        driverName: driverName,
        latitude: position.latitude,
        longitude: position.longitude,
        timestamp: DateTime.now().toUtc().toIso8601String(),
        accuracy: position.accuracy,
        speed: position.speed,
      );
      
      try {
        await _apiService.sendLocationUpdate(update);
      } catch (e) {
        // If failed, add to offline queue
        await _queueService.enqueue(update);
      }
    });
    
    // Also send updates on timer (every 5 seconds)
    _updateTimer = Timer.periodic(
      const Duration(seconds: AppConfig.gpsUpdateIntervalSeconds),
      (_) async {
        try {
          final position = await Geolocator.getCurrentPosition();
          final update = LocationUpdate(
            busId: busId,
            routeId: routeId,
            routeName: routeName,
            driverId: driverId,
            driverName: driverName,
            latitude: position.latitude,
            longitude: position.longitude,
            timestamp: DateTime.now().toUtc().toIso8601String(),
            accuracy: position.accuracy,
            speed: position.speed,
          );
          
          try {
            await _apiService.sendLocationUpdate(update);
            // Try to send queued updates
            await _queueService.processQueue(_apiService);
          } catch (e) {
            await _queueService.enqueue(update);
          }
        } catch (e) {
          print('Error getting position: $e');
        }
      },
    );
  }
  
  Future<void> stopTracking() async {
    await _positionStream?.cancel();
    _updateTimer?.cancel();
    _positionStream = null;
    _updateTimer = null;
  }
  
  void dispose() {
    stopTracking();
  }
}
```

### Step 6: Create Login Screen

Create `lib/screens/login_screen.dart`:

```dart
import 'package:flutter/material.dart';
import '../services/api_service.dart';
import '../services/storage_service.dart';
import 'route_selection_screen.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({Key? key}) : super(key: key);
  
  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _formKey = GlobalKey<FormState>();
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  final _apiService = ApiService();
  final _storageService = StorageService();
  bool _isLoading = false;
  String? _errorMessage;
  
  Future<void> _login() async {
    if (!_formKey.currentState!.validate()) return;
    
    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });
    
    try {
      final response = await _apiService.login(
        _usernameController.text,
        _passwordController.text,
      );
      
      // Save token and user info
      await _storageService.saveToken(response.token);
      await _storageService.saveUserId(response.userId);
      _apiService.setToken(response.token);
      
      // Navigate to route selection
      if (mounted) {
        Navigator.of(context).pushReplacement(
          MaterialPageRoute(
            builder: (_) => RouteSelectionScreen(apiService: _apiService),
          ),
        );
      }
    } catch (e) {
      setState(() {
        _errorMessage = e.toString().replaceFirst('Exception: ', '');
      });
    } finally {
      setState(() {
        _isLoading = false;
      });
    }
  }
  
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24.0),
          child: Form(
            key: _formKey,
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const Icon(
                  Icons.directions_bus,
                  size: 80,
                  color: Colors.blue,
                ),
                const SizedBox(height: 24),
                const Text(
                  'JnU Bus Tracker',
                  style: TextStyle(fontSize: 28, fontWeight: FontWeight.bold),
                  textAlign: TextAlign.center,
                ),
                const Text(
                  'Driver App',
                  style: TextStyle(fontSize: 18, color: Colors.grey),
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 48),
                TextFormField(
                  controller: _usernameController,
                  decoration: const InputDecoration(
                    labelText: 'Username',
                    border: OutlineInputBorder(),
                    prefixIcon: Icon(Icons.person),
                  ),
                  validator: (value) {
                    if (value == null || value.isEmpty) {
                      return 'Please enter username';
                    }
                    return null;
                  },
                ),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _passwordController,
                  decoration: const InputDecoration(
                    labelText: 'Password',
                    border: OutlineInputBorder(),
                    prefixIcon: Icon(Icons.lock),
                  ),
                  obscureText: true,
                  validator: (value) {
                    if (value == null || value.isEmpty) {
                      return 'Please enter password';
                    }
                    return null;
                  },
                ),
                if (_errorMessage != null) ...[
                  const SizedBox(height: 16),
                  Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      color: Colors.red.shade50,
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Text(
                      _errorMessage!,
                      style: const TextStyle(color: Colors.red),
                      textAlign: TextAlign.center,
                    ),
                  ),
                ],
                const SizedBox(height: 24),
                ElevatedButton(
                  onPressed: _isLoading ? null : _login,
                  style: ElevatedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                  ),
                  child: _isLoading
                      ? const CircularProgressIndicator(color: Colors.white)
                      : const Text('Login', style: TextStyle(fontSize: 18)),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
  
  @override
  void dispose() {
    _usernameController.dispose();
    _passwordController.dispose();
    super.dispose();
  }
}
```

---

## 📱 Rider App Implementation

### Step 1: Install Dependencies

Edit `rider_app/pubspec.yaml`:

```yaml
dependencies:
  flutter:
    sdk: flutter
  cupertino_icons: ^1.0.8
  http: ^1.2.0
  web_socket_channel: ^2.4.5
  google_maps_flutter: ^2.6.1
  connectivity_plus: ^6.0.3
  provider: ^6.1.2
```

### Step 2: Add Google Maps API Key

Edit `android/app/src/main/AndroidManifest.xml`:

```xml
<application>
    <meta-data
        android:name="com.google.android.geo.API_KEY"
        android:value="YOUR_GOOGLE_MAPS_API_KEY_HERE"/>
</application>
```

### Step 3: Create WebSocket Service

Create `rider_app/lib/services/websocket_service.dart`:

```dart
import 'dart:async';
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';
import '../models/bus_location.dart';

class WebSocketService {
  WebSocketChannel? _channel;
  final _busUpdatesController = StreamController<BusLocation>.broadcast();
  bool _isConnected = false;
  
  Stream<BusLocation> get busUpdates => _busUpdatesController.stream;
  bool get isConnected => _isConnected;
  
  void connect(String wsUrl) {
    try {
      _channel = WebSocketChannel.connect(Uri.parse(wsUrl));
      _isConnected = true;
      
      _channel!.stream.listen(
        (message) {
          try {
            final data = jsonDecode(message);
            if (data['type'] == 'location_update') {
              final busLocation = BusLocation.fromJson(data);
              _busUpdatesController.add(busLocation);
            }
          } catch (e) {
            print('Error parsing message: $e');
          }
        },
        onError: (error) {
          print('WebSocket error: $error');
          _isConnected = false;
        },
        onDone: () {
          print('WebSocket closed');
          _isConnected = false;
        },
      );
    } catch (e) {
      print('Connection error: $e');
      _isConnected = false;
    }
  }
  
  void disconnect() {
    _channel?.sink.close();
    _isConnected = false;
  }
  
  void dispose() {
    disconnect();
    _busUpdatesController.close();
  }
}
```

### Step 4: Create Map Screen

Create `rider_app/lib/screens/map_screen.dart`:

```dart
import 'package:flutter/material.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';
import 'package:connectivity_plus/connectivity_plus.dart';
import '../services/api_service.dart';
import '../services/websocket_service.dart';
import '../models/bus_location.dart';

class MapScreen extends StatefulWidget {
  const MapScreen({Key? key}) : super(key: key);
  
  @override
  State<MapScreen> createState() => _MapScreenState();
}

class _MapScreenState extends State<MapScreen> {
  GoogleMapController? _mapController;
  final _apiService = ApiService();
  final _wsService = WebSocketService();
  final Map<String, Marker> _markers = {};
  bool _isOnline = true;
  bool _isReconnecting = false;
  
  static const _dhaka = LatLng(23.7104, 90.4074); // JnU coordinates
  
  @override
  void initState() {
    super.initState();
    _initializeMap();
    _setupConnectivityListener();
  }
  
  Future<void> _initializeMap() async {
    // Fetch initial bus positions
    try {
      final buses = await _apiService.getAllBuses();
      setState(() {
        for (final bus in buses) {
          _addOrUpdateMarker(bus);
        }
      });
    } catch (e) {
      print('Error loading buses: $e');
    }
    
    // Connect to WebSocket
    _wsService.connect('ws://localhost:8080/ws/location');
    _wsService.busUpdates.listen((busLocation) {
      setState(() {
        _addOrUpdateMarker(busLocation);
      });
    });
  }
  
  void _addOrUpdateMarker(BusLocation bus) {
    final marker = Marker(
      markerId: MarkerId(bus.busId),
      position: LatLng(bus.latitude, bus.longitude),
      infoWindow: InfoWindow(
        title: bus.routeName ?? 'No Route',
        snippet: 'Bus: ${bus.busId}',
      ),
      icon: BitmapDescriptor.defaultMarkerWithHue(BitmapDescriptor.hueBlue),
    );
    _markers[bus.busId] = marker;
  }
  
  void _setupConnectivityListener() {
    Connectivity().onConnectivityChanged.listen((result) {
      final isOnline = result != ConnectivityResult.none;
      setState(() {
        _isOnline = isOnline;
      });
      
      if (isOnline && !_wsService.isConnected) {
        _reconnect();
      }
    });
  }
  
  Future<void> _reconnect() async {
    setState(() {
      _isReconnecting = true;
    });
    
    _wsService.connect('ws://localhost:8080/ws/location');
    await _initializeMap();
    
    setState(() {
      _isReconnecting = false;
    });
  }
  
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('JnU Live Bus Tracker'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: _initializeMap,
          ),
        ],
      ),
      body: Stack(
        children: [
          GoogleMap(
            initialCameraPosition: const CameraPosition(
              target: _dhaka,
              zoom: 14,
            ),
            markers: Set<Marker>.of(_markers.values),
            myLocationEnabled: true,
            myLocationButtonEnabled: true,
            onMapCreated: (controller) {
              _mapController = controller;
            },
          ),
          if (!_isOnline)
            Positioned(
              top: 16,
              left: 16,
              right: 16,
              child: Card(
                color: Colors.orange.shade100,
                child: Padding(
                  padding: const EdgeInsets.all(12.0),
                  child: Row(
                    children: [
                      const Icon(Icons.wifi_off, color: Colors.orange),
                      const SizedBox(width: 8),
                      Text(
                        'Offline — showing last known location',
                        style: TextStyle(color: Colors.orange.shade900),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          if (_isReconnecting)
            Positioned(
              top: 16,
              left: 16,
              right: 16,
              child: Card(
                color: Colors.blue.shade100,
                child: const Padding(
                  padding: EdgeInsets.all(12.0),
                  child: Row(
                    children: [
                      SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      ),
                      SizedBox(width: 12),
                      Text('Reconnecting...'),
                    ],
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
  
  @override
  void dispose() {
    _wsService.dispose();
    _mapController?.dispose();
    super.dispose();
  }
}
```

---

## 🌐 Admin Panel Implementation

See ADMIN_PANEL_IMPLEMENTATION_GUIDE.md for React admin panel implementation.

---

## 🚀 Running the Apps

### Driver App
```bash
cd driver_app
flutter run
```

### Rider App
```bash
cd rider_app
flutter run
```

**Note**: Make sure backend services are running on `localhost:8080` before testing the apps!

