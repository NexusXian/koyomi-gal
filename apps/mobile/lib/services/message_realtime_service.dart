import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../core/config.dart';
import '../models/message_models.dart';
import 'message_service.dart';

/// 全局私信 WebSocket：一次性 ticket 鉴权、指数退避重连、事件广播。
/// 数据真源始终是 REST/PostgreSQL，这里只做实时通知。
class MessageRealtimeService extends ChangeNotifier {
  MessageRealtimeService(this._service);

  final MessageService _service;

  WebSocketChannel? _channel;
  StreamSubscription? _subscription;
  Timer? _reconnectTimer;
  int _attempts = 0;
  bool _wantConnected = false;
  bool _disposed = false;

  final StreamController<RealtimeEvent> _events =
      StreamController<RealtimeEvent>.broadcast();

  bool connected = false;
  bool get wantConnected => _wantConnected;

  Stream<RealtimeEvent> get events => _events.stream;

  void setResyncHandler(void Function() onResync) {
    _onResync = onResync;
  }

  void Function() _onResync = () {};

  void start() {
    if (_disposed || _wantConnected) return;
    _wantConnected = true;
    _connect();
  }

  void stop() {
    _wantConnected = false;
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    _attempts = 0;
    _closeChannel();
    _setConnected(false);
  }

  /// App 从后台恢复时调用：必要时重连，已连接则同步数据。
  void resume() {
    if (!_wantConnected) return;
    if (!connected && _channel == null) {
      _reconnectTimer?.cancel();
      _reconnectTimer = null;
      _connect();
    } else if (connected) {
      _onResync();
    }
  }

  Future<void> _connect() async {
    if (_disposed || !_wantConnected || _channel != null) {
      return;
    }
    try {
      final ticket = await _service.createWsTicket();
      if (ticket.isEmpty || !_wantConnected || _channel != null) return;

      final base = AppConfig.apiBase.replaceFirst(RegExp(r'^http'), 'ws');
      final channel = WebSocketChannel.connect(
        Uri.parse('$base/api/v1/ws?ticket=$ticket'),
      );
      _channel = channel;
      await channel.ready;
      _setConnected(true);
      _subscription = channel.stream.listen(
        _handleData,
        onError: (_) => _onDisconnected(),
        onDone: _onDisconnected,
        cancelOnError: true,
      );
    } catch (_) {
      _closeChannel();
      _setConnected(false);
      _scheduleReconnect();
    }
  }

  void _handleData(dynamic data) {
    try {
      final decoded = jsonDecode(data.toString());
      if (decoded is Map<String, dynamic>) {
        _events.add(RealtimeEvent.fromMap(decoded));
      }
    } catch (_) {
      // 忽略无法解析的事件
    }
  }

  void _onDisconnected() {
    _closeChannel();
    _setConnected(false);
    if (_wantConnected) {
      _scheduleReconnect();
    }
  }

  void _scheduleReconnect() {
    if (_disposed || !_wantConnected || _reconnectTimer != null) return;
    final delay = Duration(
      milliseconds: min(1000 * pow(2, _attempts).toInt(), 30000),
    );
    _attempts += 1;
    _reconnectTimer = Timer(delay, () {
      _reconnectTimer = null;
      _connect();
    });
  }

  void _closeChannel() {
    _subscription?.cancel();
    _subscription = null;
    _channel?.sink.close();
    _channel = null;
  }

  void _setConnected(bool value) {
    if (value) {
      _attempts = 0;
    }
    if (connected == value) return;
    connected = value;
    notifyListeners();
    if (value) {
      _onResync();
    }
  }

  @override
  void dispose() {
    _disposed = true;
    stop();
    _events.close();
    super.dispose();
  }
}
