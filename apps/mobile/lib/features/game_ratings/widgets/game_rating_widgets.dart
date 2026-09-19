import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/constants/domain.dart';
import '../../../core/utils/format.dart';
import '../../../providers/app_providers.dart';
import '../../../widgets/common_views.dart';
import '../../../widgets/markdown_view.dart';
import '../../../widgets/user_widgets.dart';
import '../controllers/game_rating_controller.dart';
import '../models/game_rating_models.dart';
import '../services/game_rating_service.dart';
import 'rating_editor_sheet.dart';
import 'rating_radar.dart';

class GameRatingOverview extends ConsumerStatefulWidget {
  const GameRatingOverview({super.key, required this.galgameId});

  final int galgameId;

  @override
  ConsumerState<GameRatingOverview> createState() => _GameRatingOverviewState();
}

class _GameRatingOverviewState extends ConsumerState<GameRatingOverview> {
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final controller = ref.read(
        gameRatingControllerProvider(widget.galgameId),
      );
      controller.loadSummary();
      if (ref.read(authControllerProvider).isAuthenticated) {
        controller.loadMine();
      }
    });
  }

  Future<void> _edit() async {
    final auth = ref.read(authControllerProvider);
    if (!auth.isAuthenticated) {
      context.push('/login');
      return;
    }
    final controller = ref.read(gameRatingControllerProvider(widget.galgameId));
    final result = await showRatingEditorSheet(
      context,
      current: controller.myRating,
    );
    if (result == null || !mounted) {
      return;
    }
    if (result.delete) {
      final confirmed = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('删除评分'),
          content: const Text('确定删除你的评分和评价吗？'),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text('取消'),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text('删除'),
            ),
          ],
        ),
      );
      if (confirmed != true || !mounted) {
        return;
      }
    }

    setState(() => _saving = true);
    try {
      if (result.delete) {
        await controller.deleteMine();
      } else {
        await controller.save(result.draft!);
      }
      if (mounted) {
        showAppSnackBar(context, result.delete ? '评分已删除' : '评分已保存');
      }
    } catch (error) {
      if (mounted) {
        final message = error is RatingValidationException
            ? error.message
            : ratingErrorMessage(error);
        showAppSnackBar(context, message, error: true);
      }
    } finally {
      if (mounted) {
        setState(() => _saving = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final controller = ref.watch(
      gameRatingControllerProvider(widget.galgameId),
    );
    final auth = ref.watch(authControllerProvider);
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
      child: RatingSummaryCard(
        summary: controller.summary,
        loading: controller.loadingSummary,
        error: controller.summaryError,
        actionLabel: auth.isAuthenticated
            ? controller.myRating == null
                  ? '发表评分'
                  : '编辑我的评分'
            : '登录后评分',
        actionScore: controller.myRating?.score,
        actionLoading: _saving || controller.loadingMine,
        onAction: _edit,
        onRetry: controller.loadSummary,
      ),
    );
  }
}

class RatingSummaryCard extends StatelessWidget {
  const RatingSummaryCard({
    super.key,
    required this.summary,
    this.loading = false,
    this.error,
    this.actionLabel,
    this.actionScore,
    this.actionLoading = false,
    this.onAction,
    this.onRetry,
  });

  final GameRatingSummary? summary;
  final bool loading;
  final String? error;
  final String? actionLabel;
  final int? actionScore;
  final bool actionLoading;
  final VoidCallback? onAction;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Expanded(
                  child: Text(
                    '玩家评分',
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
                  ),
                ),
                if (onAction != null)
                  FilledButton.tonalIcon(
                    onPressed: actionLoading ? null : onAction,
                    icon: actionLoading
                        ? const SizedBox(
                            width: 14,
                            height: 14,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.rate_review_outlined, size: 17),
                    label: Text(
                      actionScore == null
                          ? actionLabel ?? '评分'
                          : '${actionLabel ?? '我的评分'} $actionScore',
                      style: const TextStyle(fontSize: 12),
                    ),
                  ),
              ],
            ),
            if (loading && summary == null)
              const Padding(
                padding: EdgeInsets.symmetric(vertical: 28),
                child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
              )
            else if (error != null && summary == null)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 14),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        error!,
                        style: TextStyle(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ),
                    TextButton(onPressed: onRetry, child: const Text('重试')),
                  ],
                ),
              )
            else if (summary == null || summary!.isEmpty)
              const Padding(
                padding: EdgeInsets.symmetric(vertical: 24),
                child: Center(child: Text('暂无评分，来发表第一条评分吧')),
              )
            else
              _SummaryContent(summary: summary!),
          ],
        ),
      ),
    );
  }
}

class _SummaryContent extends StatelessWidget {
  const _SummaryContent({required this.summary});

  final GameRatingSummary summary;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      children: [
        const SizedBox(height: 12),
        Row(
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            SizedBox(
              width: 76,
              child: Column(
                children: [
                  Text(
                    summary.average?.toStringAsFixed(1) ?? '-',
                    style: TextStyle(
                      fontSize: 32,
                      height: 1,
                      fontWeight: FontWeight.w800,
                      color: theme.colorScheme.primary,
                    ),
                  ),
                  const SizedBox(height: 5),
                  Text(
                    '${summary.count} 人评分',
                    style: const TextStyle(fontSize: 11),
                  ),
                ],
              ),
            ),
            Expanded(child: RatingRadar(summary: summary)),
          ],
        ),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (final dimension in RatingDimension.values)
              _DimensionValue(
                label: dimension.label,
                value: summary.dimension(dimension).average,
                count: summary.dimension(dimension).count,
              ),
          ],
        ),
        if (summary.recommendationCounts.isNotEmpty) ...[
          const SizedBox(height: 12),
          Align(
            alignment: Alignment.centerLeft,
            child: Wrap(
              spacing: 10,
              runSpacing: 6,
              children: [
                for (final item in RatingRecommendation.values)
                  Text(
                    '${item.label} ${summary.recommendationCounts[item.value] ?? 0}',
                    style: TextStyle(
                      fontSize: 11,
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
              ],
            ),
          ),
        ],
      ],
    );
  }
}

class _DimensionValue extends StatelessWidget {
  const _DimensionValue({
    required this.label,
    required this.value,
    required this.count,
  });

  final String label;
  final double? value;
  final int count;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 5),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        '$label ${value?.toStringAsFixed(1) ?? '-'} ($count)',
        style: const TextStyle(fontSize: 11),
      ),
    );
  }
}

class GameRatingList extends ConsumerStatefulWidget {
  const GameRatingList({super.key, required this.galgameId});

  final int galgameId;

  @override
  ConsumerState<GameRatingList> createState() => _GameRatingListState();
}

class _GameRatingListState extends ConsumerState<GameRatingList> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final controller = ref.read(
        gameRatingControllerProvider(widget.galgameId),
      );
      if (controller.ratings.isEmpty && controller.listError == null) {
        controller.loadRatings(reset: true);
      }
    });
  }

  Future<void> _like(GameRating rating) async {
    if (!ref.read(authControllerProvider).isAuthenticated) {
      context.push('/login');
      return;
    }
    final id = rating.id;
    if (id == null) {
      return;
    }
    final success = await ref
        .read(gameRatingControllerProvider(widget.galgameId))
        .toggleLike(id);
    if (!success && mounted) {
      showAppSnackBar(context, '点赞失败，请稍后重试', error: true);
    }
  }

  @override
  Widget build(BuildContext context) {
    final controller = ref.watch(
      gameRatingControllerProvider(widget.galgameId),
    );
    if (controller.loadingList && controller.ratings.isEmpty) {
      return const LoadingView();
    }
    if (controller.listError != null && controller.ratings.isEmpty) {
      return ErrorView(
        message: controller.listError!,
        onRetry: () => controller.loadRatings(reset: true),
      );
    }

    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 10, 16, 0),
          child: Row(
            children: [
              const Expanded(
                child: Text(
                  '评分与评价',
                  style: TextStyle(fontWeight: FontWeight.w600),
                ),
              ),
              DropdownButton<RatingSort>(
                value: controller.sort,
                underline: const SizedBox.shrink(),
                isDense: true,
                items: [
                  for (final sort in RatingSort.values)
                    DropdownMenuItem(value: sort, child: Text(sort.label)),
                ],
                onChanged: controller.loadingList
                    ? null
                    : (value) {
                        if (value != null) {
                          controller.changeSort(value);
                        }
                      },
              ),
            ],
          ),
        ),
        Expanded(
          child: controller.ratings.isEmpty
              ? const EmptyView(hint: '暂无评分')
              : NotificationListener<ScrollNotification>(
                  onNotification: (notification) {
                    if (notification is ScrollEndNotification &&
                        notification.metrics.pixels >=
                            notification.metrics.maxScrollExtent - 120 &&
                        controller.hasMore &&
                        !controller.loadingList) {
                      controller.loadRatings();
                    }
                    return false;
                  },
                  child: RefreshIndicator(
                    onRefresh: () => controller.loadRatings(reset: true),
                    child: ListView.separated(
                      padding: const EdgeInsets.all(16),
                      itemCount:
                          controller.ratings.length +
                          (controller.loadingList ? 1 : 0),
                      separatorBuilder: (_, _) => const SizedBox(height: 10),
                      itemBuilder: (context, index) {
                        if (index == controller.ratings.length) {
                          return const Padding(
                            padding: EdgeInsets.all(12),
                            child: Center(
                              child: SizedBox(
                                width: 20,
                                height: 20,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                ),
                              ),
                            ),
                          );
                        }
                        final rating = controller.ratings[index];
                        return GameRatingCard(
                          rating: rating,
                          likeLoading:
                              rating.id != null &&
                              controller.isLikeInFlight(rating.id!),
                          onLike: () => _like(rating),
                        );
                      },
                    ),
                  ),
                ),
        ),
      ],
    );
  }
}

class GameRatingCard extends StatelessWidget {
  const GameRatingCard({
    super.key,
    required this.rating,
    required this.onLike,
    this.likeLoading = false,
  });

  final GameRating rating;
  final VoidCallback onLike;
  final bool likeLoading;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final author = rating.author;
    final recommendation = RatingRecommendation.fromValue(
      rating.recommendation,
    );
    final spoiler = RatingSpoilerLevel.fromValue(rating.spoilerLevel);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                TappableUser(
                  username: author?.username,
                  child: UserAvatar(url: author?.avatarUrl, size: 34),
                ),
                const SizedBox(width: 9),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        author?.displayName?.isNotEmpty == true
                            ? author!.displayName!
                            : author?.username ?? '用户',
                        style: const TextStyle(
                          fontSize: 13,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      Text(
                        [
                          if (rating.playStatus != null)
                            domainLabel(userStateOptions, rating.playStatus),
                          formatRelative(rating.createdAt),
                        ].join(' · '),
                        style: TextStyle(
                          fontSize: 11,
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                ),
                if (rating.score != null)
                  Text(
                    '${rating.score}',
                    style: TextStyle(
                      fontSize: 25,
                      fontWeight: FontWeight.w800,
                      color: theme.colorScheme.primary,
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 9),
            Wrap(
              spacing: 6,
              runSpacing: 6,
              children: [
                if (rating.isPartial) const _RatingLabel(text: '部分评分'),
                if (recommendation != null)
                  _RatingLabel(text: recommendation.label),
                if (spoiler != null && spoiler != RatingSpoilerLevel.none)
                  _RatingLabel(text: spoiler.label, warning: true),
                for (final dimension in RatingDimension.values)
                  if (rating.dimensions.valueOf(dimension) != null)
                    _RatingLabel(
                      text:
                          '${dimension.label} ${rating.dimensions.valueOf(dimension)}',
                    ),
              ],
            ),
            if (rating.review.trim().isNotEmpty) ...[
              const SizedBox(height: 10),
              RatingReview(rating: rating),
            ],
            const SizedBox(height: 6),
            Align(
              alignment: Alignment.centerRight,
              child: TextButton.icon(
                onPressed: likeLoading ? null : onLike,
                icon: Icon(
                  rating.liked ? Icons.favorite : Icons.favorite_border,
                  size: 17,
                  color: rating.liked ? theme.colorScheme.error : null,
                ),
                label: Text('${rating.likeCount}'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class RatingReview extends StatefulWidget {
  const RatingReview({super.key, required this.rating});

  final GameRating rating;

  @override
  State<RatingReview> createState() => _RatingReviewState();
}

class _RatingReviewState extends State<RatingReview> {
  bool _revealed = false;

  @override
  Widget build(BuildContext context) {
    if (widget.rating.spoilerLevel == RatingSpoilerLevel.severe.value &&
        !_revealed) {
      return InkWell(
        key: const Key('severe-spoiler-cover'),
        borderRadius: BorderRadius.circular(8),
        onTap: () => setState(() => _revealed = true),
        child: Container(
          width: double.infinity,
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 18),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.errorContainer
                .withValues(alpha: 0.35),
            borderRadius: BorderRadius.circular(8),
          ),
          child: const Column(
            children: [
              Icon(Icons.visibility_off_outlined, size: 20),
              SizedBox(height: 4),
              Text('包含严重剧透，点击查看', style: TextStyle(fontSize: 12)),
            ],
          ),
        ),
      );
    }
    if (widget.rating.editorMode == 'markdown') {
      return MarkdownView(data: widget.rating.review, selectable: false);
    }
    return Text(
      widget.rating.review,
      style: const TextStyle(fontSize: 13, height: 1.55),
    );
  }
}

class _RatingLabel extends StatelessWidget {
  const _RatingLabel({required this.text, this.warning = false});

  final String text;
  final bool warning;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final color = warning ? scheme.error : scheme.primary;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(text, style: TextStyle(fontSize: 10, color: color)),
    );
  }
}
