import 'package:dio/dio.dart';
import 'package:dio_cookie_manager/dio_cookie_manager.dart';
import 'package:cookie_jar/cookie_jar.dart';

import 'api_exception.dart';

/// Mirrors the web client contract: skipAuth / skipRefresh request flags.
class ApiRequestFlags {
  const ApiRequestFlags({this.skipAuth = false, this.skipRefresh = false});

  final bool skipAuth;
  final bool skipRefresh;
}

typedef SessionRefresher = Future<String?> Function();
typedef SessionInvalidator = void Function();

class ApiClient {
  ApiClient({
    required this.baseUrl,
    required this.getAccessToken,
    required this.refreshSession,
    required this.onSessionInvalid,
    CookieJar? cookieJar,
  }) : cookieJar = cookieJar {
    dio = Dio(
      BaseOptions(
        baseUrl: baseUrl,
        connectTimeout: const Duration(seconds: 15),
        receiveTimeout: const Duration(seconds: 20),
        validateStatus: (status) =>
            status != null && status >= 200 && status < 300,
        // Web: let the browser attach/send the httpOnly refresh cookie.
        extra: {'withCredentials': true},
      ),
    );
    final jar = cookieJar;
    if (jar != null) {
      dio.interceptors.add(CookieManager(jar));
    }
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          final flags = _flagsOf(options);
          final token = getAccessToken();
          if (!flags.skipAuth && token != null && token.isNotEmpty) {
            options.headers['Authorization'] = 'Bearer $token';
          }
          handler.next(options);
        },
        onError: (error, handler) async {
          final response = error.response;
          final isUnauthorized = response?.statusCode == 401;
          final flags = _flagsOf(error.requestOptions);
          final alreadyRetried =
              error.requestOptions.extra['__retried'] == true;

          if (!isUnauthorized || flags.skipRefresh || alreadyRetried) {
            return handler.next(error);
          }

          String? newToken;
          try {
            newToken = await refreshSession();
          } catch (_) {
            onSessionInvalid();
            return handler.next(error);
          }

            final options = error.requestOptions;
            options.extra['__retried'] = true;
            if (newToken != null && newToken.isNotEmpty) {
              options.headers['Authorization'] = 'Bearer $newToken';
            }
          try {
            final retryResponse = await dio.fetch(options);
            return handler.resolve(retryResponse);
          } on DioException catch (retryError) {
            return handler.next(retryError);
          }
        },
      ),
    );
  }

  final String baseUrl;
  final String? Function() getAccessToken;
  final SessionRefresher refreshSession;
  final SessionInvalidator onSessionInvalid;
  final CookieJar? cookieJar;

  late final Dio dio;

  static ApiRequestFlags _flagsOf(RequestOptions options) {
    return ApiRequestFlags(
      skipAuth: options.extra['skipAuth'] == true,
      skipRefresh: options.extra['skipRefresh'] == true,
    );
  }

  Options _options({
    ApiRequestFlags? flags,
    String? contentType,
    ResponseType? responseType,
  }) {
    return Options(
      contentType: contentType,
      responseType: responseType,
      extra: {
        if (flags?.skipAuth ?? false) 'skipAuth': true,
        if (flags?.skipRefresh ?? false) 'skipRefresh': true,
      },
    );
  }

  dynamic _unwrap(Response response) {
    final body = response.data;
    if (body is! Map) {
      return null;
    }
    final code = body['code'];
    if (code is int && code != 0) {
      throw ApiException(code, (body['msg'] as String?) ?? '请求失败');
    }
    if (response.statusCode != null && response.statusCode! >= 400) {
      throw ApiException(
        response.statusCode,
        (body['msg'] as String?) ?? '请求失败 (${response.statusCode})',
      );
    }
    return body['data'];
  }

  Future<dynamic> get(
    String path, {
    Map<String, dynamic>? queryParameters,
    ApiRequestFlags? flags,
  }) async {
    final response = await dio.get(
      path,
      queryParameters: _cleanQuery(queryParameters),
      options: _options(flags: flags),
    );
    return _unwrap(response);
  }

  Future<dynamic> post(
    String path, {
    Object? data,
    ApiRequestFlags? flags,
  }) async {
    final response = await dio.post(
      path,
      data: data,
      options: _options(flags: flags, contentType: Headers.jsonContentType),
    );
    return _unwrap(response);
  }

  Future<dynamic> put(
    String path, {
    Object? data,
    ApiRequestFlags? flags,
  }) async {
    final response = await dio.put(
      path,
      data: data,
      options: _options(flags: flags, contentType: Headers.jsonContentType),
    );
    return _unwrap(response);
  }

  Future<dynamic> patch(
    String path, {
    Object? data,
    ApiRequestFlags? flags,
  }) async {
    final response = await dio.patch(
      path,
      data: data,
      options: _options(flags: flags, contentType: Headers.jsonContentType),
    );
    return _unwrap(response);
  }

  Future<dynamic> delete(
    String path, {
    Object? data,
    ApiRequestFlags? flags,
  }) async {
    final response = await dio.delete(
      path,
      data: data,
      options: _options(flags: flags),
    );
    return _unwrap(response);
  }

  static Map<String, dynamic>? _cleanQuery(Map<String, dynamic>? params) {
    if (params == null) {
      return null;
    }
    return params..removeWhere((key, value) => value == null);
  }
}

String apiErrorMessage(Object error) {
  if (error is ApiException) {
    return error.message;
  }
  if (error is DioException) {
    final body = error.response?.data;
    if (body is Map && body['msg'] is String) {
      return body['msg'] as String;
    }
    switch (error.type) {
      case DioExceptionType.connectionTimeout:
      case DioExceptionType.sendTimeout:
      case DioExceptionType.receiveTimeout:
        return '请求超时，请检查网络';
      case DioExceptionType.connectionError:
        return '无法连接服务器，请检查网络';
      default:
        break;
    }
    return '网络请求失败';
  }
  return error.toString();
}
