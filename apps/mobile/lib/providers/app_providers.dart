import 'dart:async';

import 'package:cookie_jar/cookie_jar.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../core/api/api_client.dart';
import '../core/config.dart';
import '../models/auth_models.dart';
import '../services/article_service.dart';
import '../services/auth_service.dart';
import '../services/galgame_service.dart';
import '../services/home_service.dart';
import '../services/notification_service.dart';
import '../services/novel_service.dart';
import '../services/post_service.dart';
import '../services/user_service.dart';

enum AuthStatus { loading, authenticated, unauthenticated }

class AuthController extends ChangeNotifier {
  AuthController(this._authService);

  final AuthService _authService;

  AuthStatus status = AuthStatus.loading;
  AuthUser? user;
  String? _accessToken;
  Future<String?>? _refreshFuture;

  String? get accessToken => _accessToken;
  bool get isAuthenticated => status == AuthStatus.authenticated;

  Future<void> initialize() async {
    try {
      final session = await _authService.refresh();
      _applySession(session);
    } catch (_) {
      status = AuthStatus.unauthenticated;
      _accessToken = null;
      user = null;
    }
    notifyListeners();
  }

  Future<void> login(String account, String password) async {
    final session = await _authService.login(account, password);
    _applySession(session);
    notifyListeners();
  }

  Future<String> register({
    required String username,
    required String email,
    required String password,
    required String confirmPassword,
    required String verificationCode,
  }) {
    return _authService.register(
      username: username,
      email: email,
      password: password,
      confirmPassword: confirmPassword,
      verificationCode: verificationCode,
    );
  }

  Future<String> sendVerificationCode(String email) {
    return _authService.sendVerificationCode(email);
  }

  Future<String> sendForgotPasswordCode(String email) {
    return _authService.sendForgotPasswordCode(email);
  }

  Future<String> verifyForgotPasswordCode(String email, String code) {
    return _authService.verifyForgotPasswordCode(email, code);
  }

  Future<void> resetForgottenPassword({
    required String resetToken,
    required String password,
    required String confirmPassword,
  }) {
    return _authService.resetForgottenPassword(
      resetToken: resetToken,
      password: password,
      confirmPassword: confirmPassword,
    );
  }

  Future<String> sendChangePasswordCode() {
    return _authService.sendChangePasswordCode();
  }

  Future<void> changePassword({
    required String code,
    required String newPassword,
    required String confirmPassword,
  }) {
    return _authService.changePassword(
      code: code,
      newPassword: newPassword,
      confirmPassword: confirmPassword,
    );
  }

  Future<void> logout() async {
    try {
      await _authService.logout();
    } catch (_) {}
    _clearSession();
    notifyListeners();
  }

  void _applySession(AuthSession session) {
    _accessToken = session.token;
    user = session.user;
    status = AuthStatus.authenticated;
  }

  void _clearSession() {
    _accessToken = null;
    user = null;
    status = AuthStatus.unauthenticated;
  }

  /// Single-flight refresh shared by all concurrent 401 retries.
  Future<String?> refreshSession() {
    return _refreshFuture ??= _doRefreshSession();
  }

  Future<String?> _doRefreshSession() async {
    try {
      final session = await _authService.refresh();
      _applySession(session);
      notifyListeners();
      return session.token;
    } catch (error) {
      _clearSession();
      notifyListeners();
      rethrow;
    } finally {
      _refreshFuture = null;
    }
  }

  void invalidateSession() {
    if (_refreshFuture != null) {
      return;
    }
    _clearSession();
    notifyListeners();
  }
}

/// Shared cookie jar keeps the httpOnly refresh cookie across restarts.
/// Overridden in main() with a PersistCookieJar bound to app storage
/// (null on web, where the browser manages cookies natively).
final cookieJarProvider = Provider<CookieJar?>((ref) => null);

final authControllerProvider = ChangeNotifierProvider<AuthController>((ref) {
  final jar = ref.watch(cookieJarProvider);
  final authApi = ApiClient(
    baseUrl: AppConfig.apiBase,
    getAccessToken: () => null,
    refreshSession: () async => null,
    onSessionInvalid: () {},
    cookieJar: jar,
  );
  final controller = AuthController(AuthService(authApi));
  controller.initialize();
  return controller;
});

final apiClientProvider = Provider<ApiClient>((ref) {
  final auth = ref.watch(authControllerProvider.notifier);
  return ApiClient(
    baseUrl: AppConfig.apiBase,
    getAccessToken: () => auth.accessToken,
    refreshSession: auth.refreshSession,
    onSessionInvalid: auth.invalidateSession,
    cookieJar: ref.watch(cookieJarProvider),
  );
});

final dioProvider = Provider<Dio>((ref) => ref.watch(apiClientProvider).dio);

final authServiceProvider = Provider<AuthService>((ref) =>
    AuthService(ref.watch(apiClientProvider)));

final galgameServiceProvider = Provider<GalgameService>(
    (ref) => GalgameService(ref.watch(apiClientProvider)));

final tagServiceProvider =
    Provider<TagService>((ref) => TagService(ref.watch(apiClientProvider)));

final developerServiceProvider = Provider<DeveloperService>(
    (ref) => DeveloperService(ref.watch(apiClientProvider)));

final resourceServiceProvider = Provider<ResourceService>(
    (ref) => ResourceService(ref.watch(apiClientProvider)));

final imageServiceProvider = Provider<ImageService>((ref) => ImageService(
      ref.watch(apiClientProvider),
      Dio(BaseOptions(
        connectTimeout: const Duration(seconds: 30),
        receiveTimeout: const Duration(seconds: 60),
      )),
    ));

final postServiceProvider =
    Provider<PostService>((ref) => PostService(ref.watch(apiClientProvider)));

final commentServiceProvider = Provider<CommentService>(
    (ref) => CommentService(ref.watch(apiClientProvider)));

final novelServiceProvider =
    Provider<NovelService>((ref) => NovelService(ref.watch(apiClientProvider)));

final articleServiceProvider = Provider<ArticleService>(
    (ref) => ArticleService(ref.watch(apiClientProvider)));

final userServiceProvider =
    Provider<UserService>((ref) => UserService(ref.watch(apiClientProvider)));

final meServiceProvider =
    Provider<MeService>((ref) => MeService(ref.watch(apiClientProvider)));

final mePermissionsProvider = FutureProvider<MePermissions>((ref) async {
  final auth = ref.watch(authControllerProvider);
  if (!auth.isAuthenticated) {
    return const MePermissions();
  }
  return ref.watch(meServiceProvider).permissions();
});

final notificationServiceProvider = Provider<NotificationService>(
    (ref) => NotificationService(ref.watch(apiClientProvider)));

final homeServiceProvider =
    Provider<HomeService>((ref) => HomeService(ref.watch(apiClientProvider)));

final feedbackServiceProvider = Provider<FeedbackService>(
    (ref) => FeedbackService(ref.watch(apiClientProvider)));

class UnreadNotifications extends ChangeNotifier {
  UnreadNotifications(this._ref) {
    _auth = _ref.read(authControllerProvider);
    _auth.addListener(_onAuthChanged);
    if (_auth.isAuthenticated) {
      _start();
    }
  }

  final Ref _ref;
  late final AuthController _auth;
  Timer? _timer;
  int count = 0;

  void _onAuthChanged() {
    if (_auth.isAuthenticated) {
      _start();
    } else {
      _stop();
      count = 0;
      notifyListeners();
    }
  }

  void _start() {
    refresh();
    _timer?.cancel();
    _timer = Timer.periodic(const Duration(seconds: 60), (_) => refresh());
  }

  void _stop() {
    _timer?.cancel();
    _timer = null;
  }

  Future<void> refresh() async {
    try {
      final service = _ref.read(notificationServiceProvider);
      count = await service.unreadCount();
      notifyListeners();
    } catch (_) {}
  }

  @override
  void dispose() {
    _stop();
    _auth.removeListener(_onAuthChanged);
    super.dispose();
  }
}

final unreadNotificationsProvider =
    ChangeNotifierProvider<UnreadNotifications>((ref) {
  ref.watch(authControllerProvider);
  return UnreadNotifications(ref);
});

final themeControllerProvider =
    ChangeNotifierProvider<ThemeController>((ref) => ThemeController());

class ThemeController extends ChangeNotifier {
  ThemeMode mode = ThemeMode.light;

  void toggle() {
    set(mode == ThemeMode.light ? ThemeMode.dark : ThemeMode.light);
  }

  void set(ThemeMode value) {
    mode = value;
    notifyListeners();
    _persist();
  }

  Future<void> _persist() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool('koyomi-dark-mode', mode == ThemeMode.dark);
  }
}

/// Called before runApp, so no listeners exist yet; no notify needed.
Future<void> loadThemePreference(ThemeController controller) async {
  final prefs = await SharedPreferences.getInstance();
  controller.mode =
      prefs.getBool('koyomi-dark-mode') == true ? ThemeMode.dark : ThemeMode.light;
}
