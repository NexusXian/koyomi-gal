import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/company_slug.dart';
import '../../models/company_models.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';
import 'widgets.dart';

class CompanyFormPage extends StatelessWidget {
  const CompanyFormPage({super.key, this.editId});
  final int? editId;

  @override
  Widget build(BuildContext context) => CompanyPermissionView(
    title: editId == null ? '新建会社' : '编辑会社',
    child: _CompanyForm(editId: editId),
  );
}

class _CompanyForm extends ConsumerStatefulWidget {
  const _CompanyForm({this.editId});
  final int? editId;

  @override
  ConsumerState<_CompanyForm> createState() => _CompanyFormState();
}

class _CompanyFormState extends ConsumerState<_CompanyForm> {
  static const _labels = {
    'name': '名称 *',
    'slug': 'Slug *',
    'name_ja': '日文名',
    'name_zh': '中文名',
    'name_en': '英文名',
    'description': '简介（Markdown）',
    'logo_url': 'Logo URL',
    'banner_url': '横幅 URL',
    'website_url': '网站 URL',
    'country': '国家 / 地区',
    'founded_date': '成立日期',
    'closed_date': '关闭日期',
    'aliases': '别名（每行一个）',
  };
  final _form = GlobalKey<FormState>();
  final _fields = {
    for (final key in _labels.keys) key: TextEditingController(),
  };
  String _status = 'active';
  CompanySummary? _parent;
  int? _parentId;
  bool _loading = false, _saving = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    if (widget.editId != null) _load();
  }

  @override
  void dispose() {
    for (final controller in _fields.values) {
      controller.dispose();
    }
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final company = await ref
          .read(companyServiceProvider)
          .adminGet(widget.editId!);
      if (!mounted) return;
      final values = CompanyRequest(
        name: company.name,
        slug: company.slug,
        nameJa: company.nameJa ?? '',
        nameZh: company.nameZh ?? '',
        nameEn: company.nameEn ?? '',
        description: company.description ?? '',
        logoUrl: company.logoUrl ?? '',
        bannerUrl: company.bannerUrl ?? '',
        websiteUrl: company.websiteUrl ?? '',
        country: company.country ?? '',
        foundedDate: company.foundedDate,
        closedDate: company.closedDate,
      ).toMap();
      for (final key in _fields.keys) {
        _fields[key]!.text = key == 'aliases'
            ? company.aliases.join('\n')
            : values[key] as String? ?? '';
      }
      setState(() {
        _status = company.status;
        _parent = company.parentCompany;
        _parentId = company.parentCompanyId;
        _loading = false;
      });
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = apiErrorMessage(error);
          _loading = false;
        });
      }
    }
  }

  String _text(String key) => _fields[key]!.text.trim();
  String? _date(String key) => _text(key).isEmpty ? null : _text(key);

  String? _validate(String key, String? value) {
    final text = key == 'slug'
        ? normalizeCompanySlug(value ?? '')
        : value?.trim() ?? '';
    if ((key == 'name' || key == 'slug') && text.isEmpty) {
      return '请填写${_labels[key]}';
    }
    if (key == 'slug' && !isValidCompanySlug(text)) {
      return '不能包含 /、\\ 或控制字符，且不能为 . 或 ..';
    }
    if (key.endsWith('_date') && text.isNotEmpty) {
      final parsed = DateTime.tryParse(text);
      if (!RegExp(r'^\d{4}-\d{2}-\d{2}$').hasMatch(text) ||
          parsed == null ||
          parsed.toIso8601String().substring(0, 10) != text) {
        return '请输入有效日期 YYYY-MM-DD';
      }
    }
    if (key.endsWith('_url') && text.isNotEmpty) {
      final uri = Uri.tryParse(text);
      if (uri == null ||
          !['https', 'http'].contains(uri.scheme) ||
          uri.host.isEmpty) {
        return '请输入有效的 HTTP / HTTPS URL';
      }
    }
    return null;
  }

  Future<void> _save() async {
    if (_saving || !_form.currentState!.validate()) return;
    final permissions = ref.read(mePermissionsProvider).asData?.value;
    if (!(permissions?.has('company:manage') ?? false)) return;
    setState(() => _saving = true);
    try {
      final request = CompanyRequest(
        name: _text('name'),
        slug: _fields['slug']!.text,
        nameJa: _text('name_ja'),
        nameZh: _text('name_zh'),
        nameEn: _text('name_en'),
        description: _text('description'),
        logoUrl: _text('logo_url'),
        bannerUrl: _text('banner_url'),
        websiteUrl: _text('website_url'),
        country: _text('country'),
        foundedDate: _date('founded_date'),
        closedDate: _date('closed_date'),
        status: _status,
        parentCompanyId: _parentId,
        aliases: _text('aliases')
            .split('\n')
            .map((line) => line.trim())
            .where((line) => line.isNotEmpty)
            .toSet()
            .toList(),
      );
      final service = ref.read(companyServiceProvider);
      final company = widget.editId == null
          ? await service.create(request)
          : await service.update(widget.editId!, request);
      if (!mounted) return;
      showAppSnackBar(context, '已保存');
      context.pop(company);
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: Text(widget.editId == null ? '新建会社' : '编辑会社')),
    body: _loading
        ? const LoadingView()
        : _error != null
        ? ErrorView(message: _error!, onRetry: _load)
        : AbsorbPointer(
            absorbing: _saving,
            child: Form(
              key: _form,
              child: ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  for (final field in _labels.entries)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 12),
                      child: TextFormField(
                        controller: _fields[field.key],
                        validator: (value) => _validate(field.key, value),
                        decoration: InputDecoration(
                          labelText: field.value,
                          hintText: field.key.endsWith('_date')
                              ? 'YYYY-MM-DD（可留空）'
                              : null,
                        ),
                        minLines: field.key == 'description'
                            ? 5
                            : field.key == 'aliases'
                            ? 2
                            : 1,
                        maxLines: field.key == 'description'
                            ? 12
                            : field.key == 'aliases'
                            ? 6
                            : 1,
                        keyboardType: field.key.endsWith('_url')
                            ? TextInputType.url
                            : null,
                      ),
                    ),
                  DropdownButtonFormField<String>(
                    initialValue: companyStatuses.containsKey(_status)
                        ? _status
                        : null,
                    decoration: const InputDecoration(labelText: '状态'),
                    items: [
                      for (final status in companyStatuses.entries)
                        DropdownMenuItem(
                          value: status.key,
                          child: Text(status.value),
                        ),
                    ],
                    onChanged: (value) =>
                        setState(() => _status = value ?? 'active'),
                  ),
                  const SizedBox(height: 12),
                  ListTile(
                    contentPadding: EdgeInsets.zero,
                    title: const Text('母公司'),
                    subtitle: Text(
                      _parent?.name ?? (_parentId == null ? '未选择' : '已关联母公司'),
                    ),
                    trailing: _parentId == null
                        ? const Icon(Icons.search)
                        : IconButton(
                            tooltip: '清除母公司',
                            icon: const Icon(Icons.close),
                            onPressed: () => setState(() {
                              _parent = null;
                              _parentId = null;
                            }),
                          ),
                    onTap: () async {
                      final parent = await selectCompany(
                        context,
                        excludeId: widget.editId,
                      );
                      if (parent != null && mounted) {
                        setState(() {
                          _parent = parent;
                          _parentId = parent.id;
                        });
                      }
                    },
                  ),
                  const SizedBox(height: 16),
                  FilledButton(
                    onPressed: _saving ? null : _save,
                    child: _saving
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Text('保存'),
                  ),
                ],
              ),
            ),
          ),
  );
}
