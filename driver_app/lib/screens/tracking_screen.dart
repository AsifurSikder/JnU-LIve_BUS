import 'package:flutter/material.dart';
import 'package:geolocator/geolocator.dart';
import '../models/route.dart';
import '../services/api_service.dart';
import '../services/storage_service.dart';
import '../services/location_service.dart';

class TrackingScreen extends StatefulWidget {
  final ApiService apiService;
  final StorageService storageService;
  final BusRoute route;

  const TrackingScreen({
    Key? key,
    required this.apiService,
    required this.storageService,
    required this.route,
  }) : super(key: key);

  @override
  State<TrackingScreen> createState() => _TrackingScreenState();
}

class _TrackingScreenState extends State<TrackingScreen> {
  late LocationService _locationService;
  bool _isTracking = false;
  Position? _currentPosition;
  String? _errorMessage;
  int _updateCount = 0;

  @override
  void initState() {
    super.initState();
    _locationService = LocationService(widget.apiService);
    _checkPermissionsAndStart();
  }

  Future<void> _checkPermissionsAndStart() async {
    final hasPermission = await _locationService.checkPermissions();
    if (!hasPermission) {
      final granted = await _locationService.requestPermissions();
      if (!granted && mounted) {
        setState(() {
          _errorMessage = 'Location permission is required to track bus';
        });
        return;
      }
    }
  }

  Future<void> _startTracking() async {
    setState(() {
      _errorMessage = null;
    });

    try {
      final userId = await widget.storageService.getUserId();
      final username = await widget.storageService.getUsername();
      final busId = await widget.storageService.getBusId() ?? 'bus-temp-001';

      await _locationService.startTracking(
        busId: busId,
        routeId: widget.route.id,
        routeName: widget.route.name,
        driverId: userId ?? 'driver-unknown',
        driverName: username ?? 'Driver',
      );

      setState(() {
        _isTracking = true;
      });

      // Update current position periodically
      _updatePosition();
    } catch (e) {
      setState(() {
        _errorMessage = e.toString();
      });
    }
  }

  Future<void> _updatePosition() async {
    while (_isTracking) {
      try {
        final position = await Geolocator.getCurrentPosition();
        if (mounted) {
          setState(() {
            _currentPosition = position;
            _updateCount++;
          });
        }
        await Future.delayed(const Duration(seconds: 5));
      } catch (e) {
        print('Error getting position: $e');
      }
    }
  }

  Future<void> _stopTracking() async {
    await _locationService.stopTracking();
    setState(() {
      _isTracking = false;
    });
  }

  Future<void> _endSession() async {
    if (_isTracking) {
      await _stopTracking();
    }

    if (mounted) {
      Navigator.of(context).pushReplacementNamed('/route-selection');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('GPS Tracking'),
        actions: [
          IconButton(
            icon: const Icon(Icons.stop_circle),
            onPressed: _endSession,
            tooltip: 'End Session',
          ),
        ],
      ),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(16.0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(16.0),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text(
                        'Route Information',
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      const SizedBox(height: 12),
                      Row(
                        children: [
                          const Icon(Icons.route, color: Colors.blue),
                          const SizedBox(width: 12),
                          Expanded(
                            child: Text(
                              widget.route.name,
                              style: const TextStyle(fontSize: 16),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 8),
                      Row(
                        children: [
                          const Icon(Icons.location_on, color: Colors.green),
                          const SizedBox(width: 12),
                          Text('${widget.route.stops.length} stops'),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 16),
              Card(
                color: _isTracking ? Colors.green.shade50 : Colors.grey.shade50,
                child: Padding(
                  padding: const EdgeInsets.all(16.0),
                  child: Column(
                    children: [
                      Icon(
                        _isTracking ? Icons.gps_fixed : Icons.gps_off,
                        size: 48,
                        color: _isTracking ? Colors.green : Colors.grey,
                      ),
                      const SizedBox(height: 12),
                      Text(
                        _isTracking ? 'Tracking Active' : 'Tracking Inactive',
                        style: TextStyle(
                          fontSize: 20,
                          fontWeight: FontWeight.bold,
                          color: _isTracking ? Colors.green : Colors.grey,
                        ),
                      ),
                      if (_isTracking) ...[
                        const SizedBox(height: 8),
                        Text(
                          'Updates sent: $_updateCount',
                          style: const TextStyle(color: Colors.grey),
                        ),
                      ],
                    ],
                  ),
                ),
              ),
              if (_currentPosition != null) ...[
                const SizedBox(height: 16),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16.0),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text(
                          'Current Location',
                          style: TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                        const SizedBox(height: 12),
                        Row(
                          children: [
                            const Icon(Icons.location_pin, size: 20),
                            const SizedBox(width: 8),
                            Expanded(
                              child: Text(
                                'Lat: ${_currentPosition!.latitude.toStringAsFixed(6)}',
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 4),
                        Row(
                          children: [
                            const Icon(Icons.location_pin, size: 20),
                            const SizedBox(width: 8),
                            Expanded(
                              child: Text(
                                'Lon: ${_currentPosition!.longitude.toStringAsFixed(6)}',
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 4),
                        Row(
                          children: [
                            const Icon(Icons.speed, size: 20),
                            const SizedBox(width: 8),
                            Text(
                              'Speed: ${_currentPosition!.speed.toStringAsFixed(1)} m/s',
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                ),
              ],
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
                  ),
                ),
              ],
              const Spacer(),
              ElevatedButton(
                onPressed: _isTracking ? _stopTracking : _startTracking,
                style: ElevatedButton.styleFrom(
                  padding: const EdgeInsets.symmetric(vertical: 16),
                  backgroundColor: _isTracking ? Colors.red : Colors.green,
                  foregroundColor: Colors.white,
                ),
                child: Text(
                  _isTracking ? 'Stop Tracking' : 'Start Tracking',
                  style: const TextStyle(fontSize: 18),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  @override
  void dispose() {
    _locationService.dispose();
    super.dispose();
  }
}
