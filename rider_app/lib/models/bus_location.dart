class BusLocation {
  final String busId;
  final double latitude;
  final double longitude;
  final String timestamp;
  final String? routeId;
  final String? routeName;
  final String? driverName;

  BusLocation({
    required this.busId,
    required this.latitude,
    required this.longitude,
    required this.timestamp,
    this.routeId,
    this.routeName,
    this.driverName,
  });

  factory BusLocation.fromJson(Map<String, dynamic> json) {
    return BusLocation(
      busId: json['busId'] ?? json['bus_id'] ?? '',
      latitude: (json['latitude'] as num).toDouble(),
      longitude: (json['longitude'] as num).toDouble(),
      timestamp: json['timestamp'] ?? DateTime.now().toIso8601String(),
      routeId: json['routeId'] ?? json['route_id'],
      routeName: json['routeName'] ?? json['route_name'],
      driverName: json['driverName'] ?? json['driver_name'],
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'busId': busId,
      'latitude': latitude,
      'longitude': longitude,
      'timestamp': timestamp,
      if (routeId != null) 'routeId': routeId,
      if (routeName != null) 'routeName': routeName,
      if (driverName != null) 'driverName': driverName,
    };
  }
}
