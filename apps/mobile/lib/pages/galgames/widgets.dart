import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/constants/domain.dart';
import '../../core/utils/format.dart';
import '../../models/galgame_models.dart';
import '../../widgets/app_image.dart';
import '../../widgets/user_widgets.dart';

class GalgameCard extends StatelessWidget {
  const GalgameCard({super.key, required this.galgame, this.onTap});

  final GalgameListItem galgame;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onTap,
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            AppImage(
              url: galgame.coverUrl,
              sensitive: galgame.coverSensitive,
              width: 92,
              height: 122,
            ),
            Expanded(
              child: Padding(
                padding: const EdgeInsets.all(10),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      galgame.title ?? '',
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 15,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    if (galgame.originalTitle != null &&
                        galgame.originalTitle!.isNotEmpty)
                      Text(
                        galgame.originalTitle!,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          fontSize: 12,
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    const SizedBox(height: 4),
                    Row(
                      children: [
                        if (galgame.rating?.average != null)
                          Padding(
                            padding: const EdgeInsets.only(right: 10),
                            child: StatRow(
                              icon: Icons.star,
                              label: galgame.rating!.average!
                                  .toStringAsFixed(1),
                              color: Colors.amber.shade700,
                            ),
                          ),
                        StatRow(
                          icon: Icons.favorite_outline,
                          label: formatCount(
                              galgame.statistics?.favoriteCount),
                        ),
                      ],
                    ),
                    const SizedBox(height: 4),
                    Text(
                      [
                        if (galgame.developer?.name != null)
                          galgame.developer!.name!,
                        formatDate(galgame.releaseDate),
                        ageRatingLabel(galgame.ageRating),
                      ].join(' · '),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        fontSize: 12,
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                    if (galgame.tags.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(top: 6),
                        child: TagChips(
                          names: galgame.tags
                              .map((tag) => tag.name ?? '')
                              .where((name) => name.isNotEmpty)
                              .toList(),
                        ),
                      ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class GalgameGridCard extends StatelessWidget {
  const GalgameGridCard({
    super.key,
    required this.id,
    required this.title,
    required this.coverUrl,
    required this.sensitive,
    required this.subtitle,
    this.badge,
    this.onTap,
  });

  final int? id;
  final String? title;
  final String? coverUrl;
  final bool sensitive;
  final String? subtitle;
  final String? badge;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return GestureDetector(
      onTap: onTap,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          AspectRatio(
            aspectRatio: 3 / 4,
            child: AppImage(
              url: coverUrl,
              sensitive: sensitive,
              borderRadius: BorderRadius.circular(10),
            ),
          ),
          const SizedBox(height: 6),
          Text(
            title ?? '',
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
          ),
          if (subtitle != null)
            Text(
              subtitle!,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                fontSize: 11,
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
        ],
      ),
    );
  }
}

/// Horizontal poster list used by the home page.
class GalgameHorizontalList extends StatelessWidget {
  const GalgameHorizontalList({super.key, required this.galgames});

  final List<HomeGalgame> galgames;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 220,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 16),
        itemCount: galgames.length,
        separatorBuilder: (_, _) => const SizedBox(width: 10),
        itemBuilder: (context, index) {
          final galgame = galgames[index];
          return SizedBox(
            width: 112,
            child: GalgameGridCard(
              id: galgame.id,
              title: galgame.title,
              coverUrl: galgame.coverUrl,
              sensitive: galgame.coverSensitive,
              subtitle: galgame.ratingAverage != null
                  ? '★ ${galgame.ratingAverage!.toStringAsFixed(1)}'
                  : galgame.developer?.name,
              onTap: galgame.id == null
                  ? null
                  : () => context.push('/galgames/${galgame.id}'),
            ),
          );
        },
      ),
    );
  }
}
