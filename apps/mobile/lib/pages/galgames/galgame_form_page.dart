import 'dart:io';

import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';

import '../../core/api/api_client.dart';
import '../../core/constants/domain.dart';
import '../../providers/app_providers.dart';
import '../../services/galgame_service.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';

class GalgameFormPage extends ConsumerStatefulWidget {
  const GalgameFormPage({super.key, this.editId});

  final int? editId;

  @override
  ConsumerState<GalgameFormPage> createState() => _GalgameFormPageState();
}

class _GalgameFormPageState extends ConsumerState<GalgameFormPage> {
  final _formKey = GlobalKey<FormState>();
  final _titleController = TextEditingController();
  final _romajiController = TextEditingController();
  final _originalTitleController = TextEditingController();
  final _slugController = TextEditingController();
  final _releaseDateController = TextEditingController();
  final _descriptionController = TextEditingController();
  final _aliasesController = TextEditingController();

  int _ageRating = 0;
  int _status = 0;
  bool _coverSensitive = false;
  int? _developerId;
  String? _coverUrl;
  String? _bannerUrl;

  List<TagDataLite> _tags = [];
  List<DeveloperDataLite> _developers = [];
  final List<int> _selectedTagIds = [];

  bool _loading = false;
  bool _saving = false;
  String? _error;

  bool get isEditing => widget.editId != null;

  @override
  void initState() {
    super.initState();
    _loadMeta();
    if (isEditing) {
      _loadEditing();
    }
  }

  @override
  void dispose() {
    _titleController.dispose();
    _romajiController.dispose();
    _originalTitleController.dispose();
    _slugController.dispose();
    _releaseDateController.dispose();
    _descriptionController.dispose();
    _aliasesController.dispose();
    super.dispose();
  }

  Future<void> _loadMeta() async {
    try {
      final results = await Future.wait([
        ref.read(tagServiceProvider).list(),
        ref.read(developerServiceProvider).list(),
      ]);
      if (!mounted) {
        return;
      }
      final developers = results[1] as List<DeveloperDataLite>;
      setState(() {
        _tags = results[0] as List<TagDataLite>;
        _developers = developers;
        // Validate developerId exists in loaded list
        if (_developerId != null &&
            !developers.any((d) => d.id == _developerId)) {
          _developerId = null;
        }
      });
    } catch (_) {}
  }

  Future<void> _loadEditing() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final detail =
          await ref.read(galgameServiceProvider).get(widget.editId!);
      if (!mounted) {
        return;
      }
      setState(() {
        _titleController.text = detail.title ?? '';
        _romajiController.text = detail.romajiTitle ?? '';
        _originalTitleController.text = detail.originalTitle ?? '';
        _slugController.text = detail.slug ?? '';
        _releaseDateController.text = _dateOnly(detail.releaseDate);
        _descriptionController.text = detail.description ?? '';
        _aliasesController.text = (detail.aliases ?? const []).join('\n');
        _ageRating = detail.ageRating ?? 0;
        _status = detail.status ?? 0;
        _coverSensitive = detail.coverSensitive;
        _coverUrl = detail.coverUrl;
        _bannerUrl = detail.bannerUrl;
        _developerId = detail.developer?.id;
        if (_developerId != null &&
            _developers.isNotEmpty &&
            !_developers.any((d) => d.id == _developerId)) {
          _developerId = null;
        }
        _selectedTagIds
            .addAll(detail.tags.map((tag) => tag.id).whereType<int>());
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
      firstDate: DateTime(1980),
      lastDate: DateTime(2100),
    );
    if (picked != null) {
      setState(() {
        _releaseDateController.text =
            DateFormat('yyyy-MM-dd').format(picked);
      });
    }
  }

  Future<void> _pickImage({required bool isBanner}) async {
    final picked = await ImagePicker()
        .pickImage(source: ImageSource.gallery, maxWidth: 2400);
    if (picked == null) {
      return;
    }
    final file = File(picked.path);
    final bytes = await file.readAsBytes();
    final mime = picked.mimeType ?? 'image/jpeg';
    setState(() => _saving = true);
    try {
      final service = ref.read(imageServiceProvider);
      final presign = await service.presign(
        filename: picked.name.split('/').last.isEmpty
            ? 'image'
            : picked.name.split('/').last,
        contentType: mime,
        size: bytes.length,
        category: 'galgames',
      );
      await service.uploadToPresigned(presign.uploadUrl, bytes, mime);
      final asset = await service.complete(presign.id);
      if (!mounted) {
        return;
      }
      setState(() {
        if (isBanner) {
          _bannerUrl = asset['url'] as String?;
        } else {
          _coverUrl = asset['url'] as String?;
        }
      });
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

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    setState(() => _saving = true);
    final payload = <String, dynamic>{
      'title': _titleController.text.trim(),
      'slug': _slugController.text.trim(),
      'romaji_title': _romajiController.text.trim(),
      'original_title': _originalTitleController.text.trim(),
      'release_date': _releaseDateController.text.trim(),
      'description': _descriptionController.text.trim(),
      'aliases': _aliasesController.text
          .split('\n')
          .map((line) => line.trim())
          .where((line) => line.isNotEmpty)
          .toList(),
      'age_rating': _ageRating,
      'status': isEditing ? _status : 0,
      'cover_sensitive': _coverSensitive,
      'cover_url': _coverUrl,
      'banner_url': _bannerUrl,
      'developer_id': _developerId,
      'tag_ids': _selectedTagIds,
    };

    try {
      final service = ref.read(galgameServiceProvider);
      final detail = isEditing
          ? await service.update(widget.editId!, payload)
          : await service.create(payload);
      if (!mounted) {
        return;
      }
      showAppSnackBar(context, isEditing ? '已保存' : '已提交，等待审核');
      if (isEditing) {
        context.pop(true);
      } else if (detail.id != null) {
        context.pushReplacement('/galgames/${detail.id}');
      } else {
        context.pop();
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
        title: Text(isEditing ? '编辑 Galgame' : '新建 Galgame'),
        actions: [
          TextButton(
            onPressed: _saving ? null : _submit,
            child: const Text('保存'),
          ),
        ],
      ),
      body: _buildBody(),
    );
  }

  Widget _buildBody() {
    if (_loading) {
      return const LoadingView();
    }
    if (_error != null) {
      return ErrorView(message: _error!, onRetry: _loadEditing);
    }
    return Form(
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
            decoration: const InputDecoration(
              labelText: 'Slug *（字母、数字、-、_）',
            ),
            validator: (value) {
              if (value == null || value.trim().isEmpty) {
                return '请输入 slug';
              }
              return null;
            },
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: _romajiController,
            decoration: const InputDecoration(labelText: '罗马音标题'),
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: _originalTitleController,
            decoration: const InputDecoration(labelText: '原文标题'),
          ),
          const SizedBox(height: 12),
          if (_developers.isNotEmpty)
            DropdownButtonFormField<int?>(
              isExpanded: true,
              value: _developers.any((d) => d.id == _developerId)
                  ? _developerId
                  : null,
              decoration: const InputDecoration(labelText: '开发商'),
              items: [
                const DropdownMenuItem(value: null, child: Text('未选择')),
                ..._developers.map(
                  (developer) => DropdownMenuItem(
                    value: developer.id,
                    child: Text(developer.name ?? ''),
                  ),
                ),
              ],
              onChanged: (value) =>
                  setState(() => _developerId = value),
            )
          else
            InputDecorator(
              decoration: const InputDecoration(labelText: '开发商'),
              child: const Text('加载中…'),
            ),
          const SizedBox(height: 12),
          InkWell(
            onTap: _pickDate,
            borderRadius: BorderRadius.circular(10),
            child: InputDecorator(
              decoration: const InputDecoration(labelText: '发行日期'),
              child: Text(
                _releaseDateController.text.isEmpty
                    ? '选择日期'
                    : _releaseDateController.text,
              ),
            ),
          ),
          const SizedBox(height: 12),
          DropdownButtonFormField<int>(
            isExpanded: true,
            value: _ageRating,
            decoration: const InputDecoration(labelText: '年龄等级'),
            items: [
              for (final option in ageRatingOptions)
                DropdownMenuItem(value: option.value, child: Text(option.label)),
            ],
            onChanged: (value) =>
                setState(() => _ageRating = value ?? 0),
          ),
          const SizedBox(height: 12),
          if (isEditing)
            DropdownButtonFormField<int>(
              isExpanded: true,
              value: _status,
              decoration: const InputDecoration(labelText: '状态'),
              items: [
                for (final option in galgameStatusOptions)
                  DropdownMenuItem(value: option.value, child: Text(option.label)),
              ],
              onChanged: (value) => setState(() => _status = value ?? 0),
            ),
          const SizedBox(height: 16),
          _buildImagePicker(
            label: '封面',
            url: _coverUrl,
            onPick: () => _pickImage(isBanner: false),
          ),
          const SizedBox(height: 12),
          _buildImagePicker(
            label: '横幅',
            url: _bannerUrl,
            onPick: () => _pickImage(isBanner: true),
          ),
          const SizedBox(height: 8),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('封面为敏感内容'),
            value: _coverSensitive,
            onChanged: (value) => setState(() => _coverSensitive = value),
          ),
          const SizedBox(height: 8),
          TextFormField(
            controller: _descriptionController,
            decoration: const InputDecoration(
              labelText: '简介（支持 Markdown）',
            ),
            minLines: 4,
            maxLines: 10,
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: _aliasesController,
            decoration: const InputDecoration(
              labelText: '别名（每行一个）',
            ),
            minLines: 2,
            maxLines: 6,
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
    );
  }

  Widget _buildImagePicker({
    required String label,
    required String? url,
    required VoidCallback onPick,
  }) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label),
        const SizedBox(height: 6),
        Row(
          children: [
            AppImage(
              url: url,
              width: label == '封面' ? 60 : 100,
              height: label == '封面' ? 80 : 56,
              fit: BoxFit.cover,
              borderRadius: BorderRadius.circular(8),
            ),
            const SizedBox(width: 12),
            OutlinedButton.icon(
              onPressed: onPick,
              icon: const Icon(Icons.upload_outlined, size: 16),
              label: const Text('上传图片'),
            ),
          ],
        ),
      ],
    );
  }
}
