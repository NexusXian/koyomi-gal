import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/article_models.dart';
import '../../providers/app_providers.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';
import '../../widgets/user_widgets.dart';

class ArticleListPage extends ConsumerStatefulWidget {
  const ArticleListPage({super.key, this.type});

  final String? type;

  @override
  ConsumerState<ArticleListPage> createState() => _ArticleListPageState();
}

class _ArticleListPageState extends ConsumerState<ArticleListPage> {
  final _scrollController = ScrollController();

  List<ArticleListItem> _items = [];
  int _page = 1;
  bool _loading = false;
  bool _hasMore = true;
  String? _error;
  String? _type;
  int _loadGeneration = 0;

  static const _types = [
    (null, '全部'),
    ('announcement', '公告'),
    ('news', '新闻'),
    ('event', '活动'),
    ('update', '更新'),
  ];

  @override
  void initState() {
    super.initState();
    _type = widget.type;
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
    if (_loading && !reset) {
      return;
    }
    final generation = reset ? ++_loadGeneration : _loadGeneration;
    final requestedType = _type;
    final requestedPage = reset ? 1 : _page;
    setState(() {
      _loading = true;
      if (reset) {
        _items = [];
        _page = 1;
        _hasMore = true;
        _error = null;
      }
    });
    try {
      final result = await ref
          .read(articleServiceProvider)
          .list(type: requestedType, page: requestedPage, limit: 20);
      if (!mounted || generation != _loadGeneration) {
        return;
      }
      setState(() {
        _items = reset ? result.items : [..._items, ...result.items];
        _hasMore = result.hasMore;
        _page = result.page + 1;
        _loading = false;
      });
    } catch (error) {
      if (!mounted || generation != _loadGeneration) {
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
    final initialTabIndex = _types.indexWhere((type) => type.$1 == _type);
    return DefaultTabController(
      length: _types.length,
      initialIndex: initialTabIndex < 0 ? 0 : initialTabIndex,
      child: Scaffold(
        appBar: AppBar(title: const Text('资讯')),
        bottomNavigationBar: Material(
          color: Theme.of(context).cardColor,
          child: TabBar(
            onTap: (index) {
              setState(() => _type = _types[index].$1);
              _load(reset: true);
            },
            tabs: [for (final type in _types) Tab(text: type.$2)],
          ),
        ),
        body: _buildList(),
      ),
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
      return const EmptyView(hint: '暂无资讯');
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
          final article = _items[index];
          return Card(
            child: InkWell(
              borderRadius: BorderRadius.circular(12),
              onTap: article.id == null
                  ? null
                  : () => context.push('/articles/${article.id}'),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  if (article.coverUrl?.isNotEmpty == true)
                    AppImage(
                      url: article.coverUrl,
                      height: 150,
                      borderRadius: const BorderRadius.vertical(
                        top: Radius.circular(12),
                      ),
                    ),
                  Padding(
                    padding: const EdgeInsets.all(12),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            if (article.isPinned)
                              const Padding(
                                padding: EdgeInsets.only(right: 6),
                                child: Icon(
                                  Icons.push_pin,
                                  size: 14,
                                  color: Colors.orange,
                                ),
                              ),
                            Expanded(
                              child: Text(
                                article.title ?? '',
                                maxLines: 2,
                                overflow: TextOverflow.ellipsis,
                                style: const TextStyle(
                                  fontSize: 15,
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                            ),
                          ],
                        ),
                        if (article.summary?.isNotEmpty == true)
                          Padding(
                            padding: const EdgeInsets.only(top: 4),
                            child: Text(
                              article.summary!,
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                fontSize: 13,
                                color: Theme.of(context)
                                    .colorScheme
                                    .onSurfaceVariant,
                              ),
                            ),
                          ),
                        const SizedBox(height: 8),
                        Row(
                          children: [
                            Text(
                              article.type ?? '',
                              style: TextStyle(
                                fontSize: 12,
                                color: Theme.of(context).colorScheme.primary,
                              ),
                            ),
                            const Spacer(),
                            StatRow(
                              icon: Icons.visibility_outlined,
                              label: formatCount(article.viewCount),
                            ),
                            const SizedBox(width: 10),
                            Text(
                              formatRelative(article.publishedAt),
                              style: TextStyle(
                                fontSize: 12,
                                color: Theme.of(context)
                                    .colorScheme
                                    .onSurfaceVariant,
                              ),
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}
