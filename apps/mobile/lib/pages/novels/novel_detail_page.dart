import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../core/api/api_client.dart';
import '../../core/api/api_exception.dart';
import '../../core/constants/domain.dart';
import '../../core/utils/format.dart';
import '../../models/novel_models.dart';
import '../../providers/app_providers.dart';
import '../../services/galgame_service.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';
import '../../widgets/markdown_view.dart';
import '../../widgets/user_widgets.dart';

class NovelDetailPage extends ConsumerStatefulWidget {
  const NovelDetailPage({super.key, required this.id});

  final int id;

  @override
  ConsumerState<NovelDetailPage> createState() => _NovelDetailPageState();
}

class _NovelDetailPageState extends ConsumerState<NovelDetailPage> {
  NovelDetail? _novel;
  String? _error;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final novel = await ref.read(novelServiceProvider).get(widget.id);
      if (!mounted) {
        return;
      }
      setState(() {
        _novel = novel;
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

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return Scaffold(appBar: AppBar(), body: const LoadingView());
    }
    if (_error != null) {
      return Scaffold(
        appBar: AppBar(),
        body: ErrorView(message: _error!, onRetry: _load),
      );
    }
    final novel = _novel!;
    final canUpdate = ref
        .watch(mePermissionsProvider)
        .maybeWhen(
          data: (permissions) => permissions.has('novel:update'),
          orElse: () => false,
        );
    final canDelete = ref
        .watch(mePermissionsProvider)
        .maybeWhen(
          data: (permissions) => permissions.has('novel:delete'),
          orElse: () => false,
        );
    return Scaffold(
      appBar: AppBar(
        actions: [
          if (canUpdate)
          IconButton(
            icon: const Icon(Icons.edit_outlined),
            tooltip: '编辑',
              onPressed: () =>
                  context.push('/novels/${widget.id}/edit').then((result) {
                    if (result == true) {
                      _load();
                    }
                  }),
            ),
          if (canDelete)
            PopupMenuButton<String>(
              onSelected: (value) {
                if (value == 'delete') {
                  _delete();
                }
              },
              itemBuilder: (context) => [
                const PopupMenuItem(
                  value: 'delete',
                  child: ListTile(
                    leading: Icon(Icons.delete_outline),
                    title: Text('删除小说'),
                    contentPadding: EdgeInsets.zero,
                    dense: true,
                  ),
                ),
              ],
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
        children: [
          _buildHeader(novel),
          if (novel.summary?.isNotEmpty == true) ...[
            const SizedBox(height: 12),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: MarkdownView(data: novel.summary!, selectable: false),
              ),
            ),
          ],
          const SizedBox(height: 16),
          _SectionTitle(
            title:
                '卷册 (${novel.statistics?.volumeCount ?? novel.volumes?.length ?? 0})',
            action: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextButton(
                  onPressed: () => context.push('/novels/${widget.id}/volumes'),
                  child: const Text('全部', style: TextStyle(fontSize: 13)),
                ),
                if (canUpdate)
                  TextButton.icon(
              onPressed: () => context
                  .push('/novels/${widget.id}/volumes/new')
                        .then((result) {
                          if (result == true) {
                            _load();
                          }
                        }),
              icon: const Icon(Icons.add, size: 16),
              label: const Text('添加卷', style: TextStyle(fontSize: 13)),
            ),
              ],
            ),
          ),
          if (novel.volumes?.isEmpty ?? true)
            const Padding(
              padding: EdgeInsets.all(16),
              child: Center(child: Text('暂无卷册')),
            )
          else
            ...novel.volumes!.map(
              (volume) => _buildVolumeTile(volume, canUpdate),
            ),
          if (novel.relatedGalgames?.isNotEmpty == true) ...[
            const SizedBox(height: 16),
            const _SectionTitle(title: '关联 Galgame'),
            ...novel.relatedGalgames!.map(_buildRelatedGalgame),
          ],
          if (novel.contributors?.isNotEmpty == true ||
              (novel.contributorCount ?? 0) > 0) ...[
            const SizedBox(height: 16),
            _SectionTitle(title: '贡献者 (${novel.contributorCount ?? 0})'),
            _buildContributors(novel),
          ],
          const SizedBox(height: 16),
          _NovelResourcesSection(novelId: widget.id),
        ],
      ),
    );
  }

  Future<void> _delete() async {
    final confirmed = await showConfirmDialog(
      context,
      title: '删除小说',
      content: '删除后无法恢复，确定删除吗？',
      confirmText: '删除',
      danger: true,
    );
    if (!confirmed) {
      return;
    }
    try {
      await ref.read(novelServiceProvider).delete(widget.id);
      if (mounted) {
        showAppSnackBar(context, '已删除');
        context.pop(true);
      }
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    }
  }

  Widget _buildContributors(NovelDetail novel) {
    final contributors = novel.contributors ?? const [];
    if (contributors.isEmpty) {
      return const Padding(
        padding: EdgeInsets.all(16),
        child: Center(child: Text('暂无贡献者')),
      );
    }
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Wrap(
          spacing: 16,
          runSpacing: 12,
          children: [
            for (final contributor in contributors)
              GestureDetector(
                onTap: contributor.username?.isEmpty ?? true
                    ? null
                    : () => context.push(
                        '/user/${Uri.encodeComponent(contributor.username!)}',
                      ),
                child: SizedBox(
                  width: 64,
                  child: Column(
                    children: [
                      UserAvatar(url: contributor.avatarUrl, size: 44),
                      const SizedBox(height: 4),
                      Text(
                        contributor.username ?? '',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(fontSize: 12),
                      ),
                      Text(
                        '${contributor.contributionCount ?? 0} 次贡献',
                        style: TextStyle(
                          fontSize: 11,
                          color: Theme.of(context).colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
        ],
      ),
      ),
    );
  }

  Widget _buildHeader(NovelDetail novel) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            AppImage(
              url: novel.coverUrl,
              sensitive: novel.isCoverSensitive,
              width: 96,
              height: 134,
              borderRadius: BorderRadius.circular(8),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    novel.title ?? '',
                    style: const TextStyle(
                      fontSize: 17,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  if (novel.originalTitle?.isNotEmpty == true)
                    Text(
                      novel.originalTitle!,
                      style: TextStyle(
                        fontSize: 12,
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  const SizedBox(height: 6),
                  Text(
                    [
                      if (novel.author?.isNotEmpty == true)
                        '作者：${novel.author}',
                      if (novel.illustrator?.isNotEmpty == true)
                        '插画：${novel.illustrator}',
                      if (novel.publisher?.isNotEmpty == true)
                        '出版社：${novel.publisher}',
                      if (novel.label?.isNotEmpty == true) '文库：${novel.label}',
                      if (novel.language?.isNotEmpty == true)
                        '语言：${novel.language}',
                      if (novel.region?.isNotEmpty == true)
                        '地区：${novel.region}',
                    ].join('\n'),
                    style: TextStyle(
                      fontSize: 12,
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  const SizedBox(height: 6),
                  Text(
                    [
                      '出版：${formatDate(novel.firstReleaseDate)}',
                      domainLabel(
                        novelReleaseStatusOptions,
                        domainValueFromSlug(
                          novelReleaseStatusOptions,
                          novel.releaseStatus,
                        ),
                      ),
                      ageRatingLabel(novel.ageRating),
                    ].join(' · '),
                    style: TextStyle(
                      fontSize: 12,
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  if (novel.tags.isNotEmpty)
                    Padding(
                      padding: const EdgeInsets.only(top: 8),
                      child: TagChips(
                        names: novel.tags
                            .map((tag) => tag.name ?? '')
                            .where((name) => name.isNotEmpty)
                            .toList(),
                        max: 6,
                      ),
                    ),
                  if (novel.officialWebsite?.isNotEmpty == true)
                    TextButton.icon(
                      onPressed: () => _openExternal(novel.officialWebsite),
                      icon: const Icon(Icons.open_in_new, size: 14),
                      label: const Text('官方网站', style: TextStyle(fontSize: 12)),
                      style: TextButton.styleFrom(
                        padding: EdgeInsets.zero,
                        visualDensity: VisualDensity.compact,
                      ),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildVolumeTile(VolumeSummary volume, bool canUpdate) {
    return Card(
      child: ListTile(
        leading: AppImage(
          url: volume.coverUrl,
          sensitive: _novel?.isCoverSensitive ?? false,
          width: 40,
          height: 56,
          borderRadius: BorderRadius.circular(4),
        ),
        title: Text(
          volume.volumeNumber != null
              ? '第 ${volume.volumeNumber} 卷 · ${volume.title ?? ''}'
              : (volume.title ?? ''),
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(fontSize: 14),
        ),
        subtitle: Text(
          [
            if (volume.originalTitle?.isNotEmpty == true) volume.originalTitle!,
          '出版：${formatDate(volume.releaseDate)}',
            if (volume.isbn?.isNotEmpty == true) 'ISBN ${volume.isbn}',
          ].join('\n'),
          style: const TextStyle(fontSize: 12),
        ),
        trailing: canUpdate && volume.id != null
            ? IconButton(
                icon: const Icon(Icons.edit_outlined, size: 20),
                onPressed: () => context
                    .push('/novels/${widget.id}/volumes/${volume.id}/edit')
                    .then((result) {
                      if (result == true) {
                        _load();
                      }
                    }),
              )
            : const Icon(Icons.chevron_right),
        onTap: volume.id == null
            ? null
            : () => context.push('/novels/${widget.id}/volumes/${volume.id}'),
      ),
    );
  }

  Widget _buildRelatedGalgame(RelatedWorkData work) {
    return Card(
      child: ListTile(
        leading: AppImage(
          url: work.coverUrl,
          sensitive: work.coverSensitive,
          width: 44,
          height: 60,
          borderRadius: BorderRadius.circular(6),
        ),
        title: Text(
          work.title ?? '',
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        subtitle: work.relationType != null
            ? Text(
                domainLabel(
                    novelRelationTypeOptions,
                    domainValueFromSlug(
                    novelRelationTypeOptions,
                    work.relationType,
                  ),
                ),
                style: const TextStyle(fontSize: 12),
              )
            : null,
        onTap: work.workId == null
            ? null
            : () => context.push('/galgames/${work.workId}'),
      ),
    );
  }

  Future<void> _openExternal(String? url) async {
    if (url == null || url.isEmpty) {
      return;
    }
    final uri = Uri.tryParse(url);
    if (uri == null) {
      return;
    }
    final opened = await launchUrl(uri, mode: LaunchMode.externalApplication);
    if (!opened && mounted) {
      showAppSnackBar(context, '无法打开链接', error: true);
    }
  }
}

class _SectionTitle extends StatelessWidget {
  const _SectionTitle({required this.title, this.action});

  final String title;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        children: [
          Expanded(
            child: Text(
              title,
              style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
            ),
          ),
          ?action,
        ],
      ),
    );
  }
}

class _NovelResourcesSection extends ConsumerStatefulWidget {
  const _NovelResourcesSection({required this.novelId});

  final int novelId;

  @override
  ConsumerState<_NovelResourcesSection> createState() =>
      _NovelResourcesSectionState();
}

class _NovelResourcesSectionState
    extends ConsumerState<_NovelResourcesSection> {
  List<ResourceDataLite> _items = [];
  int _page = 1;
  int _total = 0;
  bool _loading = false;
  bool _hasMore = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load(reset: true);
  }

  Future<void> _load({bool reset = false}) async {
    if (_loading) {
      return;
    }
    setState(() {
      _loading = true;
      if (reset) {
        _error = null;
      }
    });
    final requestPage = reset ? 1 : _page;
    try {
      final result = await ref
          .read(resourceServiceProvider)
          .listByNovel(widget.novelId, page: requestPage, limit: 20);
      if (!mounted) {
        return;
      }
      setState(() {
        _items = reset ? result.items : [..._items, ...result.items];
        _total = result.total;
        _hasMore = result.hasMore;
        _page = requestPage + 1;
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

  Future<void> _showResourceForm([ResourceDataLite? resource]) async {
    final changed = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (context) =>
          _ResourceFormSheet(novelId: widget.novelId, resource: resource),
    );
    if (changed == true) {
      _load(reset: true);
    }
  }

  Future<void> _deleteResource(ResourceDataLite resource) async {
    if (resource.id == null) {
      return;
    }
    final confirmed = await showConfirmDialog(
      context,
      title: '删除资源',
      content: '删除后无法恢复，确定删除吗？',
      confirmText: '删除',
      danger: true,
    );
    if (!confirmed) {
      return;
    }
    try {
      await ref.read(resourceServiceProvider).delete(resource.id!);
      if (mounted) {
        showAppSnackBar(context, '已删除');
        _load(reset: true);
      }
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    }
  }

  Future<void> _reportResource(ResourceDataLite resource) async {
    if (resource.id == null) {
      return;
    }
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (context) => _ResourceReportSheet(resourceId: resource.id!),
    );
  }

  Future<void> _openLink(String? url) async {
    if (url == null || url.isEmpty) {
      return;
    }
    final uri = Uri.tryParse(url);
    if (uri == null) {
      return;
    }
    final opened = await launchUrl(uri, mode: LaunchMode.externalApplication);
    if (!opened && mounted) {
      showAppSnackBar(context, '无法打开链接', error: true);
    }
  }

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(authControllerProvider);
    final permissions = ref.watch(mePermissionsProvider);
    final canUpdateAny = permissions.maybeWhen(
      data: (value) => value.has('resource:update'),
      orElse: () => false,
    );
    final canDeleteAny = permissions.maybeWhen(
      data: (value) => value.has('resource:delete'),
      orElse: () => false,
    );
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SectionTitle(
          title: '资源 ($_total)',
          action: auth.isAuthenticated
              ? TextButton.icon(
                  onPressed: _showResourceForm,
                  icon: const Icon(Icons.upload_outlined, size: 16),
                  label: const Text('上传', style: TextStyle(fontSize: 13)),
                )
              : null,
        ),
        if (_error != null && _items.isEmpty)
          ErrorView(message: _error!, onRetry: () => _load(reset: true))
        else if (_items.isEmpty && _loading)
          const LoadingView()
        else if (_items.isEmpty)
          const Padding(
            padding: EdgeInsets.all(16),
            child: Center(child: Text('暂无资源')),
          )
        else ...[
          for (final resource in _items)
            _buildResourceCard(
              resource,
              isAuthenticated: auth.isAuthenticated,
              canUpdate: canUpdateAny || resource.uploaderId == auth.user?.id,
              canDelete: canDeleteAny || resource.uploaderId == auth.user?.id,
            ),
          if (_hasMore)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Center(
                child: _loading
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : TextButton(
                        onPressed: () => _load(),
                        child: const Text('加载更多'),
                      ),
              ),
            ),
        ],
      ],
    );
  }

  Widget _buildResourceCard(
    ResourceDataLite resource, {
    required bool isAuthenticated,
    required bool canUpdate,
    required bool canDelete,
  }) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    resource.title ?? '',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                if (resource.status != null && resource.status != 1)
                  Text(
                    domainLabel(galgameStatusOptions, resource.status),
                    style: TextStyle(
                      fontSize: 11,
                      color: theme.colorScheme.tertiary,
                    ),
                  ),
              ],
            ),
            Text(
              domainLabel(resourceTypeOptions, resource.type),
              style: TextStyle(fontSize: 12, color: theme.colorScheme.primary),
            ),
            if (resource.description?.isNotEmpty == true)
              Padding(
                padding: const EdgeInsets.only(top: 4),
                child: Text(
                  resource.description!,
                  maxLines: 3,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(fontSize: 13),
                ),
              ),
            if (resource.uploader?.username?.isNotEmpty == true) ...[
              const SizedBox(height: 8),
              TappableUser(
                username: resource.uploader?.username,
                child: Row(
                  children: [
                    UserAvatar(url: resource.uploader?.avatarUrl, size: 24),
                    const SizedBox(width: 6),
                    Expanded(
                      child: Text(
                        resource.uploader?.displayName ??
                            resource.uploader?.username ??
                            '',
                        style: const TextStyle(fontSize: 12),
                      ),
                    ),
                  ],
                ),
              ),
            ],
            if (resource.links.isNotEmpty) ...[
              const SizedBox(height: 8),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  for (final link in resource.links)
                    OutlinedButton.icon(
                      onPressed: () => _openLink(link.url),
                      icon: const Icon(Icons.open_in_new, size: 14),
                      label: Text(
                        '链接 ${resource.links.indexOf(link) + 1}',
                        style: const TextStyle(fontSize: 12),
                      ),
                    ),
                ],
              ),
            ],
            if (isAuthenticated) ...[
              const SizedBox(height: 4),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  if (canUpdate)
                    TextButton.icon(
                      onPressed: () => _showResourceForm(resource),
                      icon: const Icon(Icons.edit_outlined, size: 15),
                      label: const Text('编辑'),
                    ),
                  if (canDelete)
                    TextButton.icon(
                      onPressed: () => _deleteResource(resource),
                      icon: const Icon(Icons.delete_outline, size: 15),
                      label: const Text('删除'),
                    ),
                  TextButton.icon(
                    onPressed: () => _reportResource(resource),
                    icon: const Icon(Icons.flag_outlined, size: 15),
                    label: const Text('举报'),
                  ),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _ResourceFormSheet extends ConsumerStatefulWidget {
  const _ResourceFormSheet({required this.novelId, this.resource});

  final int novelId;
  final ResourceDataLite? resource;

  @override
  ConsumerState<_ResourceFormSheet> createState() => _ResourceFormSheetState();
}

class _ResourceFormSheetState extends ConsumerState<_ResourceFormSheet> {
  final _formKey = GlobalKey<FormState>();
  late final _titleController = TextEditingController(
    text: widget.resource?.title ?? '',
  );
  late final _descriptionController = TextEditingController(
    text: widget.resource?.description ?? '',
  );
  late final _linksController = TextEditingController(
    text:
        widget.resource?.links
            .map((link) => link.url)
            .whereType<String>()
            .join('\n') ??
        '',
  );
  late int _type = widget.resource?.type ?? 0;
  bool _saving = false;

  bool get isEditing => widget.resource != null;

  @override
  void dispose() {
    _titleController.dispose();
    _descriptionController.dispose();
    _linksController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    final links = _linksController.text
        .split(RegExp(r'[\n,，]'))
        .map((value) => value.trim())
        .where((value) => value.isNotEmpty)
        .toList();
    if (links.isEmpty) {
      showAppSnackBar(context, '请至少填写一个资源链接', error: true);
      return;
    }
    setState(() => _saving = true);
    try {
      final service = ref.read(resourceServiceProvider);
      final payload = <String, dynamic>{
        'title': _titleController.text.trim(),
        'type': _type,
        'description': _descriptionController.text.trim(),
        'links': links,
      };
      if (isEditing) {
        final resource = widget.resource!;
        if (resource.id == null || resource.status == null) {
          throw ApiException(0, '资源状态缺失，无法更新');
        }
        payload['status'] = resource.status;
        await service.update(resource.id!, payload);
      } else {
        payload['target_type'] = 'novel';
        payload['target_id'] = widget.novelId;
        await service.create(payload);
      }
      if (mounted) {
        showAppSnackBar(context, isEditing ? '资源已更新' : '资源已提交，等待审核');
        Navigator.of(context).pop(true);
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
    return SafeArea(
      child: Padding(
        padding: EdgeInsets.fromLTRB(
          20,
          16,
          20,
          20 + MediaQuery.viewInsetsOf(context).bottom,
        ),
        child: SingleChildScrollView(
          child: Form(
            key: _formKey,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  isEditing ? '编辑资源' : '上传资源',
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _titleController,
                  decoration: const InputDecoration(labelText: '标题 *'),
                  validator: (value) =>
                      value == null || value.trim().isEmpty ? '请输入资源标题' : null,
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<int>(
                  initialValue: _type,
                  isExpanded: true,
                  decoration: const InputDecoration(labelText: '类型'),
                  items: [
                    for (final option in resourceTypeOptions)
                      DropdownMenuItem(
                        value: option.value,
                        child: Text(option.label),
                      ),
                  ],
                  onChanged: (value) => setState(() => _type = value ?? 0),
                ),
                const SizedBox(height: 12),
                TextFormField(
                  controller: _descriptionController,
                  decoration: const InputDecoration(labelText: '说明'),
                  minLines: 2,
                  maxLines: 5,
                ),
                const SizedBox(height: 12),
                TextFormField(
                  controller: _linksController,
                  decoration: const InputDecoration(
                    labelText: '资源链接 *',
                    hintText: '每行填写一个链接',
                  ),
                  keyboardType: TextInputType.url,
                  minLines: 3,
                  maxLines: 8,
                ),
                const SizedBox(height: 20),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed: _saving ? null : _submit,
                    child: _saving
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : Text(isEditing ? '保存' : '提交'),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _ResourceReportSheet extends ConsumerStatefulWidget {
  const _ResourceReportSheet({required this.resourceId});

  final int resourceId;

  @override
  ConsumerState<_ResourceReportSheet> createState() =>
      _ResourceReportSheetState();
}

class _ResourceReportSheetState extends ConsumerState<_ResourceReportSheet> {
  final _descriptionController = TextEditingController();
  int _reason = 0;
  bool _saving = false;

  @override
  void dispose() {
    _descriptionController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() => _saving = true);
    try {
      await ref
          .read(resourceServiceProvider)
          .report(
            widget.resourceId,
            reason: _reason,
            description: _descriptionController.text.trim(),
          );
      if (mounted) {
        showAppSnackBar(context, '举报已提交');
        Navigator.of(context).pop();
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
    return SafeArea(
      child: Padding(
        padding: EdgeInsets.fromLTRB(
          20,
          16,
          20,
          20 + MediaQuery.viewInsetsOf(context).bottom,
        ),
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('举报资源', style: Theme.of(context).textTheme.titleLarge),
              const SizedBox(height: 16),
              DropdownButtonFormField<int>(
                initialValue: _reason,
                isExpanded: true,
                decoration: const InputDecoration(labelText: '原因'),
                items: [
                  for (final option in reportReasonOptions)
                    DropdownMenuItem(
                      value: option.value,
                      child: Text(option.label),
                    ),
                ],
                onChanged: (value) => setState(() => _reason = value ?? 0),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _descriptionController,
                decoration: const InputDecoration(labelText: '补充说明'),
                maxLength: 1000,
                minLines: 3,
                maxLines: 6,
              ),
              const SizedBox(height: 12),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: _saving ? null : _submit,
                  child: _saving
                      ? const SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Text('提交举报'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
