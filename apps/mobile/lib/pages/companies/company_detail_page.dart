import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../core/api/api_client.dart';
import '../../core/constants/domain.dart';
import '../../core/utils/format.dart';
import '../../models/company_models.dart';
import '../../models/galgame_models.dart';
import '../../models/pagination.dart';
import '../../providers/app_providers.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';
import '../../widgets/markdown_view.dart';
import '../galgames/widgets.dart';
import 'widgets.dart';

class CompanyDetailPage extends ConsumerStatefulWidget {
  const CompanyDetailPage({super.key, required this.slug});
  final String slug;

  @override
  ConsumerState<CompanyDetailPage> createState() => _CompanyDetailPageState();
}

class _CompanyDetailPageState extends ConsumerState<CompanyDetailPage> {
  Company? _company;
  String? _error;
  bool _loading = true;
  int _requestId = 0;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void didUpdateWidget(covariant CompanyDetailPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.slug != widget.slug) _load();
  }

  Future<void> _load() async {
    final requestId = ++_requestId;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final company = await ref.read(companyServiceProvider).get(widget.slug);
      if (!mounted || requestId != _requestId) return;
      setState(() {
        _company = company;
        _loading = false;
      });
    } catch (error) {
      if (!mounted || requestId != _requestId) return;
      setState(() {
        _error = apiErrorMessage(error);
        _loading = false;
      });
    }
  }

  Future<void> _website(String url) async {
    final uri = Uri.tryParse(url);
    if (uri == null ||
        !['https', 'http'].contains(uri.scheme) ||
        uri.host.isEmpty) {
      showAppSnackBar(context, '无效的网站链接', error: true);
      return;
    }
    try {
      if (!await launchUrl(uri, mode: LaunchMode.externalApplication) &&
          mounted) {
        showAppSnackBar(context, '无法打开链接', error: true);
      }
    } catch (_) {
      if (mounted) showAppSnackBar(context, '无法打开链接', error: true);
    }
  }

  Widget _link(CompanySummary company, String label) => ListTile(
    contentPadding: EdgeInsets.zero,
    leading: AppImage(
      url: company.logoUrl,
      width: 36,
      height: 36,
      fit: BoxFit.contain,
    ),
    title: Text(company.name),
    subtitle: Text(label),
    trailing: const Icon(Icons.chevron_right),
    onTap: () =>
        context.push('/companies/${Uri.encodeComponent(company.slug)}'),
  );

  @override
  Widget build(BuildContext context) {
    final company = _company;
    final authenticated = ref.watch(authControllerProvider).isAuthenticated;
    final canManage =
        authenticated &&
        (ref.watch(mePermissionsProvider).asData?.value.has('company:manage') ??
            false);
    return Scaffold(
      appBar: AppBar(
        title: Text(company?.name ?? '会社'),
        actions: [
          if (canManage && company != null)
            IconButton(
              tooltip: '编辑会社',
              icon: const Icon(Icons.edit_outlined),
              onPressed: () async {
                final result = await context.push<Company>(
                  '/admin/companies/${company.id}/edit',
                );
                if (!context.mounted) return;
                if (result != null && result.slug != widget.slug) {
                  context.pushReplacement(
                    '/companies/${Uri.encodeComponent(result.slug)}',
                  );
                } else {
                  _load();
                }
              },
            ),
        ],
      ),
      body: _loading
          ? const LoadingView()
          : _error != null
          ? ErrorView(message: _error!, onRetry: _load)
          : company == null
          ? const EmptyView()
          : RefreshIndicator(
              onRefresh: _load,
              child: ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                children: [
                  if (company.bannerUrl?.isNotEmpty == true)
                    AppImage(url: company.bannerUrl, height: 180),
                  Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            AppImage(
                              url: company.logoUrl,
                              width: 72,
                              height: 72,
                              fit: BoxFit.contain,
                            ),
                            const SizedBox(width: 12),
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    company.name,
                                    style: Theme.of(context)
                                        .textTheme
                                        .titleLarge,
                                  ),
                                  Text(
                                    companyStatuses[company.status] ??
                                        company.status,
                                  ),
                                  if (company.country?.isNotEmpty == true)
                                    Text(company.country!),
                                ],
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),
                        for (final entry in {
                          '日文名': company.nameJa,
                          '中文名': company.nameZh,
                          '英文名': company.nameEn,
                          '成立日期': company.foundedDate,
                          '关闭日期': company.closedDate,
                        }.entries)
                          if (entry.value?.isNotEmpty == true)
                            Text('${entry.key}：${entry.value}'),
                        Text('Slug：${company.slug}'),
                        if (company.aliases.isNotEmpty)
                          Text('别名：${company.aliases.join(' / ')}'),
                        Text(
                          '作品 ${company.gameCount} · 开发 ${company.developerGameCount} · 发行 ${company.publisherGameCount}',
                        ),
                        if (company.createdAt != null)
                          Text('创建：${formatDate(company.createdAt)}'),
                        if (company.updatedAt != null)
                          Text('更新：${formatDate(company.updatedAt)}'),
                        if (company.websiteUrl?.isNotEmpty == true)
                          TextButton.icon(
                            onPressed: () => _website(company.websiteUrl!),
                            icon: const Icon(Icons.open_in_new),
                            label: const Text('官方网站'),
                          ),
                        if (company.parentCompany != null)
                          _link(company.parentCompany!, '母公司'),
                        for (final child in company.children)
                          _link(child, '子公司 / 品牌'),
                        const Divider(),
                        MarkdownView(
                          data: company.description?.isNotEmpty == true
                              ? company.description!
                              : '暂无简介',
                        ),
                        const SizedBox(height: 16),
                        Row(
                          children: [
                            const Expanded(
                              child: Text(
                                '相关作品',
                                style: TextStyle(fontWeight: FontWeight.w600),
                              ),
                            ),
                            TextButton(
                              onPressed: () => context.push(
                                Uri(
                                  path: '/galgames',
                                  queryParameters: {
                                    'company_id': '${company.id}',
                                    'company_name': company.name,
                                  },
                                ).toString(),
                              ),
                              child: const Text('在游戏列表筛选'),
                            ),
                          ],
                        ),
                        _CompanyGames(
                          key: ValueKey(company.slug),
                          slug: company.slug,
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
    );
  }
}

class _CompanyGames extends ConsumerStatefulWidget {
  const _CompanyGames({super.key, required this.slug});
  final String slug;

  @override
  ConsumerState<_CompanyGames> createState() => _CompanyGamesState();
}

class _CompanyGamesState extends ConsumerState<_CompanyGames> {
  final _search = TextEditingController();
  Timer? _debounce;
  Paginated<GalgameListItem>? _result;
  String? _role, _error;
  String _sort = galgameSortOptions.first.slug!;
  int _page = 1, _requestId = 0;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load(1);
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _search.dispose();
    super.dispose();
  }

  Future<void> _load(int page) async {
    final requestId = ++_requestId;
    setState(() {
      _loading = true;
      _error = null;
      _page = page;
    });
    try {
      final result = await ref
          .read(companyServiceProvider)
          .games(
            widget.slug,
            page: page,
            role: _role,
            keyword: _search.text.trim(),
            sort: _sort,
          );
      if (!mounted || requestId != _requestId) return;
      setState(() {
        _result = result;
        _loading = false;
      });
    } catch (error) {
      if (!mounted || requestId != _requestId) return;
      setState(() {
        _error = apiErrorMessage(error);
        _loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) => Column(
    children: [
      TextField(
        controller: _search,
        decoration: const InputDecoration(
          labelText: '搜索作品',
          prefixIcon: Icon(Icons.search),
        ),
        onChanged: (_) {
          _debounce?.cancel();
          _debounce = Timer(const Duration(milliseconds: 350), () => _load(1));
        },
      ),
      const SizedBox(height: 12),
      Row(
        children: [
          Expanded(
            child: DropdownButtonFormField<String>(
              initialValue: _role ?? '',
              isExpanded: true,
              decoration: const InputDecoration(labelText: '公司角色'),
              items: [
                const DropdownMenuItem(value: '', child: Text('全部')),
                for (final role in companyRoles.entries)
                  DropdownMenuItem(value: role.key, child: Text(role.value)),
              ],
              onChanged: (value) {
                _role = value == '' ? null : value;
                _load(1);
              },
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: DropdownButtonFormField<String>(
              initialValue: _sort,
              isExpanded: true,
              decoration: const InputDecoration(labelText: '排序'),
              items: [
                for (final option in galgameSortOptions)
                  DropdownMenuItem(
                    value: option.slug,
                    child: Text(option.label),
                  ),
              ],
              onChanged: (value) {
                _sort = value ?? galgameSortOptions.first.slug!;
                _load(1);
              },
            ),
          ),
        ],
      ),
      const SizedBox(height: 12),
      if (_loading)
        const Padding(padding: EdgeInsets.all(24), child: LoadingView())
      else if (_error != null)
        ErrorView(message: _error!, onRetry: () => _load(_page))
      else if (_result?.items.isEmpty ?? true)
        const EmptyView(hint: '没有找到相关作品')
      else
        for (final game in _result!.items)
          Padding(
            padding: const EdgeInsets.only(bottom: 10),
            child: GalgameCard(
              galgame: game,
              onTap: game.id == null
                  ? null
                  : () => context.push('/galgames/${game.id}'),
            ),
          ),
      CompanyPagination(
        page: _page,
        total: _result?.total ?? 0,
        hasMore: _result?.hasMore ?? false,
        loading: _loading || _error != null,
        onPage: _load,
      ),
    ],
  );
}
