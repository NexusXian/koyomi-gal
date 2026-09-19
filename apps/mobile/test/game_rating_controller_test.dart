import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:mobile/features/game_ratings/controllers/game_rating_controller.dart';
import 'package:mobile/features/game_ratings/models/game_rating_models.dart';
import 'package:mobile/features/game_ratings/services/game_rating_service.dart';

void main() {
  test('like is optimistic and locks duplicate in-flight requests', () async {
    final repository = _FakeRatingRepository();
    final completer = Completer<RatingLikeResult>();
    repository.likeHandler = (_) => completer.future;
    final controller = GameRatingController(repository, 4)
      ..ratings = const [GameRating(id: 7, likeCount: 2)];

    final first = controller.toggleLike(7);
    expect(controller.ratings.single.liked, isTrue);
    expect(controller.ratings.single.likeCount, 3);
    expect(controller.isLikeInFlight(7), isTrue);

    expect(await controller.toggleLike(7), isFalse);
    expect(repository.likeCalls, 1);

    completer.complete(const RatingLikeResult(liked: true, likeCount: 3));
    expect(await first, isTrue);
    expect(controller.isLikeInFlight(7), isFalse);
  });

  test('failed optimistic like restores the previous state', () async {
    final repository = _FakeRatingRepository()
      ..likeHandler = (_) => Future.error(Exception('offline'));
    final controller = GameRatingController(repository, 4)
      ..ratings = const [GameRating(id: 7, likeCount: 2)];

    expect(await controller.toggleLike(7), isFalse);
    expect(controller.ratings.single.liked, isFalse);
    expect(controller.ratings.single.likeCount, 2);
  });
}

class _FakeRatingRepository implements GameRatingRepository {
  Future<RatingLikeResult> Function(int id)? likeHandler;
  int likeCalls = 0;

  @override
  Future<RatingLikeResult> like(int ratingId) {
    likeCalls++;
    return likeHandler?.call(ratingId) ??
        Future.value(const RatingLikeResult(liked: true, likeCount: 1));
  }

  @override
  Future<RatingLikeResult> unlike(int ratingId) async =>
      const RatingLikeResult(liked: false, likeCount: 0);

  @override
  Future<GameRating> create(int galgameId, GameRatingDraft draft) =>
      throw UnimplementedError();

  @override
  Future<void> delete(int galgameId) => throw UnimplementedError();

  @override
  Future<GameRatingPage> list(
    int galgameId, {
    required RatingSort sort,
    required int page,
    required int pageSize,
  }) => throw UnimplementedError();

  @override
  Future<GameRating?> mine(int galgameId) => throw UnimplementedError();

  @override
  Future<GameRatingSummary> summary(int galgameId) =>
      throw UnimplementedError();

  @override
  Future<GameRating> update(int galgameId, GameRatingDraft draft) =>
      throw UnimplementedError();
}
