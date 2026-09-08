import '../core/api/api_client.dart';
import '../models/message_models.dart';

class MessageService {
  MessageService(this._api);

  final ApiClient _api;

  Future<Conversation> createConversation(int userId) async {
    final data = await _api.post(
      '/api/v1/messages/conversations',
      data: {'user_id': userId},
    );
    return Conversation.fromMap(Map<String, dynamic>.from(data as Map));
  }

  Future<ConversationPage> listConversations({
    String? cursor,
    int limit = 20,
  }) async {
    final data = await _api.get(
      '/api/v1/messages/conversations',
      queryParameters: {
        'cursor': cursor,
        'limit': limit,
      },
    );
    return ConversationPage.fromMap(Map<String, dynamic>.from(data as Map));
  }

  Future<MessagePage> listMessages(
    int conversationId, {
    int? beforeId,
    int limit = 30,
  }) async {
    final data = await _api.get(
      '/api/v1/messages/conversations/$conversationId/messages',
      queryParameters: {
        'before_id': beforeId,
        'limit': limit,
      },
    );
    return MessagePage.fromMap(Map<String, dynamic>.from(data as Map));
  }

  Future<Message> sendMessage(int conversationId, String content) async {
    final data = await _api.post(
      '/api/v1/messages/conversations/$conversationId/messages',
      data: {'type': 'text', 'content': content},
    );
    return Message.fromMap(Map<String, dynamic>.from(data as Map));
  }

  Future<void> markRead(int conversationId, int messageId) => _api.post(
        '/api/v1/messages/conversations/$conversationId/read',
        data: {'message_id': messageId},
      );

  Future<void> deleteConversation(int conversationId) =>
      _api.delete('/api/v1/messages/conversations/$conversationId');

  Future<void> deleteMessage(int messageId) =>
      _api.delete('/api/v1/messages/$messageId');

  Future<int> unreadCount() async {
    final data = await _api.get('/api/v1/messages/unread-count');
    return ((data as Map?)?['count'] as num?)?.toInt() ?? 0;
  }

  Future<String> messagePermission() async {
    final data = await _api.get('/api/v1/users/me/message-settings');
    return ((data as Map?)?['permission'] as String?) ?? 'everyone';
  }

  Future<void> updateMessagePermission(String permission) => _api.put(
        '/api/v1/users/me/message-settings',
        data: {'permission': permission},
      );

  Future<void> blockUser(int userId) =>
      _api.post('/api/v1/users/$userId/block');

  Future<void> unblockUser(int userId) =>
      _api.delete('/api/v1/users/$userId/block');

  Future<String> createWsTicket() async {
    final data = await _api.post('/api/v1/ws/ticket');
    return ((data as Map?)?['ticket'] as String?) ?? '';
  }
}
