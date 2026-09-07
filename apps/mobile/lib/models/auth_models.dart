class AuthUser {
  const AuthUser({
    this.id,
    this.username,
    this.email,
    this.avatar,
  });

  factory AuthUser.fromMap(Map<String, dynamic> map) => AuthUser(
        id: (map['id'] as num?)?.toInt(),
        username: map['username'] as String?,
        email: map['email'] as String?,
        avatar: map['avatar'] as String?,
      );

  final int? id;
  final String? username;
  final String? email;
  final String? avatar;
}

class AuthSession {
  const AuthSession({required this.user, required this.token});

  factory AuthSession.fromMap(Map<String, dynamic> map) => AuthSession(
        user: map['user'] is Map
            ? AuthUser.fromMap(Map<String, dynamic>.from(map['user']))
            : const AuthUser(),
        token: (map['token'] as String?) ?? '',
      );

  final AuthUser user;
  final String token;
}

class MePermissions {
  const MePermissions({this.roles = const [], this.permissions = const []});

  factory MePermissions.fromMap(Map<String, dynamic> map) => MePermissions(
        roles:
            (map['roles'] as List?)?.map((e) => e.toString()).toList() ?? [],
        permissions: (map['permissions'] as List?)
                ?.map((e) => e.toString())
                .toList() ??
            [],
      );

  final List<String> roles;
  final List<String> permissions;

  bool get isSuperAdmin => roles.contains('super_admin');

  bool has(String code) => isSuperAdmin || permissions.contains(code);

  bool hasAny(List<String> codes) =>
      isSuperAdmin || codes.any(permissions.contains);
}
