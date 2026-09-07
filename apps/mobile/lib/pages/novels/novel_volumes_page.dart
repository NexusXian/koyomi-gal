import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/novel_models.dart';
import '../../providers/app_providers.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';

class NovelVolumesPage extends ConsumerStatefulWidget {
  const NovelVolumesPage({super.key, required this.novelId});

  final int novelId;

  @override
  ConsumerState<NovelVolumesPage> createState() => _NovelVolumesPageState();
}

class _NovelVolumesPageState extends ConsumerState<NovelVolumesPage> {
  final _scrollController = ScrollController();
  List<VolumeData> _items = [];
  int _page = 1;
  int _total = 0;
  bool _loading = false;
  bool _hasMore = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
    _load(reset: true);
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.position.pixels >=
            _scrollController.position.maxScrollExtent - 200 &&
        !_loading &&
        _hasMore) {
      _load();
    }
  }

  Future<void> _load({bool reset = false}) async {
    if (_loading) {
      return;
    }
    final requestPage = reset ? 1 : _page;
    setState(() {
      _loading = true;
      if (reset) {
        _error = null;
      }
    });
    try {
      final result = await ref
          .read(novelServiceProvider)
          .volumes(widget.novelId, page: requestPage, limit: 50);
      if (mounted) {
        setState(() {
          _items = reset ? result.items : [..._items, ...result.items];
          _total = result.total;
          _hasMore = result.hasMore;
          _page = requestPage + 1;
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
        title: Text('全部卷册 ($_total)'),
        actions: [
          if (canUpdate)
            IconButton(
              onPressed: () => context
                  .push('/novels/${widget.novelId}/volumes/new')
                  .then((result) {
                    if (result == true) {
                      _load(reset: true);
                    }
                  }),
              icon: const Icon(Icons.add),
              tooltip: '添加卷册',
            ),
        ],
      ),
      body: _buildBody(canUpdate),
    );
  }

  Widget _buildBody(bool canUpdate) {
    if (_error != null && _items.isEmpty) {
      return ErrorView(message: _error!, onRetry: () => _load(reset: true));
    }
    if (_items.isEmpty && _loading) {
      return const LoadingView();
    }
    if (_items.isEmpty) {
      return const EmptyView(hint: '暂无卷册');
    }
    return RefreshIndicator(
      onRefresh: () => _load(reset: true),
      child: ListView.separated(
        controller: _scrollController,
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(16),
        itemCount: _items.length + (_loading ? 1 : 0),
        separatorBuilder: (_, _) => const SizedBox(height: 8),
        itemBuilder: (context, index) {
          if (index >= _items.length) {
            return const Padding(
              padding: EdgeInsets.all(12),
              child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
            );
          }
          final volume = _items[index];
          return Card(
            child: ListTile(
              leading: AppImage(
                url: volume.coverUrl,
                width: 40,
                height: 56,
                borderRadius: BorderRadius.circular(4),
              ),
              title: Text(
                volume.volumeNumber == null
                    ? (volume.title ?? '未命名卷')
                    : '第 ${volume.volumeNumber} 卷 · ${volume.title ?? ''}',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              subtitle: Text(
                [
                  if (volume.originalTitle?.isNotEmpty == true)
                    volume.originalTitle!,
                  if (volume.releaseDate?.isNotEmpty == true)
                    formatDate(volume.releaseDate),
                  if (volume.isbn?.isNotEmpty == true) 'ISBN ${volume.isbn}',
                ].join(' · '),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
              trailing: canUpdate
                  ? IconButton(
                      onPressed: () => context
                          .push(
                            '/novels/${widget.novelId}/volumes/${volume.id}/edit',
                          )
                          .then((result) {
                            if (result == true) {
                              _load(reset: true);
                            }
                          }),
                      icon: const Icon(Icons.edit_outlined, size: 20),
                    )
                  : const Icon(Icons.chevron_right),
              onTap: volume.id == null
                  ? null
                  : () => context.push(
                      '/novels/${widget.novelId}/volumes/${volume.id}',
                    ),
            ),
          );
        },
      ),
    );
  }
}
