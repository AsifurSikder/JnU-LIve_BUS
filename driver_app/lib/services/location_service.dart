import 'dart:async';
import 'package:geolocator/geolocator.dart';
import 'package:permission_handler/permission_handler.dart';
import '../config/app_config.dart';
import '../models/location_update.dart';
import 'api_service.dart';

class LocationService {
  final ApiService _apiService;
  StreamSubscription<Position>? _positionStream;
  Timer? _updateTimer;
  bool _isTracking = false;

  LocationService(this._apiService);

  bool get isTracking => _isTracking;

  Future<bool> checkPermissions() async {
    final status = await Permission.location.status;
    return status.isGranted;
  }

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
    if (_isTracking) return;

    // Check permissions
    final hasPermission = await Geolocator.checkPermission();
    if (hasPermission == LocationPermission.denied ||
        hasPermission == LocationPermission.deniedForever) {
      throw Exception('Location permission not granted');
    }

    _isTracking = true;

    // Start periodic updates
    _updateTimer = Timer.periodic(
      Duration(seconds: AppConfig.gpsUpdateIntervalSeconds),
      (_) async {
        try {
          final position = await Geolocator.getCurrentPosition(
            desiredAccuracy: LocationAccuracy.high,
          );

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
            print('Location sent: ${position.latitude}, ${position.longitude}');
          } catch (e) {
            print('Error sending location: $e');
            // TODO: Add to offline queue
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
    _isTracking = false;
  }

  void dispose() {
    stopTracking();
  }
}
