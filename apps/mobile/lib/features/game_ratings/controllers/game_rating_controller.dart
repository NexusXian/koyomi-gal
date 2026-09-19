import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../providers/app_providers.dart';
import '../models/game_rating_models.dart';
import '../services/game_rating_service.dart';

final gameRatingServiceProvider = Provider<GameRatingRepository>(
  (ref) => GameRatingService(ref.watch(apiClientProvider)),
);

final gameRatingControllerProvider = ChangeNotifierProvider.autoDispose
    .family<GameRatingController, int>(
      (ref, galgameId) =>
          GameRatingController(ref.watch(gameRatingServiceProvider), galgameId),
    );

class GameRatingController extends ChangeNotifier {
  GameRatingController(this._repository, this.galgameId);

  static const pageSize = 20;

  final GameRatingRepository _repository;
  final int galgameId;

  GameRatingSummary? summary;
  GameRating? myRating;
  List<GameRating> ratings = const [];
  RatingSort sort = RatingSort.newest;
  String? summaryError;
  String? listError;
  String? mineError;
  bool loadingSummary = false;
  bool loadingMine = false;
  bool loadingList = false;
  bool hasMore = true;
  int _page = 1;
  Completer<void>? _listCompleter;
  final Set<int> _likeInFlight = {};

  bool isLikeInFlight(int ratingId) => _likeInFlight.contains(ratingId);

  Future<void> loadSummary() async {
    if (loadingSummary) {
      return;
    }
    loadingSummary = true;
    summaryError = null;
    notifyListeners();
    try {
      summary = await _repository.summary(galgameId);
    } catch (error) {
      summaryError = ratingErrorMessage(error);
    } finally {
      loadingSummary = false;
      notifyListeners();
    }
  }

  Future<void> loadMine() async {
    if (loadingMine) {
      return;
    }
    loadingMine = true;
    mineError = null;
    notifyListeners();
    try {
      myRating = await _repository.mine(galgameId);
    } catch (error) {
      mineError = ratingErrorMessage(error);
    } finally {
      loadingMine = false;
      notifyListeners();
    }
  }

  Future<void> loadRatings({bool reset = false}) async {
    if (loadingList) {
      if (reset) {
        await _listCompleter?.future;
        return loadRatings(reset: true);
      }
      return;
    }
    loadingList = true;
    _listCompleter = Completer<void>();
    if (reset) {
      _page = 1;
      hasMore = true;
      listError = null;
    }
    notifyListeners();
    try {
      final result = await _repository.list(
        galgameId,
        sort: sort,
        page: _page,
        pageSize: pageSize,
      );
      final merged = reset ? result.items : [...ratings, ...result.items];
      final deduplicated = <int, GameRating>{};
      final withoutId = <GameRating>[];
      for (final rating in merged) {
        final id = rating.id;
        if (id == null) {
          withoutId.add(rating);
        } else {
          deduplicated[id] = rating;
        }
      }
      ratings = [...deduplicated.values, ...withoutId];
      hasMore = result.hasMore;
      _page = result.page + 1;
      listError = null;
    } catch (error) {
      listError = ratingErrorMessage(error);
    } finally {
      loadingList = false;
      _listCompleter?.complete();
      _listCompleter = null;
      notifyListeners();
    }
  }

  Future<void> changeSort(RatingSort value) async {
    if (sort == value) {
      return;
    }
    sort = value;
    ratings = const [];
    notifyListeners();
    await loadRatings(reset: true);
  }

  Future<void> save(GameRatingDraft draft) async {
    final validation = draft.validate();
    if (validation != null) {
      throw RatingValidationException(validation);
    }
    final saved = myRating == null
        ? await _repository.create(galgameId, draft)
        : await _repository.update(galgameId, draft);
    myRating = saved;
    notifyListeners();
    await _refreshAfterMutation();
  }

  Future<void> deleteMine() async {
    await _repository.delete(galgameId);
    myRating = null;
    notifyListeners();
    await _refreshAfterMutation();
  }

  Future<void> _refreshAfterMutation() async {
    await Future.wait([loadSummary(), loadMine(), loadRatings(reset: true)]);
  }

  Future<bool> toggleLike(int ratingId) async {
    if (_likeInFlight.contains(ratingId)) {
      return false;
    }
    final current = _ratingById(ratingId);
    if (current == null) {
      return false;
    }
    final optimistic = current.copyWith(
      liked: !current.liked,
      likeCount: (current.likeCount + (current.liked ? -1 : 1))
          .clamp(0, 1 << 31)
          .toInt(),
    );
    _likeInFlight.add(ratingId);
    _replaceRating(optimistic);
    notifyListeners();

    try {
      final result = optimistic.liked
          ? await _repository.like(ratingId)
          : await _repository.unlike(ratingId);
      _replaceRating(
        optimistic.copyWith(liked: result.liked, likeCount: result.likeCount),
      );
      return true;
    } catch (_) {
      _replaceRating(current);
      return false;
    } finally {
      _likeInFlight.remove(ratingId);
      notifyListeners();
    }
  }

  GameRating? _ratingById(int id) {
    for (final rating in ratings) {
      if (rating.id == id) {
        return rating;
      }
    }
    return myRating?.id == id ? myRating : null;
  }

  void _replaceRating(GameRating replacement) {
    if (myRating?.id == replacement.id) {
      myRating = replacement;
    }
    ratings = [
      for (final rating in ratings)
        if (rating.id == replacement.id) replacement else rating,
    ];
  }
}

class RatingValidationException implements Exception {
  const RatingValidationException(this.message);

  final String message;

  @override
  String toString() => message;
}
