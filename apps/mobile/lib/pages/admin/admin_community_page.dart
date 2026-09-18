import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/admin_models.dart';
import '../../models/pagination.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';

class AdminCommunityPage extends ConsumerWidget {
  const AdminCommunityPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return ref
        .watch(mePermissionsProvider)
        .when(
          loading: () => Scaffold(
            appBar: AppBar(title: const Text('社区管理')),
            body: const LoadingView(),
          ),
          error: (error, _) => Scaffold(
            appBar: AppBar(title: const Text('社区管理')),
            body: ErrorView(
              message: apiErrorMessage(error),
              onRetry: () => ref.invalidate(mePermissionsProvider),
            ),
          ),
          data: (permissions) {
            final tabs = <(String, Widget)>[];
            if (permissions.has('post:moderate')) {
              tabs.add((
                '帖子',
                _AdminPagedList<AdminPostData>(
                  fetch: (page, keyword) => ref
                      .read(adminServiceProvider)
                      .listPosts(page: page, keyword: keyword),
                  itemBuilder: (context, post) => _PostCard(
                    post: post,
                    showIP: permissions.has('ip_audit:read'),
                  ),
                  emptyHint: '暂无帖子',
                ),
              ));
            }
            if (permissions.has('comment:moderate')) {
              tabs.add((
                '评论',
                _AdminPagedList<AdminCommentData>(
                  fetch: (page, keyword) => ref
                      .read(adminServiceProvider)
                      .listComments(page: page, keyword: keyword),
                  itemBuilder: (context, comment) => _CommentCard(
                    comment: comment,
                    showIP: permissions.has('ip_audit:read'),
                  ),
                  emptyHint: '暂无评论',
                ),
              ));
            }

            if (tabs.isEmpty) {
              return Scaffold(
                appBar: AppBar(title: const Text('社区管理')),
                body: const EmptyView(hint: '无权访问社区管理'),
              );
            }
            if (tabs.length == 1) {
              return Scaffold(
                appBar: AppBar(title: const Text('社区管理')),
                body: tabs.single.$2,
              );
            }
            return DefaultTabController(
              length: tabs.length,
              child: Scaffold(
                appBar: AppBar(
                  title: const Text('社区管理'),
                  bottom: TabBar(
                    tabs: [for (final tab in tabs) Tab(text: tab.$1)],
                  ),
                ),
                body: TabBarView(children: [for (final tab in tabs) tab.$2]),
              ),
            );
          },
        );
  }
}

class _AdminPagedList<T> extends StatefulWidget {
  const _AdminPagedList({
    required this.fetch,
    required this.itemBuilder,
    required this.emptyHint,
  });

  final Future<Paginated<T>> Function(int page, String? keyword) fetch;
  final Widget Function(BuildContext context, T item) itemBuilder;
  final String emptyHint;

  @override
  State<_AdminPagedList<T>> createState() => _AdminPagedListState<T>();
}

class _AdminPagedListState<T> extends State<_AdminPagedList<T>> {
  final _searchController = TextEditingController();
  final _scrollController = ScrollController();
  List<T> _items = [];
  int _page = 1;
  bool _loading = false;
  bool _hasMore = true;
  String? _error;
  int _requestId = 0;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
    _load(reset: true);
  }

  @override
  void dispose() {
    _searchController.dispose();
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
    final requestId = ++_requestId;
    final requestPage = reset ? 1 : _page;
    setState(() {
      _loading = true;
      if (reset) {
        _error = null;
      }
    });
    try {
      final keyword = _searchController.text.trim();
      final result = await widget.fetch(
        requestPage,
        keyword.isEmpty ? null : keyword,
      );
      if (!mounted || requestId != _requestId) {
        return;
      }
      setState(() {
        _items = reset ? result.items : [..._items, ...result.items];
        _hasMore = result.items.isNotEmpty && result.hasMore;
        _page = requestPage + 1;
        _loading = false;
        _error = null;
      });
    } catch (error) {
      if (!mounted || requestId != _requestId) {
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
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
          child: TextField(
            controller: _searchController,
            textInputAction: TextInputAction.search,
            decoration: InputDecoration(
              hintText: '搜索关键词',
              prefixIcon: const Icon(Icons.search),
              suffixIcon: IconButton(
                icon: const Icon(Icons.arrow_forward),
                onPressed: () => _load(reset: true),
              ),
            ),
            onSubmitted: (_) => _load(reset: true),
          ),
        ),
        Expanded(child: _buildBody()),
      ],
    );
  }

  Widget _buildBody() {
    if (_error != null && _items.isEmpty) {
      return ErrorView(message: _error!, onRetry: () => _load(reset: true));
    }
    if (_items.isEmpty && _loading) {
      return const LoadingView();
    }
    if (_items.isEmpty) {
      return EmptyView(hint: widget.emptyHint);
    }
    return RefreshIndicator(
      onRefresh: () => _load(reset: true),
      child: ListView.separated(
        controller: _scrollController,
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(16, 4, 16, 24),
        itemCount: _items.length + (_loading ? 1 : 0),
        separatorBuilder: (_, _) => const SizedBox(height: 10),
        itemBuilder: (context, index) {
          if (index >= _items.length) {
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
          return widget.itemBuilder(context, _items[index]);
        },
      ),
    );
  }
}

class _PostCard extends StatelessWidget {
  const _PostCard({required this.post, required this.showIP});

  final AdminPostData post;
  final bool showIP;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: post.id == null ? null : () => context.push('/posts/${post.id}'),
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                post.title ?? '',
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.w600,
                ),
              ),
              const SizedBox(height: 6),
              Text(
                post.content ?? '',
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontSize: 13),
              ),
              const SizedBox(height: 8),
              Text(
                '${post.authorName ?? '已注销用户'} · ${formatDateTime(post.createdAt)}',
                style: TextStyle(
                  fontSize: 12,
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
              ),
              if (showIP &&
                  (post.ip?.isNotEmpty == true ||
                      post.ipRegion?.isNotEmpty == true)) ...[
                const SizedBox(height: 4),
                _IPText(ip: post.ip, region: post.ipRegion),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _CommentCard extends StatelessWidget {
  const _CommentCard({required this.comment, required this.showIP});

  final AdminCommentData comment;
  final bool showIP;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: comment.postId == null
            ? null
            : () => context.push('/posts/${comment.postId}'),
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                comment.content ?? '',
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontSize: 14),
              ),
              const SizedBox(height: 8),
              Text(
                '${comment.authorName ?? '已注销用户'} · ${comment.postTitle ?? ''}',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  fontSize: 12,
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
              ),
              Text(
                formatDateTime(comment.createdAt),
                style: TextStyle(
                  fontSize: 12,
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
              ),
              if (showIP &&
                  (comment.ip?.isNotEmpty == true ||
                      comment.ipRegion?.isNotEmpty == true)) ...[
                const SizedBox(height: 4),
                _IPText(ip: comment.ip, region: comment.ipRegion),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _IPText extends StatelessWidget {
  const _IPText({this.ip, this.region});

  final String? ip;
  final String? region;

  @override
  Widget build(BuildContext context) {
    return Text(
      [
        if (ip?.isNotEmpty == true) 'IP：$ip',
        if (region?.isNotEmpty == true) '属地：$region',
      ].join(' · '),
      style: TextStyle(
        fontSize: 12,
        color: Theme.of(context).colorScheme.primary,
      ),
    );
  }
}
