import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

import '../core/api/api_client.dart';
import '../providers/app_providers.dart';

/// Plain-text markdown editor with an insert toolbar (image upload included).
class MarkdownEditor extends ConsumerStatefulWidget {
  const MarkdownEditor({
    super.key,
    required this.controller,
    this.imageCategory = 'posts',
    this.minLines = 8,
  });

  final TextEditingController controller;
  final String imageCategory;
  final int minLines;

  @override
  ConsumerState<MarkdownEditor> createState() => _MarkdownEditorState();
}

class _MarkdownEditorState extends ConsumerState<MarkdownEditor> {
  bool _uploading = false;

  void _insert(String text, {String? selectPlaceholder}) {
    final controller = widget.controller;
    final selection = controller.selection;
    final value = controller.text;
    final start = selection.isValid ? selection.start : value.length;
    final end = selection.isValid ? selection.end : value.length;
    final newText =
        value.replaceRange(start, end, '$text${selectPlaceholder ?? ''}');
    controller.value = controller.value.copyWith(
      text: newText,
      selection: TextSelection.collapsed(
        offset: start + text.length + (selectPlaceholder?.length ?? 0),
      ),
    );
  }

  void _wrap(String prefix, String suffix) {
    final controller = widget.controller;
    final selection = controller.selection;
    if (!selection.isValid) {
      _insert('$prefix$prefix');
      return;
    }
    final selected = controller.text.substring(
      selection.start.clamp(0, controller.text.length),
      selection.end.clamp(0, controller.text.length),
    );
    if (selected.isEmpty) {
      _insert('$prefix$prefix');
      return;
    }
    final start = selection.start;
    final end = selection.end;
    final wrapped = '$prefix$selected$suffix';
    final newText = controller.text.replaceRange(start, end, wrapped);
    controller.value = controller.value.copyWith(
      text: newText,
      selection: TextSelection.collapsed(offset: end + prefix.length + suffix.length),
    );
  }

  Future<void> _uploadImage() async {
    final picker = ImagePicker();
    final picked = await picker.pickImage(source: ImageSource.gallery);
    if (picked == null) {
      return;
    }
    final file = File(picked.path);
    final bytes = await file.readAsBytes();
    final mime = picked.mimeType ?? 'image/jpeg';
    if (!_isAllowedMime(mime)) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('仅支持 JPG、PNG、WebP、AVIF 和 GIF 格式')),
        );
      }
      return;
    }

    setState(() => _uploading = true);
    try {
      final service = ref.read(imageServiceProvider);
      final presign = await service.presign(
        filename: picked.name.split('/').last.isEmpty
            ? 'image'
            : picked.name.split('/').last,
        contentType: mime,
        size: bytes.length,
        category: widget.imageCategory,
      );
      await service.uploadToPresigned(presign.uploadUrl, bytes, mime);
      final asset = await service.complete(presign.id);
      final url = (asset['url'] as String?) ?? '';
      if (url.isNotEmpty) {
        _insert('![]($url)');
      }
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(apiErrorMessage(error))),
        );
      }
    } finally {
      if (mounted) {
        setState(() => _uploading = false);
      }
    }
  }

  static bool _isAllowedMime(String mime) {
    return const [
      'image/jpeg',
      'image/png',
      'image/webp',
      'image/avif',
      'image/gif',
    ].contains(mime);
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(
            children: [
              _ToolButton(label: 'B', tooltip: '粗体', onTap: () => _wrap('**', '**')),
              _ToolButton(label: 'I', tooltip: '斜体', onTap: () => _wrap('*', '*')),
              _ToolButton(
                  label: '~~', tooltip: '删除线', onTap: () => _wrap('~~', '~~')),
              _ToolButton(
                  label: 'H2', tooltip: '标题', onTap: () => _insert('\n## ')),
              _ToolButton(
                  label: '“',
                  tooltip: '引用',
                  onTap: () => _insert('\n> ')),
              _ToolButton(
                  label: '</>',
                  tooltip: '代码',
                  onTap: () => _insert('\n```\n', selectPlaceholder: '\n```\n')),
              _ToolButton(
                label: '🔗',
                tooltip: '链接',
                onTap: () => _insert('[](', selectPlaceholder: ')'),
              ),
              _ToolButton(
                label: _uploading ? '…' : '🖼',
                tooltip: '插入图片',
                onTap: _uploading ? null : _uploadImage,
              ),
            ],
          ),
        ),
        const SizedBox(height: 8),
        TextField(
          controller: widget.controller,
          minLines: widget.minLines,
          maxLines: 24,
          expands: false,
          keyboardType: TextInputType.multiline,
          decoration: const InputDecoration(
            hintText: '支持 Markdown 语法',
            contentPadding: EdgeInsets.all(12),
          ),
        ),
      ],
    );
  }
}

class _ToolButton extends StatelessWidget {
  const _ToolButton({
    required this.label,
    required this.tooltip,
    required this.onTap,
  });

  final String label;
  final String tooltip;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(right: 8),
      child: Tooltip(
        message: tooltip,
        child: OutlinedButton(
          onPressed: onTap,
          style: OutlinedButton.styleFrom(
            minimumSize: const Size(40, 36),
            padding: const EdgeInsets.symmetric(horizontal: 10),
          ),
          child: Text(label, style: const TextStyle(fontSize: 13)),
        ),
      ),
    );
  }
}
