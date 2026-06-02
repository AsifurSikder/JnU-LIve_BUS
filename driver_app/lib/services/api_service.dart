import 'dart:convert';
import 'package:http/http.dart' as http;
import '../config/app_config.dart';
import '../models/user.dart';
import '../models/route.dart';
import '../models/location_update.dart';

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
