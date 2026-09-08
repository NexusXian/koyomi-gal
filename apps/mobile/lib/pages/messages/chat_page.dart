import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/message_models.dart';
import '../../models/user_models.dart';
import '../../providers/app_providers.dart';
import '../../providers/message_providers.dart';
import '../../widgets/common_views.dart';
import '../../widgets/user_widgets.dart';

class ChatPage extends ConsumerStatefulWidget {
  const ChatPage({super.key, required this.conversationId, this.conversation});

  final int conversationId;
  final Conversation? conversation;

  @override
  ConsumerState<ChatPage> createState() => _ChatPageState();
}

class _ChatPageState extends ConsumerState<ChatPage> {
  final List<Message> _messages = [];
  final ScrollController _scrollController = ScrollController();
  final TextEditingController _inputController = TextEditingController();
  final FocusNode _focusNode = FocusNode();
  StreamSubscription<RealtimeEvent>? _events;

  Conversation? _conversation;
  CommunityUserSummary? _peer;
  bool _sending = false;
  bool _loading = true;
  bool _loadingOlder = false;
  bool _hasMore = false;
  String? _error;
  double _preLoadOffset = 0;
  double _preLoadExtent = 0;

  int? get _myId => ref.read(authControllerProvider).user?.id;

  @override
  void initState() {
    super.initState();
    _conversation = widget.conversation;
    _peer = _conversation?.user;
    _scrollController.addListener(_onScroll);
    _focusNode.addListener(_onFocusChanged);
    _events =
        ref.read(messageRealtimeProvider).events.listen(_onRealtimeEvent);
    _load(reset: true);
  }

  @override
  void dispose() {
    _events?.cancel();
    _focusNode.dispose();
    _inputController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.position.pixels <= 80 &&
        _hasMore &&
        !_loadingOlder &&
        !_loading) {
      _loadOlder();
    }
  }

  void _onFocusChanged() {
    if (_focusNode.hasFocus && _isNearBottom()) {
      _scrollToBottom();
    }
  }

  Future<void> _load({required bool reset}) async {
    if (reset) {
      setState(() {
        _loading = true;
        _error = null;
      });
    }
    try {
      if (_conversation == null) {
        await _resolveConversation();
      }
      final page = await ref
          .read(messageServiceProvider)
          .listMessages(widget.conversationId);
      if (!mounted) return;
      setState(() {
        _messages
          ..clear()
          ..addAll(page.list);
        _hasMore = page.hasMore;
      });
      _scrollToBottom();
      _markReadIfNeeded();
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = apiErrorMessage(error));
    } finally {
      if (mounted) {
        setState(() => _loading = false);
      }
    }
  }

  Future<void> _resolveConversation() async {
    final page = await ref
        .read(messageServiceProvider)
        .listConversations(limit: 100);
    final match = page.list.where((item) => item.id == widget.conversationId);
    if (match.isNotEmpty) {
      _conversation = match.first;
      _peer = match.first.user;
      return;
    }
    // 会话被隐藏过：通过消息推断对方并恢复视图。
    final messages = await ref
        .read(messageServiceProvider)
        .listMessages(widget.conversationId);
    final myId = _myId;
    for (final message in messages.list) {
      if (message.sender != null && message.sender!.id != myId) {
        _peer = message.sender;
        break;
      }
    }
    final peerId = _peer?.id ??
        messages.list
            .where((item) => item.senderId == myId && item.receiverId != null)
            .map((item) => item.receiverId)
            .firstOrNull;
    if (peerId != null) {
      _conversation = await ref
          .read(messageServiceProvider)
          .createConversation(peerId);
      _peer = _conversation?.user ?? _peer;
    }
  }

  Future<void> _loadOlder() async {
    final oldestId = _messages
        .where((item) => item.id != null)
        .map((item) => item.id!)
        .reduce((a, b) => a < b ? a : b);
    setState(() => _loadingOlder = true);
    _preLoadOffset = _scrollController.offset;
    _preLoadExtent = _scrollController.position.maxScrollExtent;
    try {
      final page = await ref.read(messageServiceProvider).listMessages(
            widget.conversationId,
            beforeId: oldestId,
          );
      if (!mounted) return;
      setState(() {
        _messages.insertAll(0, page.list);
        _hasMore = page.hasMore;
      });
      // 保持视觉位置：加载后补偿高度差。
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        final delta =
            _scrollController.position.maxScrollExtent - _preLoadExtent;
        _scrollController.jumpTo(_preLoadOffset + delta);
      });
    } catch (_) {
      // 历史加载失败可重试（滚动到顶部会再次触发）
    } finally {
      if (mounted) {
        setState(() => _loadingOlder = false);
      }
    }
  }

  void _markReadIfNeeded() {
    final unread = _conversation?.unreadCount ?? 0;
    if (unread == 0 && _messages.isEmpty) return;
    final latest = _messages
        .where((item) => item.id != null)
        .map((item) => item.id!)
        .reduce((a, b) => a > b ? a : b);
    final target = _conversation?.copyWith(unreadCount: 0);
    if (target != null) {
      _conversation = target;
    }
    ref.read(unreadMessagesProvider).consume(unread);
    ref
        .read(messageServiceProvider)
        .markRead(widget.conversationId, latest)
        .catchError((_) {});
  }

  void _onRealtimeEvent(RealtimeEvent event) {
    if (!mounted) return;
    if (event.type == 'message.created') {
      final data = event.data ?? const <String, dynamic>{};
      final conversationId = (data['conversation_id'] as num?)?.toInt();
      if (conversationId != widget.conversationId) return;
      final raw = data['message'];
      if (raw is! Map) return;
      final message =
          Message.fromMap(Map<String, dynamic>.from(raw));
      final nearBottom = _isNearBottom();
      _upsertMessage(message);
      if (message.senderId != _myId) {
        if (message.id != null) {
          ref
              .read(messageServiceProvider)
              .markRead(widget.conversationId, message.id!)
              .catchError((_) {});
          final unread = _conversation?.unreadCount ?? 0;
          final target = _conversation?.copyWith(unreadCount: 0);
          if (target != null) {
            setState(() => _conversation = target);
          }
          ref.read(unreadMessagesProvider).consume(unread);
        }
      }
      if (nearBottom) {
        _scrollToBottom();
      }
    } else if (event.type == 'message.deleted') {
      final conversationId =
          (event.data?['conversation_id'] as num?)?.toInt();
      final messageId = (event.data?['message_id'] as num?)?.toInt();
      if (conversationId != widget.conversationId || messageId == null) return;
      final index = _messages.indexWhere((item) => item.id == messageId);
      if (index >= 0) {
        setState(() {
          final current = _messages.removeAt(index);
          _messages.insert(index, current.markDeleted());
        });
      }
    } else if (event.type == 'conversation.read') {
      final userId = (event.data?['user_id'] as num?)?.toInt();
      if (userId == _myId) {
        final target = _conversation?.copyWith(unreadCount: 0);
        if (target != null) {
          setState(() => _conversation = target);
        }
      }
    }
  }

  /// 按 message.id 去重合并；乐观消息由同内容 sending 项替换。
  void _upsertMessage(Message message) {
    final byId = message.id == null
        ? -1
        : _messages.indexWhere((item) => item.id == message.id);
    if (byId >= 0) {
      setState(() => _messages[byId] = message);
      return;
    }
    if (message.id != null) {
      final pendingIndex = _messages.indexWhere(
        (item) =>
            item.id == null &&
            item.status == SendStatus.sending &&
            item.content == message.content,
      );
      if (pendingIndex >= 0) {
        setState(() {
          _messages[pendingIndex] =
              _messages[pendingIndex].replacedBy(message);
        });
        return;
      }
    }
    setState(() => _messages.add(message));
  }

  bool _isNearBottom() {
    if (!_scrollController.hasClients) return true;
    final position = _scrollController.position;
    return position.maxScrollExtent - position.pixels < 120;
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_scrollController.hasClients) return;
      _scrollController.animateTo(
        _scrollController.position.maxScrollExtent,
        duration: const Duration(milliseconds: 200),
        curve: Curves.easeOut,
      );
    });
  }

  Future<void> _send() async {
    final content = _inputController.text.trim();
    if (content.isEmpty || _sending) return;
    if (content.length > 2000) {
      showAppSnackBar(context, '消息最多 2000 字', error: true);
      return;
    }
    final tempId =
        'local-${DateTime.now().millisecondsSinceEpoch}-${_messages.length}';
    final pending = Message.pending(
      conversationId: widget.conversationId,
      senderId: _myId,
      content: content,
      tempId: tempId,
    );
    setState(() {
      _messages.add(pending);
      _sending = true;
    });
    _inputController.clear();
    _scrollToBottom();
    try {
      final server = await ref
          .read(messageServiceProvider)
          .sendMessage(widget.conversationId, content);
      if (!mounted) return;
      final index =
          _messages.indexWhere((item) => item.clientTempId == tempId);
      if (index >= 0) {
        setState(() => _messages[index] = _messages[index].replacedBy(server));
      } else {
        _upsertMessage(server);
      }
    } catch (error) {
      if (!mounted) return;
      final index =
          _messages.indexWhere((item) => item.clientTempId == tempId);
      if (index >= 0) {
        setState(() => _messages[index] = _messages[index].markFailed());
      }
      showAppSnackBar(context, friendlyMessageError(error), error: true);
    } finally {
      if (mounted) {
        setState(() => _sending = false);
      }
    }
  }

  Future<void> _retry(Message message) async {
    if (message.clientTempId == null || message.status != SendStatus.failed) {
      return;
    }
    setState(() => _sending = true);
    try {
      final server = await ref
          .read(messageServiceProvider)
          .sendMessage(widget.conversationId, message.content);
      if (!mounted) return;
      final index = _messages
          .indexWhere((item) => item.clientTempId == message.clientTempId);
      if (index >= 0) {
        setState(() => _messages[index] = _messages[index].replacedBy(server));
      }
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, friendlyMessageError(error), error: true);
      }
    } finally {
      if (mounted) {
        setState(() => _sending = false);
      }
    }
  }

  Future<void> _deleteMessage(Message message) async {
    if (message.id == null) return;
    final confirmed = await showConfirmDialog(
      context,
      title: '删除消息',
      content: '删除后仅自己视角不可见原内容，无法恢复。',
      confirmText: '删除',
      danger: true,
    );
    if (!confirmed || !mounted) return;
    try {
      await ref.read(messageServiceProvider).deleteMessage(message.id!);
      if (!mounted) return;
      final index = _messages.indexWhere((item) => item.id == message.id);
      if (index >= 0) {
        setState(() {
          final current = _messages.removeAt(index);
          _messages.insert(index, current.markDeleted());
        });
      }
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, friendlyMessageError(error), error: true);
      }
    }
  }

  Future<void> _toggleBlock() async {
    final peerId = _peer?.id;
    if (peerId == null) return;
    final blocked = _conversation?.isBlocked ?? false;
    try {
      final service = ref.read(messageServiceProvider);
      if (blocked) {
        await service.unblockUser(peerId);
      } else {
        await service.blockUser(peerId);
      }
      if (!mounted) return;
      setState(() {
        _conversation = _conversation?.copyWith(
          isBlocked: !blocked,
          canSend: !blocked,
        );
      });
      showAppSnackBar(context, blocked ? '已取消拉黑' : '已拉黑该用户');
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, friendlyMessageError(error), error: true);
      }
    }
  }

  Future<void> _deleteConversation() async {
    final confirmed = await showConfirmDialog(
      context,
      title: '删除会话',
      content: '确定从你的会话列表中删除该聊天吗？对方不受影响，收到新消息后会自动恢复。',
      confirmText: '删除',
      danger: true,
    );
    if (!confirmed || !mounted) return;
    try {
      await ref
          .read(messageServiceProvider)
          .deleteConversation(widget.conversationId);
      if (mounted) {
        context.pop();
      }
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, friendlyMessageError(error), error: true);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final blocked = _conversation?.isBlocked ?? false;
    final canSend = _conversation?.canSend ?? true;
    final composerDisabled = blocked || !canSend;
    return Scaffold(
      appBar: AppBar(
        titleSpacing: 0,
        title: GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTap: _openPeerProfile,
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              UserAvatar(url: _peer?.avatarUrl, size: 34),
              const SizedBox(width: 8),
              Flexible(
                child: Text(
                  _peer?.displayName ?? _peer?.username ?? '私信',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
        ),
        actions: [
          PopupMenuButton<String>(
            onSelected: (value) {
              switch (value) {
                case 'profile':
                  _openPeerProfile();
                case 'block':
                  _toggleBlock();
                case 'delete':
                  _deleteConversation();
              }
            },
            itemBuilder: (context) => [
              if (_peer?.username != null)
                const PopupMenuItem(
                  value: 'profile',
                  child: Text('查看用户主页'),
                ),
              PopupMenuItem(
                value: 'block',
                child: Text(blocked ? '取消拉黑' : '拉黑用户'),
              ),
              const PopupMenuItem(
                value: 'delete',
                child: Text('删除会话'),
              ),
            ],
          ),
        ],
      ),
      body: _buildBody(composerDisabled, blocked),
    );
  }

  void _openPeerProfile() {
    final username = _peer?.username;
    if (username != null && username.isNotEmpty) {
      context.push('/user/$username');
    }
  }

  Widget _buildBody(bool composerDisabled, bool blocked) {
    if (_loading) {
      return const LoadingView();
    }
    if (_error != null && _messages.isEmpty) {
      return ErrorView(
        message: _error!,
        onRetry: () => _load(reset: true),
      );
    }
    if (_messages.isEmpty && _error == null) {
      return Column(
        children: [
          const Expanded(
            child: EmptyView(hint: '还没有消息，发送第一条私信吧'),
          ),
          _buildComposer(composerDisabled, blocked),
        ],
      );
    }
    return Column(
      children: [
        Expanded(child: _buildList()),
        _buildComposer(composerDisabled, blocked),
      ],
    );
  }

  Widget _buildList() {
    return ListView.builder(
      controller: _scrollController,
      reverse: false,
      padding: const EdgeInsets.symmetric(vertical: 12),
      itemCount: _messages.length + (_loadingOlder ? 1 : 0),
      itemBuilder: (context, index) {
        if (_loadingOlder && index == 0) {
          return const Padding(
            padding: EdgeInsets.symmetric(vertical: 10),
            child: Center(
              child: SizedBox(
                width: 20,
                height: 20,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            ),
          );
        }
        final messageIndex = _loadingOlder ? index - 1 : index;
        final message = _messages[messageIndex];
        return _MessageBubble(
          message: message,
          mine: message.senderId == _myId,
          showTime: _shouldShowTime(messageIndex),
          onLongPressOwn: () => _deleteMessage(message),
          onRetry: () => _retry(message),
        );
      },
    );
  }

  bool _shouldShowTime(int index) {
    if (index <= 0) return true;
    final current = _messages[index].createdAt;
    final previous = _messages[index - 1].createdAt;
    if (current == null || previous == null) return true;
    final gap = DateTime.parse(current).millisecondsSinceEpoch -
        DateTime.parse(previous).millisecondsSinceEpoch;
    return gap > 5 * 60 * 1000;
  }

  Widget _buildComposer(bool disabled, bool blocked) {
    final theme = Theme.of(context);
    return SafeArea(
      top: false,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (disabled)
            Container(
              width: double.infinity,
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
              child: Text(
                blocked ? '你已拉黑该用户，取消拉黑后可继续发送' : '当前无法向该用户发送私信',
                style: TextStyle(
                  fontSize: 12,
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ),
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 8),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _inputController,
                    focusNode: _focusNode,
                    enabled: !disabled,
                    minLines: 1,
                    maxLines: 4,
                    maxLength: 2000,
                    textInputAction: TextInputAction.send,
                    onSubmitted: (_) => _send(),
                    decoration: const InputDecoration(
                      hintText: '输入消息...',
                      isDense: true,
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                ValueListenableBuilder<TextEditingValue>(
                  valueListenable: _inputController,
                  builder: (context, value, _) {
                    final canSubmit = !disabled &&
                        !_sending &&
                        value.text.trim().isNotEmpty;
                    return IconButton.filled(
                      onPressed: canSubmit ? _send : null,
                      icon: const Icon(Icons.send_outlined),
                    );
                  },
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _MessageBubble extends StatelessWidget {
  const _MessageBubble({
    required this.message,
    required this.mine,
    required this.showTime,
    this.onLongPressOwn,
    this.onRetry,
  });

  final Message message;
  final bool mine;
  final bool showTime;
  final VoidCallback? onLongPressOwn;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final bubbleColor = mine
        ? theme.colorScheme.primary
        : theme.colorScheme.surfaceContainerHighest;
    final textColor = mine
        ? theme.colorScheme.onPrimary
        : theme.colorScheme.onSurface;
    return Column(
      crossAxisAlignment:
          mine ? CrossAxisAlignment.end : CrossAxisAlignment.start,
      children: [
        if (showTime && message.createdAt != null)
          Padding(
            padding: const EdgeInsets.only(bottom: 6, top: 6),
            child: Text(
              formatDateTime(message.createdAt),
              style: TextStyle(
                fontSize: 11,
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ),
        GestureDetector(
          onLongPress: mine && message.id != null ? onLongPressOwn : null,
          child: Container(
            margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 3),
            padding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            constraints: BoxConstraints(
              maxWidth: MediaQuery.of(context).size.width * 0.72,
            ),
            decoration: BoxDecoration(
              color: message.isDeleted
                  ? bubbleColor.withValues(alpha: 0.45)
                  : bubbleColor,
              borderRadius: BorderRadius.only(
                topLeft: const Radius.circular(12),
                topRight: const Radius.circular(12),
                bottomLeft: Radius.circular(mine ? 12 : 4),
                bottomRight: Radius.circular(mine ? 4 : 12),
              ),
            ),
            child: message.isDeleted
                ? Text(
                    '该消息已删除',
                    style: TextStyle(
                      fontSize: 14,
                      fontStyle: FontStyle.italic,
                      color: textColor.withValues(alpha: 0.7),
                    ),
                  )
                : Column(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        message.content,
                        style: TextStyle(fontSize: 14, color: textColor),
                      ),
                      if (message.status == SendStatus.sending)
                        Padding(
                          padding: const EdgeInsets.only(top: 2),
                          child: Text(
                            '正在发送...',
                            style: TextStyle(
                              fontSize: 10,
                              color: textColor.withValues(alpha: 0.7),
                            ),
                          ),
                        ),
                      if (message.status == SendStatus.failed)
                        Padding(
                          padding: const EdgeInsets.only(top: 2),
                          child: GestureDetector(
                            onTap: onRetry,
                            child: Text(
                              '发送失败，点击重试',
                              style: TextStyle(
                                fontSize: 10,
                                color: theme.colorScheme.error,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ),
                        ),
                    ],
                  ),
          ),
        ),
      ],
    );
  }
}
