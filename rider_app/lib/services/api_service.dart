import 'dart:convert';
import 'package:http/http.dart' as http;
import '../config/app_config.dart';
import '../models/bus_location.dart';

class ApiService {
  Future<List<BusLocation>> getAllBuses() async {
    final url = Uri.parse('${AppConfig.apiBaseUrl}${AppConfig.locationBusesEndpoint}');
    
    try {
      final response = await http.get(url);

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        final buses = (data['buses'] as List?)
                ?.map((b) => BusLocation.fromJson(b))
                .toList() ??
            [];
        return buses;
      } else {
        throw Exception('Failed to load buses: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Error fetching buses: $e');
    }
  }
}
