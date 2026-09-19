import 'package:flutter_test/flutter_test.dart';
import 'package:mobile/models/user_models.dart';

void main() {
  group('ProfileGalgameData', () {
    test('parses detailed rating fields', () {
      final item = ProfileGalgameData.fromMap({
        'id': 12,
        'title': 'Example Game',
        'score': 9,
        'story': 10,
        'voice': 8,
        'recommendation': 2,
        'review_text': 'Great story.',
        'spoiler_level': 1,
        'created_at': '2026-09-19T10:00:00Z',
      });

      expect(item.score, 9);
      expect(item.story, 10);
      expect(item.voice, 8);
      expect(item.visual, isNull);
      expect(item.recommendation, 2);
      expect(item.reviewText, 'Great story.');
      expect(item.spoilerLevel, 1);
      expect(item.hasDetailedRating, isTrue);
    });

    test('treats favorite entries without rating as plain rows', () {
      final item = ProfileGalgameData.fromMap({
        'id': 13,
        'title': 'Favorite Only',
        'created_at': '2026-09-19T10:00:00Z',
      });

      expect(item.score, isNull);
      expect(item.reviewText, isNull);
      expect(item.hasDetailedRating, isFalse);
    });

    test('null dimension stays null instead of zero', () {
      final item = ProfileGalgameData.fromMap({
        'id': 14,
        'score': 8,
        'voice': null,
      });

      expect(item.voice, isNull);
    });
  });
}
