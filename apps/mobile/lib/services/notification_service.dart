import '../core/api/api_client.dart';
import '../models/notification_models.dart';
import '../models/pagination.dart';

class NotificationService {
  NotificationService(this._api);

  final ApiClient _api;

  Future<Paginated<NotificationData>> list({
    String? category,
    bool? unread,
    int page = 1,
    int limit = 20,
  }) async {
    final data = await _api.get(
      '/api/v1/notifications',
      queryParameters: {
        'category': category,
        'unread': unread,
        'page': page,
        'limit': limit,
      },
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: NotificationData.fromMap,
    );
  }

  Future<int> unreadCount() async {
    final data = await _api.get('/api/v1/notifications/unread-count');
    return ((data as Map?)?['count'] as num?)?.toInt() ?? 0;
  }

  Future<void> markRead(int id) =>
      _api.patch('/api/v1/notifications/$id/read');

  Future<void> markAllRead() => _api.patch('/api/v1/notifications/read-all');
}
