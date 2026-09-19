import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile/features/game_ratings/models/game_rating_models.dart';
import 'package:mobile/features/game_ratings/widgets/game_rating_widgets.dart';

void main() {
  testWidgets('empty summary renders a useful empty state', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: RatingSummaryCard(summary: GameRatingSummary())),
      ),
    );

    expect(find.text('暂无评分，来发表第一条评分吧'), findsOneWidget);
  });

  testWidgets('severe spoiler review stays hidden until tapped', (
    tester,
  ) async {
    const rating = GameRating(score: 9, spoilerLevel: 2, review: '主角其实是幕后黑手');
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: RatingReview(rating: rating)),
      ),
    );

    expect(find.text('主角其实是幕后黑手'), findsNothing);
    expect(find.text('包含严重剧透，点击查看'), findsOneWidget);

    await tester.tap(find.byKey(const Key('severe-spoiler-cover')));
    await tester.pump();

    expect(find.text('主角其实是幕后黑手'), findsOneWidget);
  });
}
