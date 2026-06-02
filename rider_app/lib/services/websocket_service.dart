import 'dart:async';
import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';
import '../models/bus_location.dart';
import '../config/app_config.dart';

class WebSocketService {
  WebSocketChannel? _channel;
  final _busUpdatesController = StreamController<BusLocation>.broadcast();
  bool _isConnected = false;
  int _reconnectAttempts = 0;

  Stream<BusLocation> get busUpdates => _busUpdatesController.stream;
  bool get isConnected => _isConnected;

  void connect() {
    try {
      _channel = WebSocketChannel.connect(Uri.parse(AppConfig.wsUrl));
      _isConnected = true;
      _reconnectAttempts = 0;

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
          _attemptReconnect();
        },
        onDone: () {
          print('WebSocket closed');
          _isConnected = false;
          _attemptReconnect();
        },
      );
    } catch (e) {
      print('Connection error: $e');
      _isConnected = false;
      _attemptReconnect();
    }
  }

  void _attemptReconnect() {
    if (_reconnectAttempts < AppConfig.maxReconnectAttempts) {
      _reconnectAttempts++;
      print('Reconnecting... Attempt $_reconnectAttempts');
      
      Future.delayed(
        Duration(seconds: AppConfig.reconnectIntervalSeconds),
        () => connect(),
      );
    } else {
      print('Max reconnect attempts reached');
    }
  }

  void disconnect() {
    _channel?.sink.close();
    _isConnected = false;
    _reconnectAttempts = 0;
  }

  void dispose() {
    disconnect();
    _busUpdatesController.close();
  }
}
