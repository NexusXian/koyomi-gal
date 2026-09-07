import '../core/api/api_client.dart';
import '../models/pagination.dart';
import '../models/post_models.dart';

class PostService {
  PostService(this._api);

  final ApiClient _api;

  Future<Paginated<PostData>> list({
    int? galgameId,
    int page = 1,
    int limit = 20,
  }) async {
    final data = await _api.get(
      '/api/v1/posts',
      queryParameters: {
        'galgame_id': galgameId,
        'page': page,
        'limit': limit,
      },
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: PostData.fromMap,
    );
  }

  Future<PostData> get(int id) async {
    final data = await _api.get('/api/v1/posts/$id');
    return PostData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<PostData> create({
    required String title,
    required String content,
    String editorMode = 'plain',
    int? galgameId,
  }) async {
    final data = await _api.post(
      '/api/v1/posts',
      data: {
        'title': title,
        'content': content,
        'editor_mode': editorMode,
        'galgame_id': ?galgameId,
      },
    );
    return PostData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<PostData> update(
    int id, {
    required String title,
    required String content,
    String editorMode = 'plain',
  }) async {
    final data = await _api.put(
      '/api/v1/posts/$id',
      data: {
        'title': title,
        'content': content,
        'editor_mode': editorMode,
      },
    );
    return PostData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<void> delete(int id) => _api.delete('/api/v1/posts/$id');

  Future<PostLikeData> like(int id) async {
    final data = await _api.post('/api/v1/posts/$id/like');
    return PostLikeData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<PostLikeData> unlike(int id) async {
    final data = await _api.delete('/api/v1/posts/$id/like');
    return PostLikeData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<PostFavoriteData> favorite(int id) async {
    final data = await _api.post('/api/v1/posts/$id/favorite');
    return PostFavoriteData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<PostFavoriteData> unfavorite(int id) async {
    final data = await _api.delete('/api/v1/posts/$id/favorite');
    return PostFavoriteData.fromMap(Map<String, dynamic>.from(data));
  }
}

class CommentService {
  CommentService(this._api);

  final ApiClient _api;

  Future<Paginated<CommentData>> listByPost(
    int postId, {
    int page = 1,
    int limit = 20,
  }) async {
    final data = await _api.get(
      '/api/v2/posts/$postId/comments',
      queryParameters: {'page': page, 'limit': limit},
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: CommentData.fromMap,
    );
  }

  Future<Paginated<CommentData>> listReplies(
    int commentId, {
    int page = 1,
    int limit = 20,
  }) async {
    final data = await _api.get(
      '/api/v2/comments/$commentId/replies',
      queryParameters: {'page': page, 'limit': limit},
    );
    return Paginated.fromMap(
      Map<String, dynamic>.from(data),
      fromItem: CommentData.fromMap,
    );
  }

  Future<CommentData> create(
    int postId, {
    required String content,
    int? parentId,
    int? replyToCommentId,
  }) async {
    final data = await _api.post(
      '/api/v1/posts/$postId/comments',
      data: {
        'content': content,
        'parent_id': ?parentId,
        'reply_to_comment_id': ?replyToCommentId,
      },
    );
    return CommentData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<void> delete(int id) => _api.delete('/api/v1/comments/$id');

  Future<CommentLikeData> like(int id) async {
    final data = await _api.post('/api/v1/comments/$id/like');
    return CommentLikeData.fromMap(Map<String, dynamic>.from(data));
  }

  Future<CommentLikeData> unlike(int id) async {
    final data = await _api.delete('/api/v1/comments/$id/like');
    return CommentLikeData.fromMap(Map<String, dynamic>.from(data));
  }
}
