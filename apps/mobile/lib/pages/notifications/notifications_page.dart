import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/notification_models.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';
import '../../widgets/user_widgets.dart';

class NotificationsPage extends ConsumerStatefulWidget {
  const NotificationsPage({super.key});

  @override
  ConsumerState<NotificationsPage> createState() => _NotificationsPageState();
}

class _NotificationsPageState extends ConsumerState<NotificationsPage> {
  final _scrollController = ScrollController();

  List<NotificationData> _items = [];
  int _page = 1;
  bool _loading = false;
  bool _hasMore = true;
  String? _error;
  String? _category;

  static const _categories = [
    (null, '全部'),
    ('interaction', '互动'),
    ('review', '审核'),
    ('moderation', '管理'),
    ('system', '系统'),
  ];

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
      final result = await ref
          .read(notificationServiceProvider)
          .list(category: _category, page: _page, limit: 20);
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

  Future<void> _markRead(NotificationData notification) async {
    if (notification.isRead || notification.id == null) {
      return;
    }
    try {
      await ref.read(notificationServiceProvider).markRead(notification.id!);
      setState(() {
        final index = _items.indexOf(notification);
        if (index >= 0) {
          _items[index] = NotificationData(
            id: notification.id,
            title: notification.title,
            content: notification.content,
            type: notification.type,
            category: notification.category,
            entityType: notification.entityType,
            entityId: notification.entityId,
            targetUrl: notification.targetUrl,
            isRead: true,
            readAt: DateTime.now().toIso8601String(),
            actor: notification.actor,
            createdAt: notification.createdAt,
          );
        }
      });
      ref.read(unreadNotificationsProvider).refresh();
    } catch (_) {}
  }

  Future<void> _markAllRead() async {
    try {
      await ref.read(notificationServiceProvider).markAllRead();
      await _load(reset: true);
      ref.read(unreadNotificationsProvider).refresh();
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    }
  }

  void _openTarget(NotificationData notification) {
    _markRead(notification);
    final entityType = notification.entityType;
    final entityId = notification.entityId;
    if (entityId == null) {
      return;
    }
    switch (entityType) {
      case 'post':
        context.push('/posts/$entityId');
      case 'galgame':
        context.push('/galgames/$entityId');
      case 'novel':
        context.push('/novels/$entityId');
      case 'comment':
        if (notification.targetUrl?.contains('post') == true) {
          final match = RegExp(r'/posts/(\d+)')
              .firstMatch(notification.targetUrl!);
          if (match != null) {
            context.push('/posts/${match.group(1)}');
          }
        }
      default:
        break;
    }
  }

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: _categories.length,
      child: Scaffold(
      appBar: AppBar(
        title: const Text('通知'),
        actions: [
          TextButton(
            onPressed: _markAllRead,
            child: const Text('全部已读', style: TextStyle(fontSize: 13)),
          ),
        ],
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(48),
          child: Material(
            color: Theme.of(context).scaffoldBackgroundColor,
            child: TabBar(
              isScrollable: true,
              tabAlignment: TabAlignment.start,
              onTap: (index) {
                setState(() => _category = _categories[index].$1);
                _load(reset: true);
              },
                tabs: [
                  for (final category in _categories) Tab(text: category.$2),
                ],
            ),
          ),
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
      return const EmptyView(hint: '暂无通知');
    }
    return RefreshIndicator(
      onRefresh: () => _load(reset: true),
      child: ListView.separated(
        controller: _scrollController,
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        itemCount: _items.length + 1,
        separatorBuilder: (_, _) => const SizedBox(height: 8),
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
          final notification = _items[index];
          return _buildNotificationTile(notification);
        },
      ),
    );
  }

  Widget _buildNotificationTile(NotificationData notification) {
    final theme = Theme.of(context);
    return Card(
      color: notification.isRead
          ? null
          : theme.colorScheme.primaryContainer.withValues(alpha: 0.25),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () => _openTarget(notification),
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              TappableUser(
                username: notification.actor?.username,
                child: UserAvatar(url: notification.actor?.avatarUrl, size: 36),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            notification.title ?? '',
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              fontSize: 14,
                              fontWeight: notification.isRead
                                  ? FontWeight.w500
                                  : FontWeight.w700,
                            ),
                          ),
                        ),
                        if (!notification.isRead)
                          Container(
                            width: 8,
                            height: 8,
                            decoration: BoxDecoration(
                              color: theme.colorScheme.primary,
                              shape: BoxShape.circle,
                            ),
                          ),
                      ],
                    ),
                    if (notification.content?.isNotEmpty == true)
                      Padding(
                        padding: const EdgeInsets.only(top: 2),
                        child: Text(
                          notification.content!,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(fontSize: 13),
                        ),
                      ),
                    const SizedBox(height: 4),
                    Text(
                      formatRelative(notification.createdAt),
                      style: TextStyle(
                        fontSize: 11,
                        color: theme.colorScheme.onSurfaceVariant,
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
  }
}
