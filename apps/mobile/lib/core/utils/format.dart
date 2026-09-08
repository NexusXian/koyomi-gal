import 'package:intl/intl.dart';

String maskEmail(String? email) {
  final value = email?.trim() ?? '';
  final atIndex = value.indexOf('@');
  if (atIndex <= 0 || atIndex == value.length - 1) {
    return '';
  }
  final local = value.substring(0, atIndex);
  final domain = value.substring(atIndex + 1);
  final visible = local.length < 3 ? 1 : 2;
  return '${local.substring(0, visible)}***@$domain';
}

String formatDateTime(String? value) {
  if (value == null || value.isEmpty) {
    return '-';
  }
  final date = DateTime.tryParse(value);
  if (date == null) {
    return value;
  }
  return DateFormat('yyyy-MM-dd HH:mm').format(date.toLocal());
}

String formatDate(String? value) {
  if (value == null || value.isEmpty) {
    return '-';
  }
  final date = DateTime.tryParse(value);
  if (date == null) {
    return value;
  }
  return DateFormat('yyyy-MM-dd').format(date.toLocal());
}

String formatRelative(String? value) {
  if (value == null || value.isEmpty) {
    return '-';
  }
  final date = DateTime.tryParse(value)?.toLocal();
  if (date == null) {
    return value;
  }
  final diff = DateTime.now().difference(date);
  if (diff.inMinutes < 1) {
    return '刚刚';
  }
  if (diff.inMinutes < 60) {
    return '${diff.inMinutes} 分钟前';
  }
  if (diff.inHours < 24) {
    return '${diff.inHours} 小时前';
  }
  if (diff.inDays < 30) {
    return '${diff.inDays} 天前';
  }
  return DateFormat('yyyy-MM-dd').format(date);
}

String formatCount(int? value) {
  final count = value ?? 0;
  if (count >= 100000000) {
    return '${(count / 100000000).toStringAsFixed(1)}亿';
  }
  if (count >= 10000) {
    return '${(count / 10000).toStringAsFixed(1)}万';
  }
  return count.toString();
}

String formatPlayTime(int? minutes) {
  final total = minutes ?? 0;
  if (total <= 0) {
    return '0 分钟';
  }
  final hours = total ~/ 60;
  final rest = total % 60;
  if (hours <= 0) {
    return '$rest 分钟';
  }
  return rest > 0 ? '$hours 小时 $rest 分钟' : '$hours 小时';
}

String? encodeSlug(String input) {
  final trimmed = input.trim();
  if (trimmed.isEmpty) {
    return null;
  }
  final replaced = trimmed
      .replaceAll(RegExp(r'\s+'), '-')
      .replaceAll(RegExp(r'[^a-zA-Z0-9\-_]'), '');
  return replaced.isEmpty ? null : replaced;
}
