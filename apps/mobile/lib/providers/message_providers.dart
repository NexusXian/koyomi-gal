import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/message_models.dart';
import '../providers/app_providers.dart';
import '../services/message_realtime_service.dart';
import '../services/message_service.dart';

final messageServiceProvider =
    Provider<MessageService>((ref) => MessageService(ref.watch(apiClientProvider)));

final messageRealtimeProvider =
    ChangeNotifierProvider<MessageRealtimeService>((ref) {
  final service = MessageRealtimeService(ref.watch(messageServiceProvider));
  final auth = ref.read(authControllerProvider);
  void onAuthChanged() {
    if (auth.isAuthenticated) {
      service.start();
    } else {
      service.stop();
    }
  }

  auth.addListener(onAuthChanged);
  ref.onDispose(() => auth.removeListener(onAuthChanged));
  if (auth.isAuthenticated) {
    service.start();
  }
  return service;
});

/// 全局私信未读数：轮询 + WebSocket 事件增量维护，登录态变化自动清理。
class UnreadMessages extends ChangeNotifier {
  UnreadMessages(this._ref) {
    _auth = _ref.read(authControllerProvider);
    _auth.addListener(_onAuthChanged);
    _realtime = _ref.read(messageRealtimeProvider);
    _realtime.setResyncHandler(refresh);
    _events = _realtime.events.listen(_onEvent);
    if (_auth.isAuthenticated) {
      _start();
    }
  }

  final Ref _ref;
  late final AuthController _auth;
  late final MessageRealtimeService _realtime;
  late final StreamSubscription<RealtimeEvent> _events;
  Timer? _timer;
  int count = 0;

  void _onAuthChanged() {
    if (_auth.isAuthenticated) {
      _start();
    } else {
      _stop();
      count = 0;
      notifyListeners();
    }
  }

  void _start() {
    refresh();
    _timer?.cancel();
    _timer = Timer.periodic(const Duration(seconds: 60), (_) => refresh());
  }

  void _stop() {
    _timer?.cancel();
    _timer = null;
  }

  void _onEvent(RealtimeEvent event) {
    if (event.type == 'message.created') {
      final message = event.data?['message'];
      final senderId = (message is Map)
          ? (message['sender_id'] as num?)?.toInt()
          : null;
      final mine = senderId == _auth.user?.id;
      if (!mine) {
        count += 1;
        notifyListeners();
      }
    } else if (event.type == 'conversation.read') {
      final userId = (event.data?['user_id'] as num?)?.toInt();
      if (userId == _auth.user?.id) {
        refresh();
      }
    }
  }

  /// 打开会话已读后本地校正，随后仍由轮询兜底。
  void consume(int amount) {
    final next = count - amount;
    count = next > 0 ? next : 0;
    notifyListeners();
  }

  Future<void> refresh() async {
    if (!_auth.isAuthenticated) {
      return;
    }
    try {
      final service = _ref.read(messageServiceProvider);
      final latest = await service.unreadCount();
      count = latest;
      notifyListeners();
    } catch (_) {}
  }

  @override
  void dispose() {
    _stop();
    _events.cancel();
    _auth.removeListener(_onAuthChanged);
    super.dispose();
  }
}

final unreadMessagesProvider =
    ChangeNotifierProvider<UnreadMessages>((ref) {
  ref.watch(authControllerProvider);
  return UnreadMessages(ref);
});
