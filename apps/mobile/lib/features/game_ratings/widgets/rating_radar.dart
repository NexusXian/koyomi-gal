import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../models/game_rating_models.dart';

class RatingRadar extends StatelessWidget {
  const RatingRadar({super.key, required this.summary});

  final GameRatingSummary summary;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final values = [
      for (final dimension in RatingDimension.values)
        summary.dimension(dimension).average,
    ];
    return SizedBox(
      height: 190,
      child: CustomPaint(
        painter: _RatingRadarPainter(
          values: values,
          labels: [for (final item in RatingDimension.values) item.label],
          gridColor: theme.colorScheme.outlineVariant,
          fillColor: theme.colorScheme.primary.withValues(alpha: 0.2),
          strokeColor: theme.colorScheme.primary,
          labelColor: theme.colorScheme.onSurfaceVariant,
        ),
        size: Size.infinite,
      ),
    );
  }
}

class _RatingRadarPainter extends CustomPainter {
  const _RatingRadarPainter({
    required this.values,
    required this.labels,
    required this.gridColor,
    required this.fillColor,
    required this.strokeColor,
    required this.labelColor,
  });

  final List<double?> values;
  final List<String> labels;
  final Color gridColor;
  final Color fillColor;
  final Color strokeColor;
  final Color labelColor;

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2 + 4);
    final radius = math.min(size.width, size.height) * 0.34;
    const startAngle = -math.pi / 2;
    final step = math.pi * 2 / values.length;
    final gridPaint = Paint()
      ..color = gridColor
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1;

    Offset point(int index, double scale) => Offset(
      center.dx + math.cos(startAngle + step * index) * radius * scale,
      center.dy + math.sin(startAngle + step * index) * radius * scale,
    );

    for (var ring = 1; ring <= 5; ring++) {
      final path = Path();
      for (var index = 0; index < values.length; index++) {
        final current = point(index, ring / 5);
        index == 0
            ? path.moveTo(current.dx, current.dy)
            : path.lineTo(current.dx, current.dy);
      }
      path.close();
      canvas.drawPath(path, gridPaint);
    }
    for (var index = 0; index < values.length; index++) {
      canvas.drawLine(center, point(index, 1), gridPaint);
    }

    if (values.whereType<double>().length >= 3) {
      final scorePath = Path();
      for (var index = 0; index < values.length; index++) {
        final value = (values[index] ?? 0).clamp(0, 10) / 10;
        final current = point(index, value);
        index == 0
            ? scorePath.moveTo(current.dx, current.dy)
            : scorePath.lineTo(current.dx, current.dy);
      }
      scorePath.close();
      canvas.drawPath(scorePath, Paint()..color = fillColor);
      canvas.drawPath(
        scorePath,
        Paint()
          ..color = strokeColor
          ..style = PaintingStyle.stroke
          ..strokeWidth = 2,
      );
    }

    for (var index = 0; index < labels.length; index++) {
      final anchor = point(index, 1.28);
      final text = TextPainter(
        text: TextSpan(
          text: labels[index],
          style: TextStyle(color: labelColor, fontSize: 11),
        ),
        textDirection: TextDirection.ltr,
      )..layout();
      text.paint(
        canvas,
        Offset(anchor.dx - text.width / 2, anchor.dy - text.height / 2),
      );
    }
  }

  @override
  bool shouldRepaint(covariant _RatingRadarPainter oldDelegate) =>
      oldDelegate.values != values ||
      oldDelegate.gridColor != gridColor ||
      oldDelegate.strokeColor != strokeColor;
}
