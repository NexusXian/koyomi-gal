import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/post_models.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';
import '../../widgets/user_widgets.dart';

class PostListPage extends ConsumerStatefulWidget {
  const PostListPage({super.key});

  @override
  ConsumerState<PostListPage> createState() => _PostListPageState();
}

class _PostListPageState extends ConsumerState<PostListPage> {
  final _scrollController = ScrollController();

  List<PostData> _items = [];
  int _page = 1;
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
    setState(() {
      _loading = true;
      if (reset) {
        _error = null;
      }
    });
    try {
      final result = await ref.read(postServiceProvider).list(
            page: _page,
            limit: 20,
          );
      if (!mounted) {
        return;
      }
      setState(() {
        _items = reset ? result.items : [..._items, ...result.items];
        _hasMore = result.hasMore;
        _page = reset ? 2 : _page + 1;
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
    return Scaffold(
      appBar: AppBar(title: const Text('帖子')),
      floatingActionButton: FloatingActionButton(
        onPressed: () => context.push('/posts/new'),
        child: const Icon(Icons.edit),
      ),
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
      return const EmptyView(hint: '暂无帖子');
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
          final post = _items[index];
          return Card(
            child: InkWell(
              borderRadius: BorderRadius.circular(12),
              onTap: () => context.push('/posts/${post.id}'),
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        UserAvatar(
                          url: post.author?.avatarUrl ?? post.authorAvatar,
                          size: 28,
                        ),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(
                            post.author?.displayName ??
                                post.authorName ??
                                post.author?.username ??
                                '',
                            style: const TextStyle(fontSize: 13),
                          ),
                        ),
                        Text(
                          formatRelative(post.createdAt),
                          style: TextStyle(
                            fontSize: 12,
                            color:
                                Theme.of(context).colorScheme.onSurfaceVariant,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    Text(
                      post.title ?? '',
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 15,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    if (post.galgameTitle?.isNotEmpty == true)
                      Padding(
                        padding: const EdgeInsets.only(top: 2),
                        child: Text(
                          '📦 ${post.galgameTitle}',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 12,
                            color: Theme.of(context).colorScheme.primary,
                          ),
                        ),
                      ),
                    const SizedBox(height: 8),
                    Row(
                      children: [
                        StatRow(
                          icon: Icons.favorite_outline,
                          label: formatCount(post.likeCount),
                        ),
                        const SizedBox(width: 12),
                        StatRow(
                          icon: Icons.chat_bubble_outline,
                          label: formatCount(post.commentCount),
                        ),
                      ],
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
