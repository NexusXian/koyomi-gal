import '../core/api/api_client.dart';
import '../models/misc_models.dart';

class HomeService {
  HomeService(this._api);

  final ApiClient _api;

  static const _public = ApiRequestFlags(skipAuth: true, skipRefresh: true);

  Future<HomeData> getHome() async {
    final data = await _api.get('/api/v1/home', flags: _public);
    return HomeData.fromMap(Map<String, dynamic>.from(data));
  }
}

class FeedbackService {
  FeedbackService(this._api);

  final ApiClient _api;

  Future<void> submit({
    required String type,
    required String content,
    String? contact,
  }) {
    return _api.post(
      '/api/v1/feedback',
      data: {
        'type': type,
        'content': content,
        if (contact != null && contact.isNotEmpty) 'contact': contact,
      },
      flags: const ApiRequestFlags(skipAuth: true, skipRefresh: true),
    );
  }
}
