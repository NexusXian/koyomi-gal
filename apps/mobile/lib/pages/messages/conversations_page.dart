import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/message_models.dart';
import '../../providers/app_providers.dart';
import '../../providers/message_providers.dart';
import '../../widgets/common_views.dart';
import '../../widgets/user_widgets.dart';

class ConversationsPage extends ConsumerStatefulWidget {
  const ConversationsPage({super.key});

  @override
  ConsumerState<ConversationsPage> createState() => _ConversationsPageState();
}

class _ConversationsPageState extends ConsumerState<ConversationsPage> {
  final List<Conversation> _items = [];
  final ScrollController _scrollController = ScrollController();
  StreamSubscription<RealtimeEvent>? _events;
  String? _cursor;
  bool _hasMore = false;
  bool _loading = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load(reset: true);
    _scrollController.addListener(_onScroll);
    _events =
        ref.read(messageRealtimeProvider).events.listen(_onRealtimeEvent);
  }

  @override
  void dispose() {
    _events?.cancel();
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.position.pixels >=
            _scrollController.position.maxScrollExtent - 200 &&
        _hasMore &&
        !_loading) {
      _load(reset: false);
    }
  }

  Future<void> _load({required bool reset}) async {
    if (_loading) return;
    setState(() {
      _loading = true;
      if (reset) {
        _error = null;
      }
    });
    try {
      final page = await ref.read(messageServiceProvider).listConversations(
            cursor: reset ? null : _cursor,
          );
      if (!mounted) return;
      setState(() {
        if (reset) {
          _items
            ..clear()
            ..addAll(page.list);
        } else {
          _items.addAll(page.list);
        }
        _cursor = page.nextCursor;
        _hasMore = page.hasMore;
      });
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = apiErrorMessage(error));
    } finally {
      if (mounted) {
        setState(() => _loading = false);
      }
    }
  }

  void _onRealtimeEvent(RealtimeEvent event) {
    if (!mounted) return;
    if (event.type == 'message.created') {
      final data = event.data ?? const <String, dynamic>{};
      final conversationId = (data['conversation_id'] as num?)?.toInt();
      final message = data['message'];
      if (message is! Map) return;
      final senderId = (message['sender_id'] as num?)?.toInt();
      final mine = senderId == ref.read(authControllerProvider).user?.id;
      final index =
          _items.indexWhere((item) => item.id == conversationId);
      if (index >= 0) {
        setState(() {
          final current = _items.removeAt(index);
          final preview = MessagePreview.fromMap(
            Map<String, dynamic>.from(message),
          );
          _items.insert(
            0,
            current.copyWith(
              lastMessage: preview,
              lastMessageAt: preview.createdAt ?? current.lastMessageAt,
              unreadCount:
                  mine ? current.unreadCount : current.unreadCount + 1,
            ),
          );
        });
      } else {
        _load(reset: true);
      }
    } else if (event.type == 'message.deleted') {
      final conversationId =
          ((event.data?['conversation_id'] as num?)?.toInt());
      final messageId = ((event.data?['message_id'] as num?)?.toInt());
      final index = _items.indexWhere((item) => item.id == conversationId);
      if (index >= 0 &&
          _items[index].lastMessage?.id == messageId &&
          mounted) {
        setState(() {
          final current = _items.removeAt(index);
          _items.insert(
            index,
            current.copyWith(
              lastMessage: MessagePreview(
                id: messageId,
                isDeleted: true,
                content: '',
                senderId: current.lastMessage?.senderId,
                createdAt: current.lastMessage?.createdAt,
              ),
            ),
          );
        });
      }
    }
  }

  Future<void> _removeConversation(Conversation conversation) async {
    final confirmed = await showConfirmDialog(
      context,
      title: '删除会话',
      content: '确定从你的会话列表中删除该聊天吗？对方不受影响，收到新消息后会自动恢复。',
      confirmText: '删除',
      danger: true,
    );
    if (!confirmed || !mounted || conversation.id == null) return;
    try {
      await ref
          .read(messageServiceProvider)
          .deleteConversation(conversation.id!);
      if (!mounted) return;
      setState(() => _items.removeWhere((item) => item.id == conversation.id));
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, friendlyMessageError(error), error: true);
      }
    }
  }

  void _open(Conversation conversation) {
    if (conversation.id == null) return;
    ref.read(unreadMessagesProvider).consume(conversation.unreadCount);
    context.push('/messages/${conversation.id}', extra: conversation);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('私信')),
      body: _buildBody(),
    );
  }

  Widget _buildBody() {
    if (_loading && _items.isEmpty) {
      return const LoadingView();
    }
    if (_error != null && _items.isEmpty) {
      return ErrorView(message: _error!, onRetry: () => _load(reset: true));
    }
    if (_items.isEmpty) {
      return const EmptyView(hint: '还没有私信，去看看其他用户吧');
    }
    return RefreshIndicator(
      onRefresh: () => _load(reset: true),
      child: ListView.separated(
        controller: _scrollController,
        physics: const AlwaysScrollableScrollPhysics(),
        itemCount: _items.length + (_hasMore ? 1 : 0),
        separatorBuilder: (_, _) => const Divider(height: 1),
        itemBuilder: (context, index) {
          if (index >= _items.length) {
            return const Padding(
              padding: EdgeInsets.symmetric(vertical: 12),
              child: Center(
                child: SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              ),
            );
          }
          final conversation = _items[index];
          return ListTile(
            leading: TappableUser(
              username: conversation.user?.username,
              child: UserAvatar(
                url: conversation.user?.avatarUrl,
                size: 46,
              ),
            ),
            title: Text(
              conversation.user?.displayName ??
                  conversation.user?.username ??
                  '用户',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontWeight: FontWeight.w600),
            ),
            subtitle: Text(
              conversation.lastMessage == null
                  ? '还没有消息'
                  : conversation.lastMessage!.isDeleted
                      ? '消息已删除'
                      : conversation.lastMessage!.content,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            trailing: _buildTrailing(conversation),
            onLongPress: () => _removeConversation(conversation),
            onTap: () => _open(conversation),
          );
        },
      ),
    );
  }

  Widget _buildTrailing(Conversation conversation) {
    final theme = Theme.of(context);
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        Text(
          formatRelative(conversation.lastMessageAt),
          style: TextStyle(
            fontSize: 11,
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        if (conversation.unreadCount > 0)
          Container(
            margin: const EdgeInsets.only(top: 4),
            padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
            decoration: BoxDecoration(
              color: theme.colorScheme.primary,
              borderRadius: BorderRadius.circular(999),
            ),
            child: Text(
              conversation.unreadCount > 99
                  ? '99+'
                  : '${conversation.unreadCount}',
              style: TextStyle(
                fontSize: 11,
                color: theme.colorScheme.onPrimary,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
      ],
    );
  }
}
