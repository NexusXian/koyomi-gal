import '../core/api/api_client.dart';
import '../models/admin_models.dart';
import '../models/pagination.dart';

class AdminService {
  AdminService(this._api);

  final ApiClient _api;

  Future<Paginated<AdminPostData>> listPosts({
    String? keyword,
    int page = 1,
    int limit = 20,
  }) async {
    final data = await _api.get(
      '/api/v1/admin/posts',
      queryParameters: {
        if (keyword?.isNotEmpty == true) 'keyword': keyword,
        'page': page,
        'limit': limit,
      },
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: AdminPostData.fromMap,
    );
  }

  Future<Paginated<AdminCommentData>> listComments({
    String? keyword,
    int page = 1,
    int limit = 20,
  }) async {
    final data = await _api.get(
      '/api/v1/admin/comments',
      queryParameters: {
        if (keyword?.isNotEmpty == true) 'keyword': keyword,
        'page': page,
        'limit': limit,
      },
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: AdminCommentData.fromMap,
    );
  }

  Future<Paginated<AdminUserData>> listUsers({
    String? keyword,
    int page = 1,
    int limit = 20,
  }) async {
    final data = await _api.get(
      '/api/v1/admin/users',
      queryParameters: {
        if (keyword?.isNotEmpty == true) 'keyword': keyword,
        'page': page,
        'limit': limit,
      },
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: AdminUserData.fromMap,
    );
  }

  Future<Paginated<UserIPLog>> listUserIPLogs(
    int userId, {
    int page = 1,
    int limit = 20,
  }) async {
    final data = await _api.get(
      '/api/v1/admin/users/$userId/ip-logs',
      queryParameters: {'page': page, 'limit': limit},
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: UserIPLog.fromMap,
    );
  }
}
