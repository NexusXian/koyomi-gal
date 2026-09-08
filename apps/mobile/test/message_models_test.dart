import 'package:flutter_test/flutter_test.dart';
import 'package:mobile/models/message_models.dart';

void main() {
  group('Conversation', () {
    test('parses conversation payload', () {
      final conversation = Conversation.fromMap(const {
        'id': 8,
        'type': 'direct',
        'user': {
          'id': 1002,
          'username': 'alice',
          'display_name': 'Alice',
          'avatar_url': 'https://cdn.example.com/a.png',
        },
        'last_message': {
          'id': 42,
          'sender_id': 1002,
          'type': 'text',
          'content': '你好',
          'is_deleted': false,
          'created_at': '2026-09-08T12:00:00Z',
        },
        'last_message_at': '2026-09-08T12:00:00Z',
        'unread_count': 3,
        'is_blocked': false,
        'can_send': true,
      });

      expect(conversation.id, 8);
      expect(conversation.user?.username, 'alice');
      expect(conversation.lastMessage?.content, '你好');
      expect(conversation.unreadCount, 3);
      expect(conversation.canSend, true);
    });

    test('parses empty conversation without last message', () {
      final conversation = Conversation.fromMap(const {
        'id': 9,
        'type': 'direct',
        'user': {'id': 1003, 'username': 'bob'},
        'unread_count': 0,
      });

      expect(conversation.lastMessage, isNull);
      expect(conversation.lastMessageAt, isNull);
      expect(conversation.unreadCount, 0);
      expect(conversation.canSend, true);
    });
  });

  group('Message', () {
    test('parses message payload', () {
      final message = Message.fromMap(const {
        'id': 42,
        'conversation_id': 8,
        'sender_id': 1002,
        'receiver_id': 1001,
        'type': 'text',
        'content': '你好',
        'is_deleted': false,
        'created_at': '2026-09-08T12:00:00Z',
        'sender': {'id': 1002, 'username': 'alice'},
      });

      expect(message.id, 42);
      expect(message.senderId, 1002);
      expect(message.sender?.username, 'alice');
      expect(message.status, SendStatus.none);
    });

    test('pending message replaced by server message', () {
      final pending = Message.pending(
        conversationId: 8,
        senderId: 1001,
        content: 'hello',
        tempId: 'local-1',
      );
      expect(pending.id, isNull);
      expect(pending.status, SendStatus.sending);

      final replaced = pending.replacedBy(Message.fromMap(const {
        'id': 50,
        'conversation_id': 8,
        'sender_id': 1001,
        'type': 'text',
        'content': 'hello',
      }));
      expect(replaced.id, 50);
      expect(replaced.status, SendStatus.none);
      expect(replaced.clientTempId, 'local-1');

      expect(pending.markFailed().status, SendStatus.failed);
      expect(pending.markDeleted().isDeleted, true);
      expect(pending.markDeleted().content, '');
    });
  });

  group('pages', () {
    test('parses conversation page with cursor', () {
      final page = ConversationPage.fromMap(const {
        'list': [
          {
            'id': 1,
            'user': {'id': 2, 'username': 'a'},
          }
        ],
        'next_cursor': 'abc',
        'has_more': true,
      });
      expect(page.list, hasLength(1));
      expect(page.hasMore, true);
      expect(page.nextCursor, 'abc');
    });

    test('parses message page with numeric cursor', () {
      final page = MessagePage.fromMap(const {
        'list': [
          {'id': 5, 'content': 'x'}
        ],
        'next_cursor': 3,
        'has_more': true,
      });
      expect(page.list, hasLength(1));
      expect(page.nextCursor, 3);
    });

    test('empty list defaults', () {
      final page = ConversationPage.fromMap(const {});
      expect(page.list, isEmpty);
      expect(page.hasMore, false);
    });
  });

  group('RealtimeEvent', () {
    test('parses message.created event', () {
      final event = RealtimeEvent.fromMap(const {
        'type': 'message.created',
        'data': {
          'conversation_id': 8,
          'message': {'id': 42, 'sender_id': 1002, 'content': '你好'},
        },
      });
      expect(event.type, 'message.created');
      expect(event.data?['conversation_id'], 8);
    });
  });

  group('friendlyMessageError', () {
    test('maps business errors', () {
      expect(friendlyMessageError('Exception: UserBlocked'),
          '暂时无法向该用户发送私信');
      expect(friendlyMessageError('Exception: MessagePermissionDenied'),
          '该用户未开放私信');
      expect(friendlyMessageError('Exception: 消息发送过于频繁'),
          '发送太频繁，请稍后再试');
    });
  });
}
