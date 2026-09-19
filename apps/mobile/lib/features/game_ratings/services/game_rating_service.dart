import 'package:dio/dio.dart';

import '../../../core/api/api_client.dart';
import '../../../core/api/api_exception.dart';
import '../models/game_rating_models.dart';

abstract interface class GameRatingRepository {
  Future<GameRatingSummary> summary(int galgameId);

  Future<GameRatingPage> list(
    int galgameId, {
    required RatingSort sort,
    required int page,
    required int pageSize,
  });

  Future<GameRating?> mine(int galgameId);

  Future<GameRating> create(int galgameId, GameRatingDraft draft);

  Future<GameRating> update(int galgameId, GameRatingDraft draft);

  Future<void> delete(int galgameId);

  Future<RatingLikeResult> like(int ratingId);

  Future<RatingLikeResult> unlike(int ratingId);
}

class GameRatingService implements GameRatingRepository {
  GameRatingService(this._api);

  final ApiClient _api;

  @override
  Future<GameRatingSummary> summary(int galgameId) async {
    final data = await _api.get('/api/v1/galgames/$galgameId/ratings/summary');
    return GameRatingSummary.fromMap(Map<String, dynamic>.from(data));
  }

  @override
  Future<GameRatingPage> list(
    int galgameId, {
    required RatingSort sort,
    required int page,
    required int pageSize,
  }) async {
    final data = await _api.get(
      '/api/v1/galgames/$galgameId/ratings',
      queryParameters: {
        'sort': sort.value,
        'page': page,
        'page_size': pageSize,
      },
    );
    return GameRatingPage.fromMap(Map<String, dynamic>.from(data));
  }

  @override
  Future<GameRating?> mine(int galgameId) async {
    try {
      final data = await _api.get('/api/v1/galgames/$galgameId/ratings/me');
      if (data == null) {
        return null;
      }
      return GameRating.fromMap(Map<String, dynamic>.from(data));
    } on ApiException catch (error) {
      if (error.code == 404) {
        return null;
      }
      rethrow;
    } on DioException catch (error) {
      if (error.response?.statusCode == 404) {
        return null;
      }
      rethrow;
    }
  }

  @override
  Future<GameRating> create(int galgameId, GameRatingDraft draft) async {
    final data = await _api.put(
      '/api/v1/galgames/$galgameId/ratings/me',
      data: draft.toMap(),
    );
    return GameRating.fromMap(Map<String, dynamic>.from(data));
  }

  @override
  Future<GameRating> update(int galgameId, GameRatingDraft draft) async {
    final data = await _api.put(
      '/api/v1/galgames/$galgameId/ratings/me',
      data: draft.toMap(),
    );
    return GameRating.fromMap(Map<String, dynamic>.from(data));
  }

  @override
  Future<void> delete(int galgameId) =>
      _api.delete('/api/v1/galgames/$galgameId/ratings/me');

  @override
  Future<RatingLikeResult> like(int ratingId) async {
    final data = await _api.post('/api/v1/game-ratings/$ratingId/like');
    return RatingLikeResult.fromMap(Map<String, dynamic>.from(data));
  }

  @override
  Future<RatingLikeResult> unlike(int ratingId) async {
    final data = await _api.delete('/api/v1/game-ratings/$ratingId/like');
    return RatingLikeResult.fromMap(Map<String, dynamic>.from(data));
  }
}

String ratingErrorMessage(Object error) {
  final source = apiErrorMessage(error);
  final normalized = source.toLowerCase();
  if (normalized.contains('not found') || normalized.contains('不存在')) {
    return '评分不存在或已被删除';
  }
  if (normalized.contains('already') || normalized.contains('已评分')) {
    return '你已经发表过评分';
  }
  if (normalized.contains('score') ||
      normalized.contains('recommendation') ||
      normalized.contains('spoiler') ||
      normalized.contains('invalid rating')) {
    return '评分内容无效，请检查后重试';
  }
  if (normalized.contains('unauthorized') ||
      normalized.contains('permission') ||
      normalized.contains('登录')) {
    return '请登录后操作';
  }
  if (source.contains(RegExp(r'[\u4e00-\u9fff]'))) {
    return source;
  }
  return '评分请求失败，请稍后重试';
}
