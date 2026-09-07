import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/constants/domain.dart';
import '../../providers/app_providers.dart';
import '../../services/galgame_service.dart';
import '../../widgets/common_views.dart';

class NovelFormPage extends ConsumerStatefulWidget {
  const NovelFormPage({super.key, this.editId});

  final int? editId;

  @override
  ConsumerState<NovelFormPage> createState() => _NovelFormPageState();
}

class _NovelFormPageState extends ConsumerState<NovelFormPage> {
  final _formKey = GlobalKey<FormState>();
  final _titleController = TextEditingController();
  final _originalTitleController = TextEditingController();
  final _slugController = TextEditingController();
  final _authorController = TextEditingController();
  final _illustratorController = TextEditingController();
  final _publisherController = TextEditingController();
  final _labelController = TextEditingController();
  final _languageController = TextEditingController();
  final _regionController = TextEditingController();
  final _websiteController = TextEditingController();
  final _releaseDateController = TextEditingController();
  final _summaryController = TextEditingController();

  int _ageRating = 0;
  int _releaseStatusIndex = 0;
  bool _saving = false;
  bool _loading = false;
  String? _error;

  List<TagDataLite> _tags = [];
  final List<int> _selectedTagIds = [];

  bool get isEditing => widget.editId != null;

  @override
  void initState() {
    super.initState();
    _loadTags();
    if (isEditing) {
      _loadEditing();
    }
  }

  @override
  void dispose() {
    _titleController.dispose();
    _originalTitleController.dispose();
    _slugController.dispose();
    _authorController.dispose();
    _illustratorController.dispose();
    _publisherController.dispose();
    _labelController.dispose();
    _languageController.dispose();
    _regionController.dispose();
    _websiteController.dispose();
    _releaseDateController.dispose();
    _summaryController.dispose();
    super.dispose();
  }

  Future<void> _loadTags() async {
    try {
      final tags = await ref.read(tagServiceProvider).list();
      if (mounted) {
        setState(() => _tags = tags);
      }
    } catch (_) {}
  }

  Future<void> _loadEditing() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final novel = await ref.read(novelServiceProvider).get(widget.editId!);
      if (!mounted) {
        return;
      }
      setState(() {
        _titleController.text = novel.title ?? '';
        _originalTitleController.text = novel.originalTitle ?? '';
        _slugController.text = novel.slug ?? '';
        _authorController.text = novel.author ?? '';
        _illustratorController.text = novel.illustrator ?? '';
        _publisherController.text = novel.publisher ?? '';
        _labelController.text = novel.label ?? '';
        _languageController.text = novel.language ?? '';
        _regionController.text = novel.region ?? '';
        _websiteController.text = novel.officialWebsite ?? '';
        _releaseDateController.text = _dateOnly(novel.firstReleaseDate);
        _summaryController.text = novel.summary ?? '';
        _ageRating = novel.ageRating ?? 0;
        _releaseStatusIndex = domainValueFromSlug(
            novelReleaseStatusOptions, novel.releaseStatus);
        _selectedTagIds
            .addAll(novel.tags.map((tag) => tag.id).whereType<int>());
        _loading = false;
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _error = apiErrorMessage(error);
        _loading = false;
      });
    }
  }

  static String _dateOnly(String? value) {
    if (value == null) {
      return '';
    }
    return value.length >= 10 ? value.substring(0, 10) : value;
  }

  Future<void> _pickDate() async {
    final parsed = DateTime.tryParse('${_releaseDateController.text}T00:00:00');
    final picked = await showDatePicker(
      context: context,
      initialDate: parsed ?? DateTime.now(),
      firstDate: DateTime(1950),
      lastDate: DateTime(2100),
    );
    if (picked != null) {
      setState(() {
        _releaseDateController.text = DateFormat('yyyy-MM-dd').format(picked);
      });
    }
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    setState(() => _saving = true);
    final payload = <String, dynamic>{
      'title': _titleController.text.trim(),
      'slug': _slugController.text.trim(),
      'original_title': _originalTitleController.text.trim(),
      'author': _authorController.text.trim(),
      'illustrator': _illustratorController.text.trim(),
      'publisher': _publisherController.text.trim(),
      'label': _labelController.text.trim(),
      'language': _languageController.text.trim(),
      'region': _regionController.text.trim(),
      'official_website': _websiteController.text.trim(),
      'first_release_date': _releaseDateController.text.trim(),
      'summary': _summaryController.text.trim(),
      'age_rating': _ageRating,
      'release_status':
          domainSlug(novelReleaseStatusOptions, _releaseStatusIndex),
      'tag_ids': _selectedTagIds,
    };

    try {
      final service = ref.read(novelServiceProvider);
      if (isEditing) {
        payload['status'] = 1;
        payload['is_cover_sensitive'] = false;
        await service.update(widget.editId!, payload);
        if (mounted) {
          showAppSnackBar(context, '已保存');
          context.pop(true);
        }
      } else {
        final novel = await service.create(payload);
        if (mounted) {
          showAppSnackBar(context, '已提交，等待审核');
          if (novel.id != null) {
            context.pushReplacement('/novels/${novel.id}');
          } else {
            context.pop();
          }
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
        title: Text(isEditing ? '编辑小说' : '新建小说'),
        actions: [
          TextButton(
            onPressed: _saving ? null : _submit,
            child: const Text('保存'),
          ),
        ],
      ),
      body: _loading
          ? const LoadingView()
          : _error != null
              ? ErrorView(message: _error!, onRetry: _loadEditing)
              : Form(
                  key: _formKey,
                  child: ListView(
                    padding: const EdgeInsets.all(16),
                    children: [
                      TextFormField(
                        controller: _titleController,
                        decoration: const InputDecoration(labelText: '标题 *'),
                        validator: (value) =>
                            value == null || value.trim().isEmpty ? '请输入标题' : null,
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _slugController,
                        decoration:
                            const InputDecoration(labelText: 'Slug *'),
                        validator: (value) =>
                            value == null || value.trim().isEmpty ? '请输入 slug' : null,
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _originalTitleController,
                        decoration:
                            const InputDecoration(labelText: '原文标题'),
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _authorController,
                        decoration: const InputDecoration(labelText: '作者'),
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _illustratorController,
                        decoration: const InputDecoration(labelText: '插画师'),
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _publisherController,
                        decoration: const InputDecoration(labelText: '出版社'),
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _labelController,
                        decoration: const InputDecoration(labelText: '文库 / Label'),
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _languageController,
                        decoration:
                            const InputDecoration(labelText: '语言（如 ja、zh-CN）'),
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _regionController,
                        decoration: const InputDecoration(labelText: '地区'),
                      ),
                      const SizedBox(height: 12),
                      InkWell(
                        onTap: _pickDate,
                        borderRadius: BorderRadius.circular(10),
                        child: InputDecorator(
                          decoration:
                              const InputDecoration(labelText: '初版日期'),
                          child: Text(
                            _releaseDateController.text.isEmpty
                                ? '选择日期'
                                : _releaseDateController.text,
                          ),
                        ),
                      ),
                      const SizedBox(height: 12),
                      DropdownButtonFormField<int>(
                        initialValue: _ageRating,
                        decoration: const InputDecoration(labelText: '年龄等级'),
                        items: [
                          for (final option in ageRatingOptions)
                            DropdownMenuItem(
                                value: option.value, child: Text(option.label)),
                        ],
                        onChanged: (value) =>
                            setState(() => _ageRating = value ?? 0),
                      ),
                      const SizedBox(height: 12),
                      DropdownButtonFormField<int>(
                        initialValue: _releaseStatusIndex,
                        decoration: const InputDecoration(labelText: '连载状态'),
                        items: [
                          for (final option in novelReleaseStatusOptions)
                            DropdownMenuItem(
                                value: option.value, child: Text(option.label)),
                        ],
                        onChanged: (value) =>
                            setState(() => _releaseStatusIndex = value ?? 0),
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _websiteController,
                        decoration: const InputDecoration(labelText: '官方网站'),
                        keyboardType: TextInputType.url,
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _summaryController,
                        decoration:
                            const InputDecoration(labelText: '简介（支持 Markdown）'),
                        minLines: 4,
                        maxLines: 10,
                      ),
                      const SizedBox(height: 16),
                      Text('Tag（已选 ${_selectedTagIds.length}）'),
                      const SizedBox(height: 8),
                      Wrap(
                        spacing: 8,
                        runSpacing: 8,
                        children: [
                          for (final tag in _tags)
                            FilterChip(
                              label: Text(tag.name ?? ''),
                              selected: _selectedTagIds.contains(tag.id),
                              onSelected: (selected) {
                                setState(() {
                                  if (selected) {
                                    _selectedTagIds.add(tag.id!);
                                  } else {
                                    _selectedTagIds.remove(tag.id);
                                  }
                                });
                              },
                            ),
                        ],
                      ),
                      const SizedBox(height: 24),
                      FilledButton(
                        onPressed: _saving ? null : _submit,
                        child: _saving
                            ? const SizedBox(
                                width: 20,
                                height: 20,
                                child: CircularProgressIndicator(strokeWidth: 2),
                              )
                            : Text(isEditing ? '保存修改' : '提交'),
                      ),
                    ],
                  ),
                ),
    );
  }
}
