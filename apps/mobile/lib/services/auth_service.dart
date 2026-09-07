import '../core/api/api_client.dart';
import '../models/auth_models.dart';

class AuthService {
  AuthService(this._api);

  final ApiClient _api;

  static const _skip = ApiRequestFlags(skipAuth: true, skipRefresh: true);

  Future<AuthSession> login(String account, String password) async {
    final data = await _api.post(
      '/api/v1/auth/login',
      data: {'account': account, 'password': password},
      flags: _skip,
    );
    return AuthSession.fromMap(Map<String, dynamic>.from(data));
  }

  Future<AuthSession> refresh() async {
    final data = await _api.post('/api/v1/auth/refresh', flags: _skip);
    return AuthSession.fromMap(Map<String, dynamic>.from(data));
  }

  Future<String> register({
    required String username,
    required String email,
    required String password,
    required String confirmPassword,
    required String verificationCode,
  }) async {
    final body = await _api.post(
      '/api/v1/auth/register',
      data: {
        'username': username,
        'email': email,
        'password': password,
        'confirm_password': confirmPassword,
        'verification_code': verificationCode,
      },
      flags: _skip,
    );
    return body is Map && body['msg'] is String
        ? body['msg'] as String
        : '注册成功';
  }

  Future<String> sendVerificationCode(String email) async {
    final body = await _api.post(
      '/api/v1/auth/verification-codes',
      data: {'email': email, 'purpose': 'register'},
      flags: _skip,
    );
    return body is Map && body['msg'] is String
        ? body['msg'] as String
        : '验证码已发送';
  }

  Future<void> logout() => _api.post('/api/v1/auth/logout', flags: _skip);
}
