class LocationUpdate {
  final String busId;
  final String routeId;
  final String routeName;
  final String driverId;
  final String driverName;
  final double latitude;
  final double longitude;
  final String timestamp;
  final double? accuracy;
  final double? speed;

  LocationUpdate({
    required this.busId,
    required this.routeId,
    required this.routeName,
    required this.driverId,
    required this.driverName,
    required this.latitude,
    required this.longitude,
    required this.timestamp,
    this.accuracy,
    this.speed,
  });

  Map<String, dynamic> toJson() {
    final map = {
      'busId': busId,
      'routeId': routeId,
      'routeName': routeName,
      'driverId': driverId,
      'driverName': driverName,
      'latitude': latitude,
      'longitude': longitude,
      'timestamp': timestamp,
    };

    if (accuracy != null) map['accuracy'] = accuracy;
    if (speed != null) map['speed'] = speed;

    return map;
  }

  factory LocationUpdate.fromJson(Map<String, dynamic> json) {
    return LocationUpdate(
      busId: json['busId'],
      routeId: json['routeId'] ?? '',
      routeName: json['routeName'] ?? '',
      driverId: json['driverId'] ?? '',
      driverName: json['driverName'] ?? '',
      latitude: (json['latitude'] as num).toDouble(),
      longitude: (json['longitude'] as num).toDouble(),
      timestamp: json['timestamp'],
      accuracy: json['accuracy'] != null ? (json['accuracy'] as num).toDouble() : null,
      speed: json['speed'] != null ? (json['speed'] as num).toDouble() : null,
    );
  }
}
