import '../../../core/api/api_client.dart';
import '../models/app_release.dart';

class UpdateService {
  UpdateService(this._api);

  final ApiClient _api;

  Future<AppRelease> getLatestRelease({
    required int versionCode,
    required String versionName,
  }) async {
    final data = await _api.get(
      '/api/v1/app/releases/latest',
      queryParameters: {
        'platform': 'android',
        'versionCode': versionCode,
        'versionName': versionName,
      },
      flags: const ApiRequestFlags(skipAuth: true, skipRefresh: true),
    );
    return AppRelease.fromMap(Map<String, dynamic>.from(data as Map));
  }
}
