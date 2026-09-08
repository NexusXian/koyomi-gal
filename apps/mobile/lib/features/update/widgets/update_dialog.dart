import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../widgets/markdown_view.dart';
import '../models/app_release.dart';

enum UpdateDialogResult { later, ignore, opened }

Future<UpdateDialogResult?> showUpdateDialog(
  BuildContext context,
  AppRelease release,
) {
  return showDialog<UpdateDialogResult>(
    context: context,
    barrierDismissible: !release.forceUpdate,
    builder: (context) => UpdateDialog(release: release),
  );
}

class UpdateDialog extends StatefulWidget {
  const UpdateDialog({super.key, required this.release});

  final AppRelease release;

  @override
  State<UpdateDialog> createState() => _UpdateDialogState();
}

class _UpdateDialogState extends State<UpdateDialog> {
  bool _opening = false;
  String? _error;

  Future<void> _openDownload() async {
    final uri = Uri.tryParse(widget.release.downloadUrl ?? '');
    if (uri == null || (uri.scheme != 'https' && uri.scheme != 'http')) {
      setState(() => _error = '下载地址无效，请稍后重试');
      return;
    }

    setState(() {
      _opening = true;
      _error = null;
    });
    try {
      final opened = await launchUrl(uri, mode: LaunchMode.externalApplication);
      if (!opened) {
        throw StateError('Unable to open download URL');
      }
      if (mounted && !widget.release.forceUpdate) {
        Navigator.of(context).pop(UpdateDialogResult.opened);
      }
    } catch (_) {
      if (mounted) {
        setState(() => _error = '无法打开浏览器，请重试');
      }
    } finally {
      if (mounted) {
        setState(() => _opening = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final release = widget.release;
    final version = release.latestVersion;
    final title = release.title?.trim();
    final changelog = release.changelog?.trim();
    final fileSize = _formatFileSize(release.fileSize);

    return PopScope(
      canPop: !release.forceUpdate,
      child: AlertDialog(
        title: Text(
          release.forceUpdate
              ? '当前版本需要更新'
              : title == null || title.isEmpty
              ? '发现新版本'
              : title,
        ),
        content: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                if (release.forceUpdate && version != null) ...[
                  Text('当前版本已不再支持，请更新至 ${version.versionName} 后继续使用'),
                  const SizedBox(height: 12),
                ],
                if (version != null)
                  Text(
                    '版本 ${version.versionName} (${version.versionCode})',
                    style: Theme.of(context).textTheme.titleSmall,
                  ),
                if (fileSize != null) ...[
                  const SizedBox(height: 4),
                  Text(
                    '安装包大小：$fileSize',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                ],
                if (changelog != null && changelog.isNotEmpty) ...[
                  const SizedBox(height: 16),
                  MarkdownView(data: changelog),
                ],
                if (_error != null) ...[
                  const SizedBox(height: 12),
                  Text(
                    _error!,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                  ),
                  const SizedBox(height: 8),
                  SelectableText(widget.release.downloadUrl ?? ''),
                ],
              ],
            ),
          ),
        ),
        actions: [
          if (!release.forceUpdate)
            TextButton(
              onPressed: _opening
                  ? null
                  : () => Navigator.of(context).pop(UpdateDialogResult.ignore),
              child: const Text('忽略此版本'),
            ),
          if (!release.forceUpdate)
            TextButton(
              onPressed: _opening
                  ? null
                  : () => Navigator.of(context).pop(UpdateDialogResult.later),
              child: const Text('稍后'),
            ),
          FilledButton(
            onPressed: _opening ? null : _openDownload,
            child: Text(
              _opening
                  ? '正在打开…'
                  : _error == null
                  ? '立即更新'
                  : '重新尝试',
            ),
          ),
        ],
      ),
    );
  }
}

String? _formatFileSize(int? bytes) {
  if (bytes == null || bytes < 0) {
    return null;
  }
  if (bytes < 1024) {
    return '$bytes B';
  }
  const units = ['KB', 'MB', 'GB', 'TB'];
  var value = bytes / 1024;
  var unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return '${value.toStringAsFixed(value >= 10 ? 1 : 2)} ${units[unit]}';
}
