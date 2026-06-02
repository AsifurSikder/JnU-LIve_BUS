import 'package:flutter/material.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';
import 'package:connectivity_plus/connectivity_plus.dart';
import '../services/api_service.dart';
import '../services/websocket_service.dart';
import '../models/bus_location.dart';
import '../config/app_config.dart';

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
  bool _isLoading = true;

  static const _defaultLocation = LatLng(
    AppConfig.defaultLatitude,
    AppConfig.defaultLongitude,
  );

  @override
  void initState() {
    super.initState();
    _initializeMap();
    _setupConnectivityListener();
  }

  Future<void> _initializeMap() async {
    setState(() {
      _isLoading = true;
    });

    // Fetch initial bus positions
    try {
      final buses = await _apiService.getAllBuses();
      setState(() {
        for (final bus in buses) {
          _addOrUpdateMarker(bus);
        }
        _isLoading = false;
      });
    } catch (e) {
      print('Error loading buses: $e');
      setState(() {
        _isLoading = false;
      });
    }

    // Connect to WebSocket
    _wsService.connect();
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
        snippet: 'Bus: ${bus.busId}${bus.driverName != null ? ' • Driver: ${bus.driverName}' : ''}',
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

    _wsService.connect();
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
            tooltip: 'Refresh',
          ),
        ],
      ),
      body: Stack(
        children: [
          _isLoading
              ? const Center(child: CircularProgressIndicator())
              : GoogleMap(
                  initialCameraPosition: CameraPosition(
                    target: _defaultLocation,
                    zoom: AppConfig.defaultZoom,
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
                      Icon(Icons.wifi_off, color: Colors.orange.shade700),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          'Offline — showing last known location',
                          style: TextStyle(
                            color: Colors.orange.shade900,
                            fontWeight: FontWeight.w500,
                          ),
                        ),
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
                child: Padding(
                  padding: const EdgeInsets.all(12.0),
                  child: Row(
                    children: [
                      SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          valueColor: AlwaysStoppedAnimation<Color>(
                            Colors.blue.shade700,
                          ),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Text(
                        'Reconnecting...',
                        style: TextStyle(
                          color: Colors.blue.shade900,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          if (_markers.isEmpty && !_isLoading)
            Positioned(
              top: 16,
              left: 16,
              right: 16,
              child: Card(
                color: Colors.grey.shade100,
                child: Padding(
                  padding: const EdgeInsets.all(16.0),
                  child: Column(
                    children: [
                      Icon(Icons.directions_bus_outlined,
                          size: 48, color: Colors.grey.shade600),
                      const SizedBox(height: 8),
                      Text(
                        'No buses are currently active',
                        style: TextStyle(
                          color: Colors.grey.shade700,
                          fontSize: 16,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () {
          _mapController?.animateCamera(
            CameraUpdate.newCameraPosition(
              CameraPosition(
                target: _defaultLocation,
                zoom: AppConfig.defaultZoom,
              ),
            ),
          );
        },
        child: const Icon(Icons.my_location),
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
