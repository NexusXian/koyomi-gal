import '../../../core/api/api_client.dart';
import '../models/announcement.dart';

class AnnouncementService {
  AnnouncementService(this._api);

  final ApiClient _api;

  Future<List<Announcement>> getActive() async {
    final data = await _api.get(
      '/api/v1/announcements/active',
      queryParameters: const {'platform': 'android'},
      flags: const ApiRequestFlags(skipAuth: true, skipRefresh: true),
    );
    return (data as List? ?? const [])
        .whereType<Map>()
        .map((item) => Announcement.fromMap(Map<String, dynamic>.from(item)))
        .toList();
  }
}
