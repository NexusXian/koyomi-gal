import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../models/galgame_models.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';
import '../../widgets/markdown_editor.dart';

class PostFormPage extends ConsumerStatefulWidget {
  const PostFormPage({super.key, this.editId, this.galgameId});

  final int? editId;
  final int? galgameId;

  @override
  ConsumerState<PostFormPage> createState() => _PostFormPageState();
}

class _PostFormPageState extends ConsumerState<PostFormPage> {
  final _formKey = GlobalKey<FormState>();
  final _titleController = TextEditingController();
  final _contentController = TextEditingController();

  String _editorMode = 'markdown';
  int? _galgameId;
  String? _galgameTitle;

  bool _loading = false;
  bool _saving = false;

  bool get isEditing => widget.editId != null;

  @override
  void initState() {
    super.initState();
    _galgameId = widget.galgameId;
    if (isEditing) {
      _loadEditing();
    }
  }

  @override
  void dispose() {
    _titleController.dispose();
    _contentController.dispose();
    super.dispose();
  }

  Future<void> _loadEditing() async {
    setState(() => _loading = true);
    try {
      final post = await ref.read(postServiceProvider).get(widget.editId!);
      if (!mounted) {
        return;
      }
      setState(() {
        _titleController.text = post.title ?? '';
        _contentController.text = post.content ?? '';
        _editorMode = post.editorMode;
        _galgameId = post.galgameId;
        _galgameTitle = post.galgameTitle;
        _loading = false;
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() => _loading = false);
      showAppSnackBar(context, apiErrorMessage(error), error: true);
    }
  }

  Future<void> _pickGalgame() async {
    final picked = await showModalBottomSheet<GalgameListItem>(
      context: context,
      isScrollControlled: true,
      builder: (context) => _GalgamePickerSheet(
        onPicked: (galgame) => Navigator.of(context).pop(galgame),
      ),
    );
    if (picked != null) {
      setState(() {
        _galgameId = picked.id;
        _galgameTitle = picked.title;
      });
    }
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    setState(() => _saving = true);
    try {
      final service = ref.read(postServiceProvider);
      if (isEditing) {
        final post = await service.update(
          widget.editId!,
          title: _titleController.text.trim(),
          content: _contentController.text,
          editorMode: _editorMode,
        );
        if (mounted) {
          showAppSnackBar(context, '已保存');
          context.pop(post);
        }
      } else {
        final post = await service.create(
          title: _titleController.text.trim(),
          content: _contentController.text,
          editorMode: _editorMode,
          galgameId: _galgameId,
        );
        if (mounted) {
          showAppSnackBar(context, '发布成功');
          context.pushReplacement('/posts/${post.id}');
        }
      }
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    } finally {
      if (mounted) {
        setState(() => _saving = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(isEditing ? '编辑帖子' : '发布帖子'),
        actions: [
          TextButton(
            onPressed: _saving ? null : _submit,
            child: const Text('发布'),
          ),
        ],
      ),
      body: _loading
          ? const LoadingView()
          : Form(
              key: _formKey,
              child: ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  SegmentedButton<String>(
                    segments: const [
                      ButtonSegment(
                        value: 'markdown',
                        label: Text('Markdown'),
                        icon: Icon(Icons.code, size: 16),
                      ),
                      ButtonSegment(
                        value: 'plain',
                        label: Text('纯文本'),
                        icon: Icon(Icons.text_fields, size: 16),
                      ),
                    ],
                    selected: {_editorMode},
                    onSelectionChanged: (selection) =>
                        setState(() => _editorMode = selection.first),
                  ),
                  const SizedBox(height: 12),
                  TextFormField(
                    controller: _titleController,
                    decoration: const InputDecoration(labelText: '标题 *'),
                    validator: (value) =>
                        value == null || value.trim().isEmpty ? '请输入标题' : null,
                  ),
                  const SizedBox(height: 12),
                  InkWell(
                    onTap: isEditing ? null : _pickGalgame,
                    borderRadius: BorderRadius.circular(10),
                    child: InputDecorator(
                      decoration: const InputDecoration(
                        labelText: '关联 Galgame（可选）',
                      ),
                      child: Row(
                        children: [
                          Expanded(
                            child: Text(
                              _galgameTitle ??
                                  (isEditing ? '保持不变' : '点击选择 Galgame'),
                              style: TextStyle(
                                fontSize: 14,
                                color: _galgameTitle == null
                                    ? Theme.of(context).colorScheme.onSurfaceVariant
                                    : null,
                              ),
                            ),
                          ),
                          if (!isEditing && _galgameId != null)
                            IconButton(
                              icon: const Icon(Icons.close, size: 16),
                              onPressed: () => setState(() {
                                _galgameId = null;
                                _galgameTitle = null;
                              }),
                            ),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 16),
                  if (_editorMode == 'markdown')
                    MarkdownEditor(
                      controller: _contentController,
                      imageCategory: 'posts',
                      minLines: 10,
                    )
                  else
                    TextFormField(
                      controller: _contentController,
                      decoration: const InputDecoration(
                        hintText: '正文内容',
                      ),
                      minLines: 10,
                      maxLines: 24,
                      validator: (value) => value == null || value.isEmpty
                          ? '请输入正文'
                          : null,
                    ),
                ],
              ),
            ),
    );
  }
}

class _GalgamePickerSheet extends ConsumerStatefulWidget {
  const _GalgamePickerSheet({required this.onPicked});

  final void Function(GalgameListItem) onPicked;

  @override
  ConsumerState<_GalgamePickerSheet> createState() =>
      _GalgamePickerSheetState();
}

class _GalgamePickerSheetState extends ConsumerState<_GalgamePickerSheet> {
  final _searchController = TextEditingController();
  List<GalgameListItem> _items = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _search('');
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  Future<void> _search(String keyword) async {
    setState(() => _loading = true);
    try {
      final result = await ref.read(galgameServiceProvider).list(
            keyword: keyword.isEmpty ? null : keyword,
            limit: 20,
          );
      if (mounted) {
        setState(() => _items = result.items);
      }
    } catch (_) {
      if (mounted) {
        setState(() => _items = []);
      }
    } finally {
      if (mounted) {
        setState(() => _loading = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
      child: Column(
        children: [
          TextField(
            controller: _searchController,
            decoration: InputDecoration(
              isDense: true,
              hintText: '搜索 Galgame',
              suffixIcon: IconButton(
                icon: const Icon(Icons.search, size: 18),
                onPressed: () => _search(_searchController.text),
              ),
            ),
            onSubmitted: _search,
          ),
          const SizedBox(height: 8),
          Expanded(
            child: _loading
                ? const Center(
                    child: SizedBox(
                      width: 22,
                      height: 22,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                  )
                : ListView.separated(
                    itemCount: _items.length,
                    separatorBuilder: (_, _) => const Divider(height: 1),
                    itemBuilder: (context, index) {
                      final galgame = _items[index];
                      return ListTile(
                        dense: true,
                        title: Text(
                          galgame.title ?? '',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        onTap: () => widget.onPicked(galgame),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}
