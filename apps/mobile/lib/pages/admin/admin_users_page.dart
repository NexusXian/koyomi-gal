import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/admin_models.dart';
import '../../models/pagination.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';

class AdminUsersPage extends ConsumerWidget {
  const AdminUsersPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return ref
        .watch(mePermissionsProvider)
        .when(
          loading: () => Scaffold(
            appBar: AppBar(title: const Text('用户与 IP 审计')),
            body: const LoadingView(),
          ),
          error: (error, _) => Scaffold(
            appBar: AppBar(title: const Text('用户与 IP 审计')),
            body: ErrorView(
              message: apiErrorMessage(error),
              onRetry: () => ref.invalidate(mePermissionsProvider),
            ),
          ),
          data: (permissions) {
            final canListUsers = permissions.has('user:list');
            final canViewIPAudit = permissions.has('ip_audit:read');
            return Scaffold(
              appBar: AppBar(title: const Text('用户与 IP 审计')),
              body: canListUsers
                  ? _AdminUserList(canViewIPAudit: canViewIPAudit)
                  : canViewIPAudit
                  ? const _DirectIPLookup()
                  : const EmptyView(hint: '无权访问用户管理'),
            );
          },
        );
  }
}

class _AdminUserList extends ConsumerStatefulWidget {
  const _AdminUserList({required this.canViewIPAudit});

  final bool canViewIPAudit;

  @override
  ConsumerState<_AdminUserList> createState() => _AdminUserListState();
}

class _AdminUserListState extends ConsumerState<_AdminUserList> {
  final _searchController = TextEditingController();
  Paginated<AdminUserData>? _result;
  int _page = 1;
  bool _loading = true;
  String? _error;
  int _requestId = 0;

  @override
  void initState() {
    super.initState();
    _load(1);
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  Future<void> _load(int page) async {
    final requestId = ++_requestId;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final keyword = _searchController.text.trim();
      final result = await ref
          .read(adminServiceProvider)
          .listUsers(keyword: keyword.isEmpty ? null : keyword, page: page);
      if (!mounted || requestId != _requestId) {
        return;
      }
      setState(() {
        _result = result;
        _page = page;
        _loading = false;
      });
    } catch (error) {
      if (!mounted || requestId != _requestId) {
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
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
          child: TextField(
            controller: _searchController,
            textInputAction: TextInputAction.search,
            decoration: InputDecoration(
              hintText: '搜索用户、邮箱或用户 ID',
              prefixIcon: const Icon(Icons.search),
              suffixIcon: IconButton(
                icon: const Icon(Icons.arrow_forward),
                onPressed: () => _load(1),
              ),
            ),
            onSubmitted: (_) => _load(1),
          ),
        ),
        Expanded(child: _buildBody()),
      ],
    );
  }

  Widget _buildBody() {
    if (_loading && _result == null) {
      return const LoadingView();
    }
    if (_error != null && _result == null) {
      return ErrorView(message: _error!, onRetry: () => _load(_page));
    }
    final result = _result;
    if (result == null || result.items.isEmpty) {
      return const EmptyView(hint: '暂无用户');
    }
    return Column(
      children: [
        if (_error != null)
          MaterialBanner(
            content: Text(_error!),
            actions: [
              TextButton(
                onPressed: () => _load(_page),
                child: const Text('重试'),
              ),
            ],
          ),
        Expanded(
          child: RefreshIndicator(
            onRefresh: () => _load(_page),
            child: ListView.separated(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
              itemCount: result.items.length,
              separatorBuilder: (_, _) => const SizedBox(height: 10),
              itemBuilder: (context, index) =>
                  _buildUserCard(result.items[index]),
            ),
          ),
        ),
        _PaginationBar(
          page: _page,
          total: result.total,
          loading: _loading,
          hasMore: result.hasMore,
          onPage: _load,
        ),
      ],
    );
  }

  Widget _buildUserCard(AdminUserData user) {
    final roleNames = user.roles
        .map((role) => role.name ?? role.code)
        .whereType<String>()
        .where((name) => name.isNotEmpty)
        .join('、');
    return Card(
      child: ListTile(
        leading: CircleAvatar(
          child: Text(
            (user.username?.isNotEmpty == true ? user.username![0] : '?')
                .toUpperCase(),
          ),
        ),
        title: Row(
          children: [
            Flexible(child: Text(user.username ?? '用户 #${user.id}')),
            if (user.isBanned) ...[
              const SizedBox(width: 6),
              Text(
                '已封禁',
                style: TextStyle(
                  fontSize: 11,
                  color: Theme.of(context).colorScheme.error,
                ),
              ),
            ],
          ],
        ),
        subtitle: Text(
          [
            '#${user.id} · ${user.email ?? ''}',
            if (roleNames.isNotEmpty) roleNames,
            '注册于 ${formatDateTime(user.createdAt)}',
          ].join('\n'),
          style: const TextStyle(fontSize: 12),
        ),
        isThreeLine: true,
        trailing: widget.canViewIPAudit && user.id != null
            ? IconButton(
                tooltip: '查看 IP 历史',
                icon: const Icon(Icons.location_searching),
                onPressed: () => _openIPHistory(user),
              )
            : null,
      ),
    );
  }

  void _openIPHistory(AdminUserData user) {
    final uri = Uri(
      path: '/admin/users/${user.id}/ip-logs',
      queryParameters: user.username?.isNotEmpty == true
          ? {'name': user.username!}
          : null,
    );
    context.push(uri.toString());
  }
}

class _DirectIPLookup extends StatefulWidget {
  const _DirectIPLookup();

  @override
  State<_DirectIPLookup> createState() => _DirectIPLookupState();
}

class _DirectIPLookupState extends State<_DirectIPLookup> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _open() {
    final userId = int.tryParse(_controller.text.trim());
    if (userId == null || userId <= 0) {
      showAppSnackBar(context, '请输入有效的用户 ID', error: true);
      return;
    }
    context.push('/admin/users/$userId/ip-logs');
  }

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        const Text('输入用户 ID 查看该用户的 IP 历史记录。'),
        const SizedBox(height: 12),
        TextField(
          controller: _controller,
          keyboardType: TextInputType.number,
          textInputAction: TextInputAction.go,
          decoration: const InputDecoration(
            labelText: '用户 ID',
            prefixIcon: Icon(Icons.person_search_outlined),
          ),
          onSubmitted: (_) => _open(),
        ),
        const SizedBox(height: 12),
        FilledButton.icon(
          onPressed: _open,
          icon: const Icon(Icons.location_searching),
          label: const Text('查看 IP 历史'),
        ),
      ],
    );
  }
}

class _PaginationBar extends StatelessWidget {
  const _PaginationBar({
    required this.page,
    required this.total,
    required this.loading,
    required this.hasMore,
    required this.onPage,
  });

  final int page;
  final int total;
  final bool loading;
  final bool hasMore;
  final ValueChanged<int> onPage;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 6, 16, 10),
        child: Row(
          children: [
            OutlinedButton(
              onPressed: loading || page <= 1 ? null : () => onPage(page - 1),
              child: const Text('上一页'),
            ),
            Expanded(
              child: Text(
                '第 $page 页 · 共 $total 条',
                textAlign: TextAlign.center,
                style: const TextStyle(fontSize: 12),
              ),
            ),
            OutlinedButton(
              onPressed: loading || !hasMore ? null : () => onPage(page + 1),
              child: const Text('下一页'),
            ),
          ],
        ),
      ),
    );
  }
}
