class BusRoute {
  final String id;
  final String name;
  final List<Stop> stops;
  final String? assignedBusId;
  final String? assignedDriverName;

  BusRoute({
    required this.id,
    required this.name,
    required this.stops,
    this.assignedBusId,
    this.assignedDriverName,
  });

  factory BusRoute.fromJson(Map<String, dynamic> json) {
    return BusRoute(
      id: json['id'],
      name: json['name'],
      stops: (json['stops'] as List?)
              ?.map((s) => Stop.fromJson(s))
              .toList() ??
          [],
      assignedBusId: json['assignedBusId'],
      assignedDriverName: json['assignedDriverName'],
    );
  }
}

class Stop {
  final String id;
  final String routeId;
  final String name;
  final double latitude;
  final double longitude;
  final int stopOrder;

  Stop({
    required this.id,
    required this.routeId,
    required this.name,
    required this.latitude,
    required this.longitude,
    required this.stopOrder,
  });

  factory Stop.fromJson(Map<String, dynamic> json) {
    return Stop(
      id: json['id'],
      routeId: json['routeId'] ?? json['route_id'],
      name: json['name'],
      latitude: (json['latitude'] as num).toDouble(),
      longitude: (json['longitude'] as num).toDouble(),
      stopOrder: json['stopOrder'] ?? json['stop_order'] ?? 0,
    );
  }
}
