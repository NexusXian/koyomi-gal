import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/message_models.dart';
import '../../models/pagination.dart';
import '../../models/user_models.dart';
import '../../providers/app_providers.dart';
import '../../providers/message_providers.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';
import '../../widgets/user_widgets.dart';

class UserProfilePage extends ConsumerStatefulWidget {
  const UserProfilePage({super.key, required this.username});

  final String username;

  @override
  ConsumerState<UserProfilePage> createState() => _UserProfilePageState();
}

class _UserProfilePageState extends ConsumerState<UserProfilePage> {
  PublicUserProfile? _profile;
  String? _error;
  bool _loading = true;
  bool _messaging = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _startConversation() async {
    final profile = _profile;
    if (profile?.id == null || _messaging) return;
    setState(() => _messaging = true);
    try {
      final conversation = await ref
          .read(messageServiceProvider)
          .createConversation(profile!.id!);
      if (!mounted) return;
      ref.read(unreadMessagesProvider).consume(0);
      context.push('/messages/${conversation.id}', extra: conversation);
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, friendlyMessageError(error), error: true);
      }
    } finally {
      if (mounted) {
        setState(() => _messaging = false);
      }
    }
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final profile =
          await ref.read(userServiceProvider).profile(widget.username);
      if (!mounted) {
        return;
      }
      setState(() {
        _profile = profile;
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
      appBar: AppBar(title: Text(_profile?.displayName ?? widget.username)),
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
    final profile = _profile!;
    final access = profile.access;

    if (access == null || !access.canViewProfile) {
      return const EmptyView(hint: '该用户的资料仅对其本人可见');
    }

    final tabs = <(_TabDef, Widget)>[];
    if (access.canViewPosts) {
      tabs.add((
        _TabDef('帖子', profile.postCount),
        _ProfileListTab<ProfilePostData>(
          username: widget.username,
          fetch: (page) => ref
              .read(userServiceProvider)
              .posts(widget.username, page: page),
        ),
      ));
    }
    if (access.canViewComments) {
      tabs.add((
        _TabDef('评论', profile.commentCount),
        _ProfileListTab<ProfileCommentData>(
          username: widget.username,
          fetch: (page) => ref
              .read(userServiceProvider)
              .comments(widget.username, page: page),
        ),
      ));
    }
    if (access.canViewRatings) {
      tabs.add((
        _TabDef('评分', profile.ratingCount),
        _ProfileListTab<ProfileGalgameData>(
          username: widget.username,
          fetch: (page) =>
              ref.read(userServiceProvider).ratings(widget.username, page: page),
        ),
      ));
    }
    if (access.canViewFavorites) {
      tabs.add((
        _TabDef('收藏', profile.favoriteCount),
        _ProfileListTab<ProfileGalgameData>(
          username: widget.username,
          fetch: (page) => ref
              .read(userServiceProvider)
              .favorites(widget.username, page: page),
        ),
      ));
    }

    return DefaultTabController(
      length: tabs.length,
      child: NestedScrollView(
        headerSliverBuilder: (context, innerBoxIsScrolled) => [
          SliverToBoxAdapter(child: _buildHeader(profile)),
          SliverToBoxAdapter(
            child: TabBar(
              isScrollable: true,
              tabAlignment: TabAlignment.start,
              tabs: [
                for (final tab in tabs)
                  Tab(text: '${tab.$1.label} ${tab.$1.count}'),
              ],
            ),
          ),
        ],
        body: tabs.isEmpty
            ? const EmptyView(hint: '暂无公开内容')
            : TabBarView(
                children: [for (final tab in tabs) tab.$2],
              ),
      ),
    );
  }

  Widget _buildHeader(PublicUserProfile profile) {
    final theme = Theme.of(context);
    return Column(
      children: [
        if (profile.bannerUrl?.isNotEmpty == true)
          AppImage(url: profile.bannerUrl, height: 120)
        else
          Container(
            height: 120,
            decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: [
                  theme.colorScheme.primaryContainer.withValues(alpha: 0.5),
                  theme.colorScheme.secondary.withValues(alpha: 0.25),
                ],
              ),
            ),
          ),
        Transform.translate(
          offset: const Offset(0, -36),
          child: Column(
            children: [
              UserAvatar(url: profile.avatarUrl, size: 76),
              const SizedBox(height: 8),
              Text(
                profile.displayName ?? profile.username ?? '',
                style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w700),
              ),
              Text(
                '@${profile.username}',
                style: TextStyle(
                  fontSize: 12,
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 6),
              LevelBadge(level: profile.level),
              if (profile.bio?.isNotEmpty == true)
                Padding(
                  padding: const EdgeInsets.fromLTRB(32, 10, 32, 0),
                  child: Text(
                    profile.bio!,
                    textAlign: TextAlign.center,
                    style: const TextStyle(fontSize: 13),
                  ),
                ),
              const SizedBox(height: 10),
              Wrap(
                spacing: 10,
                runSpacing: 4,
                alignment: WrapAlignment.center,
                children: [
                  if (profile.gender != null &&
                      profile.gender!.isNotEmpty &&
                      profile.gender != 'undisclosed')
                    _MetaChip(icon: Icons.wc_outlined, label: profile.gender!),
                  if (profile.location?.isNotEmpty == true)
                    _MetaChip(
                        icon: Icons.place_outlined, label: profile.location!),
                  _MetaChip(
                    icon: Icons.cake_outlined,
                    label: '加入于 ${formatDate(profile.registeredAt)}',
                  ),
                ],
              ),
              if (profile.websiteUrl?.isNotEmpty == true)
                Padding(
                  padding: const EdgeInsets.only(top: 6),
                  child: Text(
                    profile.websiteUrl!,
                    style: TextStyle(
                      fontSize: 12,
                      color: theme.colorScheme.primary,
                    ),
                  ),
                ),
              const SizedBox(height: 12),
              _buildMessageAction(profile),
              const SizedBox(height: 12),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildMessageAction(PublicUserProfile profile) {
    final auth = ref.watch(authControllerProvider.select(
      (controller) => controller.status,
    ));
    if (auth != AuthStatus.authenticated || profile.isSelf || profile.id == null) {
      return const SizedBox.shrink();
    }
    return OutlinedButton.icon(
      onPressed: _messaging ? null : _startConversation,
      icon: _messaging
          ? const SizedBox(
              width: 14,
              height: 14,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : const Icon(Icons.mail_outline, size: 18),
      label: Text(_messaging ? '正在打开...' : '发私信'),
    );
  }
}

class _TabDef {
  const _TabDef(this.label, this.count);

  final String label;
  final int count;
}

class _MetaChip extends StatelessWidget {
  const _MetaChip({required this.icon, required this.label});

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 13, color: theme.colorScheme.onSurfaceVariant),
        const SizedBox(width: 3),
        Text(
          label,
          style: TextStyle(
            fontSize: 12,
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}

class _ProfileListTab<T> extends ConsumerStatefulWidget {
  const _ProfileListTab({
    required this.username,
    required this.fetch,
  });

  final String username;
  final Future<Paginated<T>> Function(int page) fetch;

  @override
  ConsumerState<_ProfileListTab<T>> createState() => _ProfileListTabState<T>();
}

class _ProfileListTabState<T> extends ConsumerState<_ProfileListTab<T>> {
  final _scrollController = ScrollController();
  List<T> _items = [];
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
      final result = await widget.fetch(_page);
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
    if (_error != null && _items.isEmpty) {
      return ErrorView(message: _error!, onRetry: () => _load(reset: true));
    }
    if (_items.isEmpty && _loading) {
      return const LoadingView();
    }
    if (_items.isEmpty) {
      return const EmptyView();
    }
    return ListView.separated(
      controller: _scrollController,
      padding: const EdgeInsets.all(16),
      itemCount: _items.length + 1,
      separatorBuilder: (_, _) => const SizedBox(height: 10),
      itemBuilder: (context, index) {
        if (index >= _items.length) {
          return _loading
              ? const Padding(
                  padding: EdgeInsets.all(12),
                  child: Center(
                    child: SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                  ),
                )
              : const SizedBox.shrink();
        }
        final item = _items[index];
        return Card(child: _buildTile(context, item));
      },
    );
  }

  Widget _buildTile(BuildContext context, T item) {
    if (item is ProfilePostData) {
      return ListTile(
        title: Text(
          item.title ?? '',
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(fontSize: 14),
        ),
        subtitle: Text(
          '${formatRelative(item.createdAt)} · 👍${item.likeCount} 💬${item.commentCount}',
          style: const TextStyle(fontSize: 12),
        ),
        onTap: item.id == null ? null : () => context.push('/posts/${item.id}'),
      );
    }
    if (item is ProfileCommentData) {
      return ListTile(
        title: Text(
          item.content ?? '',
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(fontSize: 13),
        ),
        subtitle: Text(
          '评论于 ${item.postTitle ?? ''} · ${formatRelative(item.createdAt)}',
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(fontSize: 12),
        ),
        onTap:
            item.postId == null ? null : () => context.push('/posts/${item.postId}'),
      );
    }
    if (item is ProfileGalgameData) {
      return ListTile(
        leading: AppImage(
          url: item.coverUrl,
          sensitive: item.coverSensitive,
          width: 44,
          height: 60,
          borderRadius: BorderRadius.circular(6),
        ),
        title: Text(
          item.title ?? '',
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(fontSize: 14),
        ),
        subtitle: item.score != null
            ? Text('评分 ${item.score}', style: const TextStyle(fontSize: 12))
            : null,
        onTap: item.id == null ? null : () => context.push('/galgames/${item.id}'),
      );
    }
    return const SizedBox.shrink();
  }
}
