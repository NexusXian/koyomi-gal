import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/company_models.dart';
import '../../models/pagination.dart';
import '../../providers/app_providers.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';
import 'widgets.dart';

class CompanyListPage extends StatelessWidget {
  const CompanyListPage({
    super.key,
    this.admin = false,
    this.onSelected,
    this.excludeId,
  });

  final bool admin;
  final ValueChanged<CompanySummary>? onSelected;
  final int? excludeId;

  @override
  Widget build(BuildContext context) {
    final child = _CompanyList(
      admin: admin,
      onSelected: onSelected,
      excludeId: excludeId,
    );
    return admin ? CompanyPermissionView(title: '会社管理', child: child) : child;
  }
}

class _CompanyList extends ConsumerStatefulWidget {
  const _CompanyList({required this.admin, this.onSelected, this.excludeId});

  final bool admin;
  final ValueChanged<CompanySummary>? onSelected;
  final int? excludeId;

  @override
  ConsumerState<_CompanyList> createState() => _CompanyListState();
}

class _CompanyListState extends ConsumerState<_CompanyList> {
  final _search = TextEditingController();
  final _country = TextEditingController();
  Timer? _debounce;
  Paginated<Company>? _result;
  String? _status, _error;
  String _sort = 'created_at';
  int _page = 1, _requestId = 0;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    if (widget.onSelected != null) _sort = 'name';
    _load(1);
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _search.dispose();
    _country.dispose();
    super.dispose();
  }

  void _searchChanged(String _) {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 350), () => _load(1));
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
          .list(
            admin: widget.admin,
            keyword: _search.text.trim(),
            country: _country.text.trim(),
            status: _status,
            sort: _sort,
            page: page,
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

  Future<void> _edit([int? id]) async {
    await context.push(
      id == null ? '/admin/companies/new' : '/admin/companies/$id/edit',
    );
    if (mounted) await _load(_page);
  }

  Future<void> _delete(Company company) async {
    final confirmed = await showConfirmDialog(
      context,
      title: '删除公司',
      content: '确定删除 ${company.name}？',
      danger: true,
    );
    if (!confirmed || !mounted) return;
    setState(() => _loading = true);
    try {
      await ref.read(companyServiceProvider).delete(company.id);
      if (!mounted) return;
      showAppSnackBar(context, '已删除');
      await _load(_result?.items.length == 1 && _page > 1 ? _page - 1 : _page);
    } catch (error) {
      if (mounted) {
        setState(() => _loading = false);
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      title: Text(
        widget.onSelected != null
            ? '选择会社'
            : widget.admin
            ? '会社管理'
            : '会社',
      ),
    ),
    floatingActionButton: !widget.admin
        ? null
        : FloatingActionButton(
            onPressed: _loading ? null : () => _edit(),
            child: const Icon(Icons.add),
          ),
    body: Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            children: [
              TextField(
                controller: _search,
                onChanged: _searchChanged,
                onSubmitted: (_) => _load(1),
                decoration: InputDecoration(
                  labelText: '搜索公司名称或别名',
                  prefixIcon: const Icon(Icons.search),
                  suffixIcon: IconButton(
                    tooltip: '清空搜索',
                    icon: const Icon(Icons.close),
                    onPressed: () {
                      _debounce?.cancel();
                      _search.clear();
                      _load(1);
                    },
                  ),
                ),
              ),
              if (widget.onSelected == null) ...[
                const SizedBox(height: 12),
                Row(
                  children: [
                    Expanded(
                      child: DropdownButtonFormField<String>(
                        initialValue: _status ?? '',
                        isExpanded: true,
                        decoration: const InputDecoration(labelText: '状态'),
                        items: [
                          const DropdownMenuItem(value: '', child: Text('全部')),
                          for (final item in companyStatuses.entries)
                            DropdownMenuItem(
                              value: item.key,
                              child: Text(item.value),
                            ),
                        ],
                        onChanged: (value) {
                          _status = value == '' ? null : value;
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
                        items: const [
                          DropdownMenuItem(
                            value: 'created_at',
                            child: Text('创建时间'),
                          ),
                          DropdownMenuItem(value: 'name', child: Text('名称')),
                          DropdownMenuItem(
                            value: 'game_count',
                            child: Text('作品数'),
                          ),
                        ],
                        onChanged: (value) {
                          _sort = value ?? 'created_at';
                          _load(1);
                        },
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _country,
                  onChanged: _searchChanged,
                  decoration: const InputDecoration(
                    labelText: '国家 / 地区',
                    isDense: true,
                  ),
                ),
              ],
            ],
          ),
        ),
        Expanded(child: _body()),
        CompanyPagination(
          page: _page,
          total: _result?.total ?? 0,
          hasMore: _result?.hasMore ?? false,
          loading: _loading || _error != null,
          onPage: _load,
        ),
      ],
    ),
  );

  Widget _body() {
    if (_loading) return const LoadingView();
    if (_error != null) {
      return ErrorView(message: _error!, onRetry: () => _load(_page));
    }
    final items = _result?.items ?? [];
    return RefreshIndicator(
      onRefresh: () => _load(_page),
      child: ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(16, 0, 16, 80),
        children: [
          if (items.isEmpty)
            const SizedBox(height: 220, child: EmptyView(hint: '没有找到公司')),
          for (final company in items)
            Card(
              child: ListTile(
                enabled: company.id != widget.excludeId,
                leading: AppImage(
                  url: company.logoUrl,
                  width: 48,
                  height: 48,
                  fit: BoxFit.contain,
                ),
                title: Text(company.name),
                subtitle: Text(
                  [
                    if (widget.admin) 'ID：${company.id} · Slug：${company.slug}',
                    '${company.country ?? ''} · ${companyStatuses[company.status] ?? company.status} · ${company.gameCount} 部作品',
                    if (widget.admin) '更新：${formatDateTime(company.updatedAt)}',
                  ].join('\n'),
                ),
                isThreeLine: widget.admin,
                onTap: () => widget.onSelected != null
                    ? widget.onSelected!(company)
                    : context.push(
                        '/companies/${Uri.encodeComponent(company.slug)}',
                      ),
                trailing: !widget.admin
                    ? const Icon(Icons.chevron_right)
                    : PopupMenuButton<String>(
                        enabled: !_loading,
                        onSelected: (value) => value == 'edit'
                            ? _edit(company.id)
                            : _delete(company),
                        itemBuilder: (_) => const [
                          PopupMenuItem(value: 'edit', child: Text('编辑')),
                          PopupMenuItem(value: 'delete', child: Text('删除')),
                        ],
                      ),
              ),
            ),
        ],
      ),
    );
  }
}
