import 'package:flutter/material.dart';

import '../models/user_models.dart';
import 'app_image.dart';

class UserAvatar extends StatelessWidget {
  const UserAvatar({super.key, this.url, this.size = 40});

  final String? url;
  final double size;

  @override
  Widget build(BuildContext context) {
    if (url == null || url!.isEmpty) {
      return CircleAvatar(
        radius: size / 2,
        backgroundColor: Theme.of(context).colorScheme.surfaceContainerHighest,
        child: Icon(
          Icons.person,
          size: size * 0.55,
          color: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      );
    }
    return ClipOval(
      child: AppImage(url: url, width: size, height: size),
    );
  }
}

class LevelBadge extends StatelessWidget {
  const LevelBadge({super.key, this.level});

  final LevelSummary? level;

  @override
  Widget build(BuildContext context) {
    if (level == null || level!.level == null) {
      return const SizedBox.shrink();
    }
    final color = _parseColor(level!.color) ??
        Theme.of(context).colorScheme.primary;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(
        'Lv.${level!.level}${level!.name == null ? '' : ' ${level!.name}'}',
        style: TextStyle(
          fontSize: 10,
          fontWeight: FontWeight.w600,
          color: color,
        ),
      ),
    );
  }
}

Color? _parseColor(String? hex) {
  if (hex == null || hex.isEmpty) {
    return null;
  }
  var value = hex.replaceFirst('#', '');
  if (value.length == 6) {
    value = 'FF$value';
  }
  final parsed = int.tryParse(value, radix: 16);
  return parsed == null ? null : Color(parsed);
}

class TagChips extends StatelessWidget {
  const TagChips({super.key, required this.names, this.max = 3});

  final List<String> names;
  final int max;

  @override
  Widget build(BuildContext context) {
    final shown = names.take(max).toList();
    if (shown.isEmpty) {
      return const SizedBox.shrink();
    }
    return Wrap(
      spacing: 6,
      runSpacing: 4,
      children: [
        for (final name in shown)
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
            decoration: BoxDecoration(
              color: Theme.of(context)
                  .colorScheme
                  .primaryContainer
                  .withValues(alpha: 0.45),
              borderRadius: BorderRadius.circular(6),
            ),
            child: Text(
              name,
              style: TextStyle(
                fontSize: 11,
                color: Theme.of(context).colorScheme.primary,
              ),
            ),
          ),
      ],
    );
  }
}

class StatRow extends StatelessWidget {
  const StatRow({
    super.key,
    required this.icon,
    required this.label,
    this.color,
  });

  final IconData icon;
  final String label;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 14, color: color ?? Theme.of(context).colorScheme.onSurfaceVariant),
        const SizedBox(width: 3),
        Text(
          label,
          style: TextStyle(
            fontSize: 12,
            color: color ?? Theme.of(context).colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}
