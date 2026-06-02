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

  Future<void> saveUsername(String username) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(AppConfig.usernameKey, username);
  }

  Future<String?> getUsername() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(AppConfig.usernameKey);
  }

  Future<void> saveBusId(String busId) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(AppConfig.busIdKey, busId);
  }

  Future<String?> getBusId() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(AppConfig.busIdKey);
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

  Future<void> clearAll() async {
    await _secureStorage.deleteAll();
    final prefs = await SharedPreferences.getInstance();
    await prefs.clear();
  }
}
