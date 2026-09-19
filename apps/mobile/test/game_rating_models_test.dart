import 'package:flutter_test/flutter_test.dart';
import 'package:mobile/features/game_ratings/models/game_rating_models.dart';

void main() {
  group('rating models', () {
    test('parses nullable dimensions', () {
      final rating = GameRating.fromMap(const {
        'id': 12,
        'overall': 8,
        'dimensions': {
          'visual': 7,
          'story': 9,
          'music': null,
          'character': null,
          'branch': 6,
          'system': 8,
          'voice': null,
          'replay': 5,
        },
        'recommendation': true,
        'spoiler_level': 0,
      });

      expect(rating.dimensions.story, 9);
      expect(rating.dimensions.character, isNull);
      expect(rating.dimensions.visual, 7);
      expect(rating.dimensions.music, isNull);
      expect(rating.dimensions.voice, isNull);
      expect(rating.recommendation, 1);
      expect(rating.isPartial, isTrue);
    });

    test('parses nested summary values and counts', () {
      final summary = GameRatingSummary.fromMap(const {
        'overall': '8.25',
        'count': 40,
        'dimensions': {
          'story': {'average': 9.1, 'count': 32},
          'character': {'average': null, 'count': 0},
          'visual': {'average': '7.5', 'count': 28},
        },
        'recommendation_counts': {'-1': 2, '0': 3, '1': 20, '2': 15},
      });

      expect(summary.average, 8.25);
      expect(summary.count, 40);
      expect(summary.dimension(RatingDimension.story).average, 9.1);
      expect(summary.dimension(RatingDimension.story).count, 32);
      expect(summary.dimension(RatingDimension.character).average, isNull);
      expect(summary.dimension(RatingDimension.visual).average, 7.5);
      expect(summary.recommendationCounts[-1], 2);
      expect(summary.recommendationCounts[2], 15);
    });
  });

  group('rating editor validation', () {
    test('uses exact recommendation and spoiler enum values', () {
      expect(
        RatingRecommendation.values.map((item) => item.value),
        orderedEquals([-1, 0, 1, 2]),
      );
      expect(RatingRecommendation.neutral.label, '中立');
      expect(
        RatingSpoilerLevel.values.map((item) => item.value),
        orderedEquals([0, 1, 2]),
      );
      expect(RatingSpoilerLevel.mild.label, '部分剧透');
    });

    test('serializes the shared API payload shape', () {
      final payload = const GameRatingDraft(
        score: 8,
        dimensions: RatingDimensions(story: 9, voice: 7),
        recommendation: 1,
        spoilerLevel: 2,
        review: ' **很好** ',
        editorMode: 'markdown',
      ).toMap();

      expect(payload['overall'], 8);
      expect(payload['recommendation'], 1);
      expect(payload['spoiler_level'], 2);
      expect(payload['review_text'], '**很好**');
      expect(payload['story'], 9);
      expect(payload['voice'], 7);
      expect(payload, isNot(contains('dimensions')));
    });

    test('validates required score, ranges and enums', () {
      expect(const GameRatingDraft(score: null).validate(), '请选择综合评分');
      expect(const GameRatingDraft(score: 11).validate(), contains('1 到 10'));
      expect(
        const GameRatingDraft(
          score: 8,
          dimensions: RatingDimensions(story: 0),
        ).validate(),
        contains('剧情'),
      );
      expect(
        const GameRatingDraft(score: 8, recommendation: 3).validate(),
        '推荐度选项无效',
      );
      expect(
        const GameRatingDraft(score: 8, spoilerLevel: 3).validate(),
        '剧透等级选项无效',
      );
      expect(
        const GameRatingDraft(
          score: 8,
          recommendation: -1,
          spoilerLevel: 2,
          editorMode: 'markdown',
        ).validate(),
        isNull,
      );
    });
  });
}
