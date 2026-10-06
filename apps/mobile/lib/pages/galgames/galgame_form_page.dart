import 'dart:io';

import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';

import '../../core/api/api_client.dart';
import '../../core/constants/domain.dart';
import '../../models/galgame_models.dart';
import '../../models/company_models.dart';
import '../../providers/app_providers.dart';
import '../../services/galgame_service.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';
import '../companies/widgets.dart';

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
  final _aliasesController = TextEditingController();

  static const _descriptionLocales = [
    ('zh-CN', '中文'),
    ('en-US', 'English'),
    ('ja-JP', '日本語'),
  ];

  static const _sourceTypeOptions = [
    ('nextmoe', 'NextMoe', 'NextMoe 资料库'),
    ('vndb', 'VNDB', 'VNDB'),
    ('official', '游戏官网', '游戏官网'),
    ('bangumi', 'Bangumi', 'Bangumi'),
    ('steam', 'Steam', 'Steam'),
    ('manual', '手动录入', '手动录入'),
    ('unknown', '其他', '未知来源'),
  ];

  // 每种语言的简介相互独立，分别保存内容与来源信息。
  final Map<String, TextEditingController> _descriptionControllers = {
    for (final (locale, _) in _descriptionLocales)
      locale: TextEditingController(),
  };
  final Map<String, TextEditingController> _sourceNameControllers = {
    for (final (locale, _) in _descriptionLocales)
      locale: TextEditingController(),
  };
  final Map<String, TextEditingController> _sourceUrlControllers = {
    for (final (locale, _) in _descriptionLocales)
      locale: TextEditingController(),
  };
  final Map<String, String> _sourceTypes = {
    for (final (locale, _) in _descriptionLocales)
      locale: locale == 'zh-CN'
          ? 'nextmoe'
          : locale == 'en-US'
          ? 'vndb'
          : 'official',
  };
  final Map<String, bool> _sourceOfficial = {
    for (final (locale, _) in _descriptionLocales) locale: locale == 'ja-JP',
  };
  String _descriptionLocale = 'zh-CN';

  int _ageRating = 0;
  int _status = 0;
  bool _coverSensitive = false;
  int? _developerId;
  String? _coverUrl;
  String? _bannerUrl;

  List<TagDataLite> _tags = [];
  List<DeveloperDataLite> _developers = [];
  final List<int> _selectedTagIds = [];
  List<GameCompany> _companies = [];
  bool _companiesDirty = false;
  int? _savedNewGameId;
  int? _savedNewGameStatus;
  String? _editingTitle;

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
    _aliasesController.dispose();
    for (final controller in _descriptionControllers.values) {
      controller.dispose();
    }
    for (final controller in _sourceNameControllers.values) {
      controller.dispose();
    }
    for (final controller in _sourceUrlControllers.values) {
      controller.dispose();
    }
    super.dispose();
  }

  void _loadDescriptionDrafts(GalgameDetail detail) {
    for (final (locale, _) in _descriptionLocales) {
      final stored = detail.descriptions[locale];
      if (stored == null) {
        continue;
      }
      _descriptionControllers[locale]!.text = stored.content;
      if (stored.source.type != 'unknown') {
        _sourceTypes[locale] = stored.source.type;
        _sourceNameControllers[locale]!.text = stored.source.name;
        _sourceUrlControllers[locale]!.text = stored.source.url ?? '';
        _sourceOfficial[locale] = stored.source.official;
      }
    }
  }

  void _onSourceTypeChanged(String locale, String sourceType) {
    setState(() {
      _sourceTypes[locale] = sourceType;
      _sourceOfficial[locale] = sourceType == 'official';
      _sourceNameControllers[locale]!.text = _sourceTypeOptions
          .firstWhere((option) => option.$1 == sourceType)
          .$3;
    });
  }

  List<Map<String, dynamic>> _descriptionPayload() {
    return [
      for (final (locale, _) in _descriptionLocales)
        {
          'language': locale,
          'content': _descriptionControllers[locale]!.text.trim(),
          'source_type': _sourceTypes[locale],
          'source_name': _sourceNameControllers[locale]!.text.trim(),
          'source_url': _sourceUrlControllers[locale]!.text.trim(),
          'is_official': _sourceOfficial[locale] ?? false,
        },
    ];
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
      final permissions = await ref.read(mePermissionsProvider.future);
      if (!mounted) return;
      final service = ref.read(galgameServiceProvider);
      final detail = permissions.hasAny(['galgame:review', 'company:manage'])
          ? await service.adminGet(widget.editId!)
          : await service.get(widget.editId!);
      if (!mounted) {
        return;
      }
      setState(() {
        _titleController.text = detail.title ?? '';
        _editingTitle = detail.title;
        _romajiController.text = detail.romajiTitle ?? '';
        _originalTitleController.text = detail.originalTitle ?? '';
        _slugController.text = detail.slug ?? '';
        _releaseDateController.text = _dateOnly(detail.releaseDate);
        _loadDescriptionDrafts(detail);
        _aliasesController.text = (detail.aliases ?? const []).join('\n');
        _ageRating = detail.ageRating ?? 0;
        _status = detail.status ?? 0;
        _coverSensitive = detail.coverSensitive;
        _coverUrl = detail.coverUrl;
        _bannerUrl = detail.bannerUrl;
        _developerId = detail.developer?.id;
        _companies = detail.companies;
        if (_developerId != null &&
            _developers.isNotEmpty &&
            !_developers.any((d) => d.id == _developerId)) {
          _developerId = null;
        }
        _selectedTagIds
          ..clear()
          ..addAll(detail.tags.map((tag) => tag.id).whereType<int>());
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
        _releaseDateController.text = DateFormat('yyyy-MM-dd').format(picked);
      });
    }
  }

  Future<void> _pickImage({required bool isBanner}) async {
    final picked = await ImagePicker().pickImage(
      source: ImageSource.gallery,
      maxWidth: 2400,
    );
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
    if (_saving || _loading || _error != null) {
      return;
    }
    final permissions = ref.read(mePermissionsProvider).asData?.value;
    if (_savedNewGameId != null ||
        (isEditing && !(permissions?.has('galgame:update') ?? false))) {
      await _saveCompanies();
      return;
    }
    if (!(permissions?.has(isEditing ? 'galgame:update' : 'galgame:create') ??
            false) ||
        !(_formKey.currentState?.validate() ?? false)) {
      return;
    }
    if (_companiesDirty &&
        !(ref.read(mePermissionsProvider).asData?.value.has('company:manage') ??
            false)) {
      showAppSnackBar(context, '无法保存公司关联，请确认公司管理权限', error: true);
      return;
    }
    setState(() => _saving = true);
    final payload = <String, dynamic>{
      'title': _titleController.text.trim(),
      'slug': _slugController.text.trim(),
      'romaji_title': _romajiController.text.trim(),
      'original_title': _originalTitleController.text.trim(),
      'release_date': _releaseDateController.text.trim(),
      // 三种语言相互独立，同时保留旧字段兼容旧版后端。
      'descriptions': _descriptionPayload(),
      'description': _descriptionControllers['zh-CN']!.text.trim(),
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
      if (!isEditing) {
        _savedNewGameId = detail.id;
        _savedNewGameStatus = detail.status;
      }
      if (!mounted) return;
      final canManageCompanies =
          ref.read(mePermissionsProvider).asData?.value.has('company:manage') ??
          false;
      if (_companiesDirty && !canManageCompanies) {
        showAppSnackBar(context, '游戏已保存，公司关联未保存，请确认公司管理权限后重试', error: true);
        return;
      }
      if (_companiesDirty && canManageCompanies) {
        final id = detail.id ?? widget.editId;
        if (id == null) throw StateError('游戏已保存，但未返回 ID，无法关联公司');
        try {
          await ref
              .read(companyServiceProvider)
              .replaceGameCompanies(id, _companies);
          _companiesDirty = false;
        } catch (error) {
          if (mounted) {
            showAppSnackBar(
              context,
              '游戏已保存，公司关联保存失败，请重试：${apiErrorMessage(error)}',
              error: true,
            );
          }
          return;
        }
      }
      if (!mounted) {
        return;
      }
      _finishSave();
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

  Future<void> _saveCompanies() async {
    final id = widget.editId ?? _savedNewGameId;
    if (id == null ||
        _saving ||
        _loading ||
        _error != null ||
        !_companiesDirty ||
        !(ref.read(mePermissionsProvider).asData?.value.has('company:manage') ??
            false)) {
      return;
    }
    setState(() => _saving = true);
    try {
      await ref
          .read(companyServiceProvider)
          .replaceGameCompanies(id, _companies);
      if (!mounted) return;
      setState(() => _companiesDirty = false);
      if (_savedNewGameId != null) {
        _finishSave();
      } else {
        showAppSnackBar(context, '公司关联已保存');
      }
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  void _finishSave() {
    final published = _savedNewGameStatus == 1;
    ScaffoldMessenger.of(context).clearSnackBars();
    showAppSnackBar(
      context,
      isEditing
          ? '已保存'
          : published
          ? '已创建'
          : '已提交，等待审核',
    );
    if (isEditing) {
      context.pop(true);
    } else if (published && _savedNewGameId != null) {
      context.pushReplacement('/galgames/$_savedNewGameId');
    } else {
      context.go('/galgames');
    }
  }

  @override
  Widget build(BuildContext context) {
    final permissions = ref.watch(mePermissionsProvider);
    final canEditMetadata =
        permissions.asData?.value.has(
          isEditing ? 'galgame:update' : 'galgame:create',
        ) ??
        false;
    final canManageCompanies =
        permissions.asData?.value.has('company:manage') ?? false;
    final relationsOnly =
        _savedNewGameId != null || (isEditing && !canEditMetadata);
    return Scaffold(
      appBar: AppBar(
        title: Text(
          relationsOnly
              ? '编辑 Galgame 公司关联'
              : isEditing
              ? '编辑 Galgame'
              : '新建 Galgame',
        ),
        actions: [
          if (relationsOnly ? canManageCompanies : canEditMetadata)
            TextButton(
              onPressed:
                  _saving ||
                      _loading ||
                      _error != null ||
                      (relationsOnly && !_companiesDirty)
                  ? null
                  : _submit,
              child: Text(relationsOnly ? '保存公司关联' : '保存'),
            ),
        ],
      ),
      body: permissions.when(
        loading: () => const LoadingView(),
        error: (error, _) => ErrorView(
          message: apiErrorMessage(error),
          onRetry: () {
            ref.invalidate(mePermissionsProvider);
            if (isEditing) _loadEditing();
          },
        ),
        data: (_) => _buildBody(
          relationsOnly: relationsOnly,
          canEditMetadata: canEditMetadata,
          canManageCompanies: canManageCompanies,
        ),
      ),
    );
  }

  Widget _buildBody({
    required bool relationsOnly,
    required bool canEditMetadata,
    required bool canManageCompanies,
  }) {
    if (_loading) {
      return const LoadingView();
    }
    if (_error != null) {
      return ErrorView(message: _error!, onRetry: _loadEditing);
    }
    if (relationsOnly) {
      if (!canManageCompanies) return const EmptyView(hint: '无权编辑公司关联');
      return ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Text(
            '${_editingTitle ?? _titleController.text} · ID ${widget.editId ?? _savedNewGameId}',
          ),
          const SizedBox(height: 12),
          if (_savedNewGameId != null) const Text('游戏已保存，仅需重试公司关联。'),
          const SizedBox(height: 12),
          _buildCompanyEditor(),
        ],
      );
    }
    if (!canEditMetadata) return const EmptyView(hint: '无权编辑游戏');
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
            decoration: const InputDecoration(labelText: 'Slug *（字母、数字、-、_）'),
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
              onChanged: (value) => setState(() => _developerId = value),
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
          if (canManageCompanies) ...[
            const SizedBox(height: 16),
            _buildCompanyEditor(),
          ],
          const SizedBox(height: 12),
          DropdownButtonFormField<int>(
            isExpanded: true,
            value: _ageRating,
            decoration: const InputDecoration(labelText: '年龄等级'),
            items: [
              for (final option in ageRatingOptions)
                DropdownMenuItem(
                  value: option.value,
                  child: Text(option.label),
                ),
            ],
            onChanged: (value) => setState(() => _ageRating = value ?? 0),
          ),
          const SizedBox(height: 12),
          if (isEditing)
            DropdownButtonFormField<int>(
              isExpanded: true,
              value: _status,
              decoration: const InputDecoration(labelText: '状态'),
              items: [
                for (final option in galgameStatusOptions)
                  DropdownMenuItem(
                    value: option.value,
                    child: Text(option.label),
                  ),
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
          _buildDescriptionEditor(),
          const SizedBox(height: 12),
          TextFormField(
            controller: _aliasesController,
            decoration: const InputDecoration(labelText: '别名（每行一个）'),
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

  Widget _buildCompanyEditor() => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      GameCompanyEditor(
        companies: _companies,
        enabled: !_saving,
        onChanged: (companies) => setState(() {
          _companies = companies;
          _companiesDirty = true;
        }),
      ),
      if (widget.editId != null || _savedNewGameId != null)
        OutlinedButton(
          onPressed: _saving || !_companiesDirty ? null : _saveCompanies,
          child: const Text('保存公司关联'),
        ),
    ],
  );

  Widget _buildDescriptionEditor() {
    final locale = _descriptionLocale;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text('多语言简介（支持 Markdown）'),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (final (value, label) in _descriptionLocales)
              ChoiceChip(
                label: Text(label),
                selected: locale == value,
                onSelected: (selected) {
                  if (selected) {
                    setState(() => _descriptionLocale = value);
                  }
                },
              ),
          ],
        ),
        const SizedBox(height: 12),
        TextFormField(
          controller: _descriptionControllers[locale],
          decoration: const InputDecoration(labelText: '简介内容'),
          minLines: 4,
          maxLines: 10,
        ),
        const SizedBox(height: 12),
        DropdownButtonFormField<String>(
          isExpanded: true,
          value: _sourceTypes[locale],
          decoration: const InputDecoration(labelText: '来源类型'),
          items: [
            for (final (value, label, _) in _sourceTypeOptions)
              DropdownMenuItem(value: value, child: Text(label)),
          ],
          onChanged: (value) =>
              _onSourceTypeChanged(locale, value ?? 'unknown'),
        ),
        const SizedBox(height: 12),
        TextFormField(
          controller: _sourceNameControllers[locale],
          decoration: const InputDecoration(labelText: '来源名称'),
        ),
        const SizedBox(height: 12),
        TextFormField(
          controller: _sourceUrlControllers[locale],
          decoration: const InputDecoration(
            labelText: '来源链接',
            hintText: 'https://',
          ),
          keyboardType: TextInputType.url,
        ),
        SwitchListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('官方来源'),
          value: _sourceOfficial[locale] ?? false,
          onChanged: (value) => setState(() => _sourceOfficial[locale] = value),
        ),
        const Text(
          '清空内容会保留记录与来源信息；三种语言的简介互不影响。',
          style: TextStyle(fontSize: 12),
        ),
      ],
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
