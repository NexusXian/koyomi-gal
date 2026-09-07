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

class VolumeDetailPage extends ConsumerStatefulWidget {
  const VolumeDetailPage({
    super.key,
    required this.novelId,
    required this.volumeId,
  });

  final int novelId;
  final int volumeId;

  @override
  ConsumerState<VolumeDetailPage> createState() => _VolumeDetailPageState();
}

class _VolumeDetailPageState extends ConsumerState<VolumeDetailPage> {
  VolumeData? _volume;
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
      final volume = await ref
          .read(novelServiceProvider)
          .getVolume(widget.novelId, widget.volumeId);
      if (mounted) {
        setState(() {
          _volume = volume;
          _loading = false;
        });
      }
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = apiErrorMessage(error);
          _loading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final canUpdate = ref
        .watch(mePermissionsProvider)
        .maybeWhen(
          data: (permissions) => permissions.has('novel:update'),
          orElse: () => false,
        );
    return Scaffold(
      appBar: AppBar(
        title: const Text('卷册详情'),
        actions: [
          if (canUpdate)
            IconButton(
              onPressed: () => context
                  .push(
                    '/novels/${widget.novelId}/volumes/${widget.volumeId}/edit',
                  )
                  .then((result) {
                    if (result == true) {
                      _load();
                    }
                  }),
              icon: const Icon(Icons.edit_outlined),
              tooltip: '编辑',
            ),
        ],
      ),
      body: _buildBody(),
    );
  }

  Widget _buildBody() {
    if (_loading) {
      return const LoadingView();
    }
    if (_error != null) {
      return ErrorView(message: _error!, onRetry: _load);
    }
    final volume = _volume;
    if (volume == null) {
      return const EmptyView(hint: '未找到该卷册');
    }
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                AppImage(
                  url: volume.coverUrl,
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
                        volume.title?.isNotEmpty == true
                            ? volume.title!
                            : '未命名卷',
                        style: const TextStyle(
                          fontSize: 17,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      if (volume.originalTitle?.isNotEmpty == true)
                        Text(
                          volume.originalTitle!,
                          style: TextStyle(
                            fontSize: 12,
                            color: Theme.of(context)
                                .colorScheme
                                .onSurfaceVariant,
                          ),
                        ),
                      const SizedBox(height: 8),
                      Text(
                        [
                          if (volume.volumeNumber != null)
                            '第 ${volume.volumeNumber} 卷',
                          if (volume.releaseDate?.isNotEmpty == true)
                            '出版：${formatDate(volume.releaseDate)}',
                          if (volume.isbn?.isNotEmpty == true)
                            'ISBN：${volume.isbn}',
                          if (volume.status != null)
                            '状态：${domainLabel(galgameStatusOptions, volume.status)}',
                        ].join('\n'),
                        style: TextStyle(
                          fontSize: 12,
                          color: Theme.of(context).colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
        if (volume.summary?.isNotEmpty == true) ...[
          const SizedBox(height: 12),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: MarkdownView(data: volume.summary!, selectable: true),
            ),
          ),
        ],
      ],
    );
  }
}
