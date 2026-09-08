import '../core/api/api_client.dart';
import 'user_models.dart';

enum SendStatus { none, sending, failed }

class MessagePreview {
  const MessagePreview({
    this.id,
    this.senderId,
    this.type,
    required this.content,
    this.isDeleted = false,
    this.createdAt,
  });

  factory MessagePreview.fromMap(Map<String, dynamic> map) => MessagePreview(
        id: (map['id'] as num?)?.toInt(),
        senderId: (map['sender_id'] as num?)?.toInt(),
        type: map['type'] as String?,
        content: map['content'] as String? ?? '',
        isDeleted: map['is_deleted'] as bool? ?? false,
        createdAt: map['created_at'] as String?,
      );

  final int? id;
  final int? senderId;
  final String? type;
  final String content;
  final bool isDeleted;
  final String? createdAt;
}

class Conversation {
  const Conversation({
    this.id,
    this.type,
    this.user,
    this.lastMessage,
    this.lastMessageAt,
    this.unreadCount = 0,
    this.isBlocked = false,
    this.canSend = true,
  });

  factory Conversation.fromMap(Map<String, dynamic> map) => Conversation(
        id: (map['id'] as num?)?.toInt(),
        type: map['type'] as String?,
        user: map['user'] is Map
            ? CommunityUserSummary.fromMap(
                Map<String, dynamic>.from(map['user'] as Map))
            : null,
        lastMessage: map['last_message'] is Map
            ? MessagePreview.fromMap(
                Map<String, dynamic>.from(map['last_message'] as Map))
            : null,
        lastMessageAt: map['last_message_at'] as String?,
        unreadCount: (map['unread_count'] as num?)?.toInt() ?? 0,
        isBlocked: map['is_blocked'] as bool? ?? false,
        canSend: map['can_send'] as bool? ?? true,
      );

  Conversation copyWith({
    int? unreadCount,
    bool? isBlocked,
    bool? canSend,
    MessagePreview? lastMessage,
    String? lastMessageAt,
  }) {
    return Conversation(
      id: id,
      type: type,
      user: user,
      lastMessage: lastMessage ?? this.lastMessage,
      lastMessageAt: lastMessageAt ?? this.lastMessageAt,
      unreadCount: unreadCount ?? this.unreadCount,
      isBlocked: isBlocked ?? this.isBlocked,
      canSend: canSend ?? this.canSend,
    );
  }

  final int? id;
  final String? type;
  final CommunityUserSummary? user;
  final MessagePreview? lastMessage;
  final String? lastMessageAt;
  final int unreadCount;
  final bool isBlocked;
  final bool canSend;
}

class ConversationPage {
  const ConversationPage({
    this.list = const [],
    this.nextCursor,
    this.hasMore = false,
  });

  factory ConversationPage.fromMap(Map<String, dynamic> map) =>
      ConversationPage(
        list: (map['list'] as List? ?? [])
            .whereType<Map>()
            .map((item) =>
                Conversation.fromMap(Map<String, dynamic>.from(item)))
            .toList(),
        nextCursor: map['next_cursor'] as String?,
        hasMore: map['has_more'] as bool? ?? false,
      );

  final List<Conversation> list;
  final String? nextCursor;
  final bool hasMore;
}

class Message {
  const Message({
    this.id,
    this.conversationId,
    this.senderId,
    this.receiverId,
    this.type,
    required this.content,
    this.isDeleted = false,
    this.createdAt,
    this.sender,
    this.clientTempId,
    this.status = SendStatus.none,
  });

  factory Message.fromMap(Map<String, dynamic> map) => Message(
        id: (map['id'] as num?)?.toInt(),
        conversationId: (map['conversation_id'] as num?)?.toInt(),
        senderId: (map['sender_id'] as num?)?.toInt(),
        receiverId: (map['receiver_id'] as num?)?.toInt(),
        type: map['type'] as String?,
        content: map['content'] as String? ?? '',
        isDeleted: map['is_deleted'] as bool? ?? false,
        createdAt: map['created_at'] as String?,
        sender: map['sender'] is Map
            ? CommunityUserSummary.fromMap(
                Map<String, dynamic>.from(map['sender'] as Map))
            : null,
      );

  /// 本地乐观消息：尚未获得服务端 ID。
  Message.pending({
    required this.conversationId,
    required this.senderId,
    required String tempId,
    required this.content,
  })  : id = null,
        receiverId = null,
        type = 'text',
        isDeleted = false,
        createdAt = null,
        sender = null,
        clientTempId = tempId,
        status = SendStatus.sending;

  Message replacedBy(Message server) => Message(
        id: server.id,
        conversationId: server.conversationId ?? conversationId,
        senderId: server.senderId ?? senderId,
        receiverId: server.receiverId,
        type: server.type ?? type,
        content: server.content.isNotEmpty ? server.content : content,
        isDeleted: server.isDeleted,
        createdAt: server.createdAt,
        sender: server.sender,
        clientTempId: clientTempId,
        status: SendStatus.none,
      );

  Message markFailed() => _copy(status: SendStatus.failed);

  Message markDeleted() => Message(
        id: id,
        conversationId: conversationId,
        senderId: senderId,
        receiverId: receiverId,
        type: type,
        content: '',
        isDeleted: true,
        createdAt: createdAt,
        sender: sender,
        clientTempId: clientTempId,
        status: status,
      );

  Message _copy({SendStatus? status}) => Message(
        id: id,
        conversationId: conversationId,
        senderId: senderId,
        receiverId: receiverId,
        type: type,
        content: content,
        isDeleted: isDeleted,
        createdAt: createdAt,
        sender: sender,
        clientTempId: clientTempId,
        status: status ?? this.status,
      );

  final int? id;
  final int? conversationId;
  final int? senderId;
  final int? receiverId;
  final String? type;
  final String content;
  final bool isDeleted;
  final String? createdAt;
  final CommunityUserSummary? sender;
  final String? clientTempId;
  final SendStatus status;
}

class MessagePage {
  const MessagePage({this.list = const [], this.nextCursor, this.hasMore = false});

  factory MessagePage.fromMap(Map<String, dynamic> map) => MessagePage(
        list: (map['list'] as List? ?? [])
            .whereType<Map>()
            .map((item) => Message.fromMap(Map<String, dynamic>.from(item)))
            .toList(),
        nextCursor: (map['next_cursor'] as num?)?.toInt(),
        hasMore: map['has_more'] as bool? ?? false,
      );

  final List<Message> list;
  final int? nextCursor;
  final bool hasMore;
}

class RealtimeEvent {
  const RealtimeEvent({required this.type, this.data});

  factory RealtimeEvent.fromMap(Map<String, dynamic> map) => RealtimeEvent(
        type: map['type'] as String?,
        data: map['data'] is Map
            ? Map<String, dynamic>.from(map['data'] as Map)
            : null,
      );

  final String? type;
  final Map<String, dynamic>? data;
}

/// 将后端业务错误信息映射为用户可读文案。
String friendlyMessageError(Object error) {
  final raw = error.toString();
  if (raw.contains('UserBlocked')) {
    return '暂时无法向该用户发送私信';
  }
  if (raw.contains('MessagePermissionDenied')) {
    return '该用户未开放私信';
  }
  if (raw.contains('MessageRateLimited') || raw.contains('发送过于频繁')) {
    return '发送太频繁，请稍后再试';
  }
  if (raw.contains('MessageTooLong') || raw.contains('2000')) {
    return '消息最多 2000 字';
  }
  if (raw.contains('ConversationAccessDenied')) {
    return '无权访问该会话';
  }
  if (raw.contains('CannotMessageSelf') || raw.contains('自己')) {
    return '不能给自己发送私信';
  }
  return apiErrorMessage(error);
}
