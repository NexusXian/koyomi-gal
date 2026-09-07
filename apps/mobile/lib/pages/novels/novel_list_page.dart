import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/constants/domain.dart';
import '../../models/novel_models.dart';
import '../../providers/app_providers.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';
import '../../widgets/user_widgets.dart';

class NovelListPage extends ConsumerStatefulWidget {
  const NovelListPage({super.key});

  @override
  ConsumerState<NovelListPage> createState() => _NovelListPageState();
}

class _NovelListPageState extends ConsumerState<NovelListPage> {
  final _scrollController = ScrollController();
  final _searchController = TextEditingController();

  List<NovelListItem> _items = [];
  int _page = 1;
  bool _loading = false;
  bool _hasMore = true;
  String? _error;

  String _keyword = '';
  int _sortIndex = 0;
  String? _releaseStatus;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
    _load(reset: true);
  }

  @override
  void dispose() {
    _scrollController.dispose();
    _searchController.dispose();
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
    setState(() {
      _loading = true;
      if (reset) {
        _error = null;
      }
    });
    final requestPage = reset ? 1 : _page;
    try {
      final result = await ref
          .read(novelServiceProvider)
          .list(
            keyword: _keyword.isEmpty ? null : _keyword,
            sort: domainSlug(novelSortOptions, _sortIndex),
            releaseStatus: _releaseStatus,
            page: requestPage,
            limit: 20,
          );
      if (!mounted) {
        return;
      }
      setState(() {
        _items = reset ? result.items : [..._items, ...result.items];
        _hasMore = result.hasMore;
        _page = requestPage + 1;
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
    final canCreate = ref
        .watch(mePermissionsProvider)
        .maybeWhen(
          data: (permissions) => permissions.has('novel:create'),
          orElse: () => false,
        );
    return Scaffold(
      appBar: AppBar(
        title: const Text('小说'),
        actions: [
          PopupMenuButton<String>(
            icon: const Icon(Icons.sort),
            onSelected: (value) {
              setState(() {
                if (value.startsWith('sort:')) {
                  _sortIndex = int.parse(value.substring(5));
                } else if (value == 'all') {
                  _releaseStatus = null;
                } else {
                  _releaseStatus = value;
                }
              });
              _load(reset: true);
            },
            itemBuilder: (context) => [
              const PopupMenuItem(value: 'sort:0', child: Text('排序：最近更新')),
              const PopupMenuItem(value: 'sort:1', child: Text('排序：最新收录')),
              const PopupMenuItem(value: 'sort:2', child: Text('排序：最早出版')),
              const PopupMenuItem(value: 'sort:3', child: Text('排序：最新出版')),
              const PopupMenuDivider(),
              const PopupMenuItem(value: 'all', child: Text('连载状态：全部')),
              for (final option in novelReleaseStatusOptions)
                PopupMenuItem(
                  value: option.slug,
                  child: Text('连载状态：${option.label}'),
                ),
            ],
          ),
        ],
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(56),
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
            child: TextField(
              controller: _searchController,
              decoration: InputDecoration(
                isDense: true,
                hintText: '搜索标题 / 原文标题 / 作者',
                prefixIcon: const Icon(Icons.search, size: 20),
                suffixIcon: IconButton(
                  icon: const Icon(Icons.search, size: 18),
                  onPressed: () {
                    _keyword = _searchController.text.trim();
                    _load(reset: true);
                  },
                ),
              ),
              onSubmitted: (value) {
                _keyword = value.trim();
                _load(reset: true);
              },
            ),
          ),
        ),
      ),
      floatingActionButton: canCreate
          ? FloatingActionButton(
              heroTag: 'novel-create',
              onPressed: () => context.push('/novels/new').then((result) {
                if (result == true) {
                  _load(reset: true);
                }
              }),
        child: const Icon(Icons.add),
            )
          : null,
      body: _buildList(),
    );
  }

  Widget _buildList() {
    if (_error != null && _items.isEmpty) {
      return ErrorView(message: _error!, onRetry: () => _load(reset: true));
    }
    if (_items.isEmpty && _loading) {
      return const LoadingView();
    }
    if (_items.isEmpty) {
      return const EmptyView(hint: '没有找到小说');
    }
    return RefreshIndicator(
      onRefresh: () => _load(reset: true),
      child: ListView.separated(
        controller: _scrollController,
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        itemCount: _items.length + 1,
        separatorBuilder: (_, _) => const SizedBox(height: 10),
        itemBuilder: (context, index) {
          if (index >= _items.length) {
            if (_loading) {
              return const Padding(
                padding: EdgeInsets.all(12),
                child: Center(
                  child: SizedBox(
                    width: 20,
                    height: 20,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  ),
                ),
              );
            }
            return const SizedBox.shrink();
          }
          final novel = _items[index];
          return Card(
            child: InkWell(
              borderRadius: BorderRadius.circular(12),
              onTap: novel.id == null
                  ? null
                  : () => context.push('/novels/${novel.id}').then((result) {
                      if (result == true) {
                        _load(reset: true);
                      }
                    }),
              child: Padding(
                padding: const EdgeInsets.all(10),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    AppImage(
                      url: novel.coverUrl,
                      sensitive: novel.isCoverSensitive,
                      width: 76,
                      height: 106,
                      borderRadius: BorderRadius.circular(8),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            novel.title ?? '',
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                              fontSize: 15,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                          if (novel.originalTitle?.isNotEmpty == true)
                            Text(
                              novel.originalTitle!,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                fontSize: 12,
                                color: Theme.of(context)
                                    .colorScheme
                                    .onSurfaceVariant,
                              ),
                            ),
                          const SizedBox(height: 4),
                          Text(
                            [
                              if (novel.author?.isNotEmpty == true)
                                '作者：${novel.author}',
                              if (novel.publisher?.isNotEmpty == true)
                                novel.publisher!,
                            ].join(' · '),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              fontSize: 12,
                              color: Theme.of(context)
                                  .colorScheme
                                  .onSurfaceVariant,
                            ),
                          ),
                          const SizedBox(height: 4),
                          Row(
                            children: [
                              if (novel.releaseStatus != null)
                                Text(
                                  domainLabel(
                                    novelReleaseStatusOptions,
                                    domainValueFromSlug(
                                        novelReleaseStatusOptions,
                                      novel.releaseStatus,
                                    ),
                                  ),
                                  style: TextStyle(
                                    fontSize: 12,
                                    color: Theme.of(context)
                                        .colorScheme
                                        .primary,
                                  ),
                                ),
                              const SizedBox(width: 10),
                              Text(
                                '${novel.statistics?.volumeCount ?? 0} 卷',
                                style: TextStyle(
                                  fontSize: 12,
                                  color: Theme.of(context)
                                      .colorScheme
                                      .onSurfaceVariant,
                                ),
                              ),
                            ],
                          ),
                          if (novel.tags.isNotEmpty)
                            Padding(
                              padding: const EdgeInsets.only(top: 6),
                              child: TagChips(
                                names: novel.tags
                                    .map((tag) => tag.name ?? '')
                                    .where((name) => name.isNotEmpty)
                                    .toList(),
                              ),
                            ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }
}
