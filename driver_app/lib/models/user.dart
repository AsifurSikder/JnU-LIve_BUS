class User {
  final String id;
  final String username;
  final String fullName;
  final String role;

  User({
    required this.id,
    required this.username,
    required this.fullName,
    required this.role,
  });

  factory User.fromJson(Map<String, dynamic> json) {
    return User(
      id: json['userId'] ?? json['id'],
      username: json['username'] ?? '',
      fullName: json['fullName'] ?? json['full_name'] ?? '',
      role: json['role'] ?? 'driver',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'username': username,
      'fullName': fullName,
      'role': role,
    };
  }
}

class LoginResponse {
  final String token;
  final String expiresAt;
  final String role;
  final String userId;

  LoginResponse({
    required this.token,
    required this.expiresAt,
    required this.role,
    required this.userId,
  });

  factory LoginResponse.fromJson(Map<String, dynamic> json) {
    return LoginResponse(
      token: json['token'],
      expiresAt: json['expiresAt'],
      role: json['role'],
      userId: json['userId'],
    );
  }
}
