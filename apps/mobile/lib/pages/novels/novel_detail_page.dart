import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/constants/domain.dart';
import '../../core/utils/format.dart';
import '../../models/novel_models.dart';
import '../../providers/app_providers.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';
import '../../widgets/markdown_view.dart';
import '../../widgets/user_widgets.dart';

class NovelDetailPage extends ConsumerStatefulWidget {
  const NovelDetailPage({super.key, required this.id});

  final int id;

  @override
  ConsumerState<NovelDetailPage> createState() => _NovelDetailPageState();
}

class _NovelDetailPageState extends ConsumerState<NovelDetailPage> {
  NovelDetail? _novel;
  String? _error;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final novel = await ref.read(novelServiceProvider).get(widget.id);
      if (!mounted) {
        return;
      }
      setState(() {
        _novel = novel;
        _loading = false;
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _error = apiErrorMessage(error);
        _loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return Scaffold(
        appBar: AppBar(),
        body: const LoadingView(),
      );
    }
    if (_error != null) {
      return Scaffold(
        appBar: AppBar(),
        body: ErrorView(message: _error!, onRetry: _load),
      );
    }
    final novel = _novel!;
    return Scaffold(
      appBar: AppBar(
        actions: [
          IconButton(
            icon: const Icon(Icons.edit_outlined),
            tooltip: '编辑',
            onPressed: () => context.push('/novels/${widget.id}/edit'),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
        children: [
          _buildHeader(novel),
          if (novel.summary?.isNotEmpty == true) ...[
            const SizedBox(height: 12),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: MarkdownView(data: novel.summary!, selectable: false),
              ),
            ),
          ],
          const SizedBox(height: 16),
          _SectionTitle(
            title: '卷册 (${novel.volumes?.length ?? 0})',
            action: TextButton.icon(
              onPressed: () => context
                  .push('/novels/${widget.id}/volumes/new')
                  .then((_) => _load()),
              icon: const Icon(Icons.add, size: 16),
              label: const Text('添加卷', style: TextStyle(fontSize: 13)),
            ),
          ),
          if (novel.volumes?.isEmpty ?? true)
            const Padding(
              padding: EdgeInsets.all(16),
              child: Center(child: Text('暂无卷册')),
            )
          else
            ...novel.volumes!.map(_buildVolumeTile),
          if (novel.relatedGalgames?.isNotEmpty == true) ...[
            const SizedBox(height: 16),
            const _SectionTitle(title: '关联 Galgame'),
            ...novel.relatedGalgames!.map(_buildRelatedGalgame),
          ],
        ],
      ),
    );
  }

  Widget _buildHeader(NovelDetail novel) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            AppImage(
              url: novel.coverUrl,
              sensitive: novel.isCoverSensitive,
              width: 96,
              height: 134,
              borderRadius: BorderRadius.circular(8),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    novel.title ?? '',
                    style: const TextStyle(
                      fontSize: 17,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  if (novel.originalTitle?.isNotEmpty == true)
                    Text(
                      novel.originalTitle!,
                      style: TextStyle(
                        fontSize: 12,
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  const SizedBox(height: 6),
                  Text(
                    [
                      if (novel.author?.isNotEmpty == true) '作者：${novel.author}',
                      if (novel.illustrator?.isNotEmpty == true)
                        '插画：${novel.illustrator}',
                      if (novel.publisher?.isNotEmpty == true)
                        '出版社：${novel.publisher}',
                      if (novel.label?.isNotEmpty == true) '文库：${novel.label}',
                    ].join('\n'),
                    style: TextStyle(
                      fontSize: 12,
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  const SizedBox(height: 6),
                  Text(
                    [
                      '出版：${formatDate(novel.firstReleaseDate)}',
                      domainLabel(
                        novelReleaseStatusOptions,
                        domainValueFromSlug(
                            novelReleaseStatusOptions, novel.releaseStatus),
                      ),
                      ageRatingLabel(novel.ageRating),
                    ].join(' · '),
                    style: TextStyle(
                      fontSize: 12,
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  if (novel.tags.isNotEmpty)
                    Padding(
                      padding: const EdgeInsets.only(top: 8),
                      child: TagChips(
                        names: novel.tags
                            .map((tag) => tag.name ?? '')
                            .where((name) => name.isNotEmpty)
                            .toList(),
                        max: 6,
                      ),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildVolumeTile(VolumeSummary volume) {
    return Card(
      child: ListTile(
        leading: AppImage(
          url: volume.coverUrl,
          width: 40,
          height: 56,
          borderRadius: BorderRadius.circular(4),
        ),
        title: Text(
          volume.volumeNumber != null
              ? '第 ${volume.volumeNumber} 卷 · ${volume.title ?? ''}'
              : (volume.title ?? ''),
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(fontSize: 14),
        ),
        subtitle: Text(
          '出版：${formatDate(volume.releaseDate)}',
          style: const TextStyle(fontSize: 12),
        ),
        trailing: const Icon(Icons.chevron_right),
        onTap: volume.id == null
            ? null
            : () => context
                .push('/novels/${widget.id}/volumes/${volume.id}/edit')
                .then((_) => _load()),
      ),
    );
  }

  Widget _buildRelatedGalgame(RelatedWorkData work) {
    return Card(
      child: ListTile(
        leading: AppImage(
          url: work.coverUrl,
          width: 44,
          height: 60,
          borderRadius: BorderRadius.circular(6),
        ),
        title: Text(work.title ?? '',
            maxLines: 1, overflow: TextOverflow.ellipsis),
        subtitle: work.relationType != null
            ? Text(
                domainLabel(
                    novelRelationTypeOptions,
                    domainValueFromSlug(
                        novelRelationTypeOptions, work.relationType)),
                style: const TextStyle(fontSize: 12),
              )
            : null,
        onTap: work.workId == null ? null : () => context.push('/galgames/${work.workId}'),
      ),
    );
  }
}

class _SectionTitle extends StatelessWidget {
  const _SectionTitle({required this.title, this.action});

  final String title;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        children: [
          Expanded(
            child: Text(
              title,
              style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
            ),
          ),
          ?action,
        ],
      ),
    );
  }
}
