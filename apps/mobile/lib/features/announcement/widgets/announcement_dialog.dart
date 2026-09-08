import 'package:flutter/material.dart';

import '../../../widgets/markdown_view.dart';
import '../models/announcement.dart';

Future<void> showAnnouncementDialog(
  BuildContext context,
  Announcement announcement, {
  required Future<void> Function() onAcknowledge,
}) async {
  final acknowledged = await showDialog<bool>(
    context: context,
    barrierDismissible: announcement.dismissible,
    builder: (context) => AnnouncementDialog(
      announcement: announcement,
      onAcknowledge: onAcknowledge,
    ),
  );
  if (announcement.dismissible && acknowledged != true) {
    await onAcknowledge();
  }
}

class AnnouncementDialog extends StatefulWidget {
  const AnnouncementDialog({
    super.key,
    required this.announcement,
    required this.onAcknowledge,
  });

  final Announcement announcement;
  final Future<void> Function() onAcknowledge;

  @override
  State<AnnouncementDialog> createState() => _AnnouncementDialogState();
}

class _AnnouncementDialogState extends State<AnnouncementDialog> {
  bool _saving = false;
  bool _allowPop = false;
  String? _error;

  Future<void> _acknowledge() async {
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await widget.onAcknowledge();
      if (!mounted) {
        return;
      }
      setState(() => _allowPop = true);
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          Navigator.of(context).pop(true);
        }
      });
    } catch (_) {
      if (mounted) {
        setState(() {
          _saving = false;
          _error = '无法保存确认状态，请重试';
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final announcement = widget.announcement;
    return PopScope(
      canPop: announcement.dismissible || _allowPop,
      child: AlertDialog(
        title: Text(announcement.title),
        content: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                MarkdownView(data: announcement.content),
                if (_error != null) ...[
                  const SizedBox(height: 12),
                  Text(
                    _error!,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                  ),
                ],
              ],
            ),
          ),
        ),
        actions: [
          FilledButton(
            onPressed: _saving ? null : _acknowledge,
            child: Text(_saving ? '正在保存…' : '我知道了'),
          ),
        ],
      ),
    );
  }
}
