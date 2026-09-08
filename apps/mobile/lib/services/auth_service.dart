import '../core/api/api_client.dart';
import '../core/api/api_exception.dart';
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

  Future<String> sendForgotPasswordCode(String email) async {
    await _api.post(
      '/api/v1/auth/password/forgot/code',
      data: {'email': email},
      flags: _skip,
    );
    return '如果该邮箱已注册，验证码将发送到邮箱';
  }

  Future<String> verifyForgotPasswordCode(String email, String code) async {
    final data = await _api.post(
      '/api/v1/auth/password/forgot/verify',
      data: {'email': email, 'code': code},
      flags: _skip,
    );
    final token = data is Map ? data['reset_token'] as String? : null;
    if (token == null || token.isEmpty) {
      throw ApiException(233, '验证码错误或已过期');
    }
    return token;
  }

  Future<void> resetForgottenPassword({
    required String resetToken,
    required String password,
    required String confirmPassword,
  }) {
    return _api.post(
      '/api/v1/auth/password/forgot/reset',
      data: {
        'reset_token': resetToken,
        'password': password,
        'confirm_password': confirmPassword,
      },
      flags: _skip,
    );
  }

  Future<String> sendChangePasswordCode() async {
    final data = await _api.post('/api/v1/users/me/password/code');
    return data is Map && data['email'] is String
        ? data['email'] as String
        : '';
  }

  Future<void> changePassword({
    required String code,
    required String newPassword,
    required String confirmPassword,
  }) {
    return _api.put(
      '/api/v1/users/me/password',
      data: {
        'code': code,
        'new_password': newPassword,
        'confirm_password': confirmPassword,
      },
    );
  }

  Future<void> logout() => _api.post('/api/v1/auth/logout', flags: _skip);
}
