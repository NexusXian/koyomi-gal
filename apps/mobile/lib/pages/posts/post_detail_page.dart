import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/post_models.dart';
import '../../models/user_models.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';
import '../../widgets/markdown_view.dart';
import '../../widgets/user_widgets.dart';

class PostDetailPage extends ConsumerStatefulWidget {
  const PostDetailPage({super.key, required this.id});

  final int id;

  @override
  ConsumerState<PostDetailPage> createState() => _PostDetailPageState();
}

class _PostDetailPageState extends ConsumerState<PostDetailPage> {
  PostData? _post;
  String? _error;
  bool _loading = true;

  bool _liked = false;
  bool _favorited = false;
  int _likeCount = 0;

  final _commentController = TextEditingController();
  final _commentFocus = FocusNode();
  int? _replyParentId;
  CommunityUserSummary? _replyTo;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _commentController.dispose();
    _commentFocus.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final post = await ref.read(postServiceProvider).get(widget.id);
      if (!mounted) {
        return;
      }
      setState(() {
        _post = post;
        _likeCount = post.likeCount;
        _loading = false;
      });
      _loadInteraction();
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

  Future<void> _loadInteraction() async {
    final auth = ref.read(authControllerProvider);
    if (!auth.isAuthenticated) {
      return;
    }
    // The list API returns liked/favorited state for the post in lists only,
    // so approximate by silently ignoring failures on the detail endpoint.
    setState(() {
      _liked = false;
      _favorited = false;
    });
  }

  Future<void> _toggleLike() async {
    final auth = ref.read(authControllerProvider);
    if (!auth.isAuthenticated) {
      context.push('/login');
      return;
    }
    setState(() {
      _liked = !_liked;
      _likeCount += _liked ? 1 : -1;
    });
    try {
      final result = _liked
          ? await ref.read(postServiceProvider).like(widget.id)
          : await ref.read(postServiceProvider).unlike(widget.id);
      if (mounted) {
        setState(() {
          _liked = result.liked;
          _likeCount = result.likeCount;
        });
      }
    } catch (error) {
      if (mounted) {
        setState(() {
          _liked = !_liked;
          _likeCount += _liked ? 1 : -1;
        });
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    }
  }

  Future<void> _toggleFavorite() async {
    final auth = ref.read(authControllerProvider);
    if (!auth.isAuthenticated) {
      context.push('/login');
      return;
    }
    final target = !_favorited;
    try {
      final result = target
          ? await ref.read(postServiceProvider).favorite(widget.id)
          : await ref.read(postServiceProvider).unfavorite(widget.id);
      if (mounted) {
        setState(() => _favorited = result.favorited);
      }
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    }
  }

  Future<void> _deletePost() async {
    final confirmed = await showConfirmDialog(
      context,
      title: '删除帖子',
      content: '删除后无法恢复，确定删除吗？',
      confirmText: '删除',
      danger: true,
    );
    if (!confirmed) {
      return;
    }
    try {
      await ref.read(postServiceProvider).delete(widget.id);
      if (mounted) {
        showAppSnackBar(context, '已删除');
        context.pop(true);
      }
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    }
  }

  void _startReply(CommentData comment) {
    setState(() {
      _replyParentId = comment.parentId ?? comment.id;
      _replyTo = comment.author;
    });
    _commentFocus.requestFocus();
  }

  void _cancelReply() {
    setState(() {
      _replyParentId = null;
      _replyTo = null;
    });
    _commentController.clear();
  }

  Future<void> _submitComment() async {
    final content = _commentController.text.trim();
    if (content.isEmpty) {
      return;
    }
    final parent = _replyParentId;
    setState(() => _replyParentId = null);
    try {
      await ref.read(commentServiceProvider).create(
            widget.id,
            content: content,
            parentId: parent,
          );
      _commentController.clear();
      _replyTo = null;
      if (mounted) {
        showAppSnackBar(context, '评论成功');
      }
    } catch (error) {
      if (mounted) {
        setState(() => _replyParentId = parent);
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        actions: [
          if (_post != null)
            IconButton(
              icon: const Icon(Icons.delete_outline),
              tooltip: '删除',
              onPressed: _deletePost,
            ),
        ],
      ),
      body: _buildBody(),
      bottomNavigationBar: _post == null ? null : _buildCommentBar(),
    );
  }

  Widget _buildBody() {
    if (_loading) {
      return const LoadingView();
    }
    if (_error != null) {
      return ErrorView(message: _error!, onRetry: _load);
    }
    final post = _post!;
    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
      children: [
        Text(
          post.title ?? '',
          style: const TextStyle(fontSize: 19, fontWeight: FontWeight.w700),
        ),
        const SizedBox(height: 8),
        Row(
          children: [
            UserAvatar(
              url: post.author?.avatarUrl ?? post.authorAvatar,
              size: 32,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Text(
                        post.author?.displayName ??
                            post.authorName ??
                            post.author?.username ??
                            '',
                        style: const TextStyle(
                            fontSize: 13, fontWeight: FontWeight.w500),
                      ),
                      const SizedBox(width: 6),
                      LevelBadge(level: post.author?.level),
                    ],
                  ),
                  Text(
                    formatDateTime(post.createdAt),
                    style: TextStyle(
                      fontSize: 11,
                      color: Theme.of(context).colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
            if (post.galgameId != null)
              TextButton(
                onPressed: () => context.push('/galgames/${post.galgameId}'),
                child: Text(
                  post.galgameTitle ?? '查看 Galgame',
                  style: const TextStyle(fontSize: 12),
                ),
              ),
          ],
        ),
        const Divider(height: 20),
        if (post.editorMode == 'markdown' && post.content != null)
          MarkdownView(data: post.content!)
        else
          Text(
            post.content ?? '',
            style: const TextStyle(fontSize: 15, height: 1.65),
          ),
        const SizedBox(height: 20),
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            OutlinedButton.icon(
              onPressed: _toggleLike,
              icon: Icon(
                _liked ? Icons.favorite : Icons.favorite_border,
                size: 16,
                color: _liked ? Theme.of(context).colorScheme.error : null,
              ),
              label: Text('$_likeCount'),
            ),
            const SizedBox(width: 12),
            OutlinedButton.icon(
              onPressed: _toggleFavorite,
              icon: Icon(
                _favorited ? Icons.bookmark : Icons.bookmark_border,
                size: 16,
                color: _favorited ? Theme.of(context).colorScheme.primary : null,
              ),
              label: Text(_favorited ? '已收藏' : '收藏'),
            ),
          ],
        ),
        const SizedBox(height: 16),
        Row(
          children: [
            Text(
              '评论 ${_post?.commentCount ?? 0}',
              style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
            ),
            const Spacer(),
            IconButton(
              icon: const Icon(Icons.edit_outlined, size: 18),
              tooltip: '编辑帖子',
              onPressed: () => context.push('/posts/${widget.id}/edit'),
            ),
          ],
        ),
        _CommentsSection(
          postId: widget.id,
          onReply: _startReply,
          onChanged: _load,
        ),
      ],
    );
  }

  Widget _buildCommentBar() {
    final replying = _replyParentId != null;
    final replyToName = _replyTo?.displayName ?? _replyTo?.username;
    return SafeArea(
      child: Container(
        padding: EdgeInsets.only(
          left: 16,
          right: 16,
          top: 8,
          bottom: MediaQuery.of(context).viewInsets.bottom > 0 ? 8 : 12,
        ),
        decoration: BoxDecoration(
          color: Theme.of(context).cardColor,
          border: Border(
            top: BorderSide(color: Theme.of(context).colorScheme.outlineVariant),
          ),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (replying)
              Padding(
                padding: const EdgeInsets.only(bottom: 6),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        '回复 $replyToName',
                        style: TextStyle(
                          fontSize: 12,
                          color: Theme.of(context).colorScheme.primary,
                        ),
                      ),
                    ),
                    GestureDetector(
                      onTap: _cancelReply,
                      child: const Icon(Icons.close, size: 16),
                    ),
                  ],
                ),
              ),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _commentController,
                    focusNode: _commentFocus,
                    minLines: 1,
                    maxLines: 4,
                    decoration: InputDecoration(
                      isDense: true,
                      hintText: replying ? '回复 $replyToName' : '写下你的评论…',
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                IconButton.filled(
                  onPressed: _submitComment,
                  icon: const Icon(Icons.send, size: 18),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _CommentsSection extends ConsumerStatefulWidget {
  const _CommentsSection({
    required this.postId,
    required this.onReply,
    required this.onChanged,
  });

  final int postId;
  final ValueChanged<CommentData> onReply;
  final VoidCallback onChanged;

  @override
  ConsumerState<_CommentsSection> createState() => _CommentsSectionState();
}

class _CommentsSectionState extends ConsumerState<_CommentsSection> {
  List<CommentData> _comments = [];
  bool _loading = true;
  String? _error;

  final Map<int, bool> _likedMap = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final result = await ref
          .read(commentServiceProvider)
          .listByPost(widget.postId, limit: 100);
      if (!mounted) {
        return;
      }
      setState(() {
        _comments = result.items;
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

  Future<void> _toggleCommentLike(CommentData comment) async {
    final auth = ref.read(authControllerProvider);
    if (!auth.isAuthenticated) {
      context.push('/login');
      return;
    }
    final id = comment.id;
    if (id == null) {
      return;
    }
    final liked = _likedMap[id] ?? false;
    setState(() => _likedMap[id] = !liked);
    try {
      final result = !liked
          ? await ref.read(commentServiceProvider).like(id)
          : await ref.read(commentServiceProvider).unlike(id);
      if (mounted) {
        setState(() => _likedMap[id] = result.liked);
      }
    } catch (error) {
      if (mounted) {
        setState(() => _likedMap[id] = liked);
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    }
  }

  Future<void> _deleteComment(CommentData comment) async {
    final confirmed = await showConfirmDialog(
      context,
      title: '删除评论',
      content: '删除后无法恢复，确定删除吗？',
      confirmText: '删除',
      danger: true,
    );
    if (!confirmed || comment.id == null) {
      return;
    }
    try {
      await ref.read(commentServiceProvider).delete(comment.id!);
      await _load();
      widget.onChanged();
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return const Padding(
        padding: EdgeInsets.all(24),
        child: Center(
          child: SizedBox(
            width: 22,
            height: 22,
            child: CircularProgressIndicator(strokeWidth: 2),
          ),
        ),
      );
    }
    if (_error != null) {
      return ErrorView(message: _error!, onRetry: _load);
    }
    if (_comments.isEmpty) {
      return const Padding(
        padding: EdgeInsets.all(24),
        child: Center(child: Text('暂无评论，来抢沙发吧')),
      );
    }
    return Column(
      children: [
        for (final comment in _comments) _buildCommentTile(comment),
      ],
    );
  }

  Widget _buildCommentTile(CommentData comment) {
    final id = comment.id;
    final liked = id == null ? false : (_likedMap[id] ?? false);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          UserAvatar(url: comment.author?.avatarUrl, size: 34),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Text(
                      comment.author?.displayName ??
                          comment.author?.username ??
                          '',
                      style: const TextStyle(
                          fontSize: 13, fontWeight: FontWeight.w600),
                    ),
                    const SizedBox(width: 6),
                    LevelBadge(level: comment.author?.level),
                    const Spacer(),
                    Text(
                      formatRelative(comment.createdAt),
                      style: TextStyle(
                        fontSize: 11,
                        color: Theme.of(context).colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 4),
                Text(
                  comment.content ?? '',
                  style: const TextStyle(fontSize: 14, height: 1.5),
                ),
                const SizedBox(height: 4),
                Row(
                  children: [
                    GestureDetector(
                      onTap: () => _toggleCommentLike(comment),
                      child: Row(
                        children: [
                          Icon(
                            liked ? Icons.favorite : Icons.favorite_outline,
                            size: 13,
                            color: liked
                                ? Theme.of(context).colorScheme.error
                                : Theme.of(context).colorScheme.onSurfaceVariant,
                          ),
                          const SizedBox(width: 3),
                          Text(
                            '${comment.likeCount}',
                            style: TextStyle(
                              fontSize: 12,
                              color: Theme.of(context)
                                  .colorScheme
                                  .onSurfaceVariant,
                            ),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(width: 14),
                    GestureDetector(
                      onTap: () => widget.onReply(comment),
                      child: Row(
                        children: [
                          Icon(
                            Icons.reply,
                            size: 14,
                            color:
                                Theme.of(context).colorScheme.onSurfaceVariant,
                          ),
                          const SizedBox(width: 3),
                          Text(
                            comment.replyCount > 0
                                ? '回复 ${comment.replyCount}'
                                : '回复',
                            style: TextStyle(
                              fontSize: 12,
                              color: Theme.of(context)
                                  .colorScheme
                                  .onSurfaceVariant,
                            ),
                          ),
                        ],
                      ),
                    ),
                    const Spacer(),
                    GestureDetector(
                      onTap: () => _deleteComment(comment),
                      child: Icon(
                        Icons.delete_outline,
                        size: 14,
                        color: Theme.of(context).colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
