import '../core/api/api_client.dart';
import '../models/article_models.dart';
import '../models/pagination.dart';

class ArticleService {
  ArticleService(this._api);

  final ApiClient _api;

  static const _public = ApiRequestFlags(skipAuth: true, skipRefresh: true);

  Future<Paginated<ArticleListItem>> list({
    String? type,
    int page = 1,
    int limit = 20,
  }) async {
    final data = await _api.get(
      '/api/v1/articles',
      queryParameters: {'type': type, 'page': page, 'limit': limit},
      flags: _public,
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: ArticleListItem.fromMap,
    );
  }

  Future<ArticleDetail> get(int id) async {
    final data = await _api.get('/api/v1/articles/$id', flags: _public);
    return ArticleDetail.fromMap(Map<String, dynamic>.from(data));
  }
}
