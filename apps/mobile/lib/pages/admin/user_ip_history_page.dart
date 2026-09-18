import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/admin_models.dart';
import '../../models/pagination.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';

class UserIPHistoryPage extends ConsumerWidget {
  const UserIPHistoryPage({super.key, required this.userId, this.username});

  final int userId;
  final String? username;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(
        title: Text(
          username?.isNotEmpty == true
              ? '$username 的 IP 历史'
              : '用户 #$userId IP 历史',
        ),
      ),
      body: ref
          .watch(mePermissionsProvider)
          .when(
            loading: () => const LoadingView(),
            error: (error, _) => ErrorView(
              message: apiErrorMessage(error),
              onRetry: () => ref.invalidate(mePermissionsProvider),
            ),
            data: (permissions) => permissions.has('ip_audit:read')
                ? _IPLogList(userId: userId)
                : const EmptyView(hint: '无权查看 IP 历史'),
          ),
    );
  }
}

class _IPLogList extends ConsumerStatefulWidget {
  const _IPLogList({required this.userId});

  final int userId;

  @override
  ConsumerState<_IPLogList> createState() => _IPLogListState();
}

class _IPLogListState extends ConsumerState<_IPLogList> {
  Paginated<UserIPLog>? _result;
  int _page = 1;
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load(1);
  }

  Future<void> _load(int page) async {
    if (_loading && _result != null) {
      return;
    }
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result = await ref
          .read(adminServiceProvider)
          .listUserIPLogs(widget.userId, page: page);
      if (!mounted) {
        return;
      }
      setState(() {
        _result = result;
        _page = page;
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
    if (_loading && _result == null) {
      return const LoadingView();
    }
    if (_error != null && _result == null) {
      return ErrorView(message: _error!, onRetry: () => _load(_page));
    }
    final result = _result;
    if (result == null || result.items.isEmpty) {
      return const EmptyView(hint: '暂无 IP 历史记录');
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
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
              itemCount: result.items.length,
              separatorBuilder: (_, _) => const SizedBox(height: 10),
              itemBuilder: (context, index) =>
                  _IPLogCard(log: result.items[index]),
            ),
          ),
        ),
        _HistoryPaginationBar(
          page: _page,
          total: result.total,
          loading: _loading,
          hasMore: result.hasMore,
          onPage: _load,
        ),
      ],
    );
  }
}

class _IPLogCard extends StatelessWidget {
  const _IPLogCard({required this.log});

  final UserIPLog log;

  @override
  Widget build(BuildContext context) {
    final location = [
      log.country,
      log.region,
      log.city,
    ].whereType<String>().where((value) => value.isNotEmpty).join(' ');
    final action = switch (log.action) {
      'post' => '发帖',
      'comment' => '评论',
      _ => log.action ?? '未知操作',
    };
    final actionText = log.entityId == null
        ? action
        : '$action #${log.entityId}';
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: SelectableText(
                    log.ip ?? '-',
                    style: const TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                Text(
                  actionText,
                  style: TextStyle(
                    fontSize: 12,
                    color: Theme.of(context).colorScheme.primary,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            if (location.isNotEmpty)
              _DetailRow(icon: Icons.place_outlined, text: location),
            if (log.isp?.isNotEmpty == true)
              _DetailRow(icon: Icons.network_check_outlined, text: log.isp!),
            _DetailRow(
              icon: Icons.schedule_outlined,
              text: formatDateTime(log.createdAt),
            ),
          ],
        ),
      ),
    );
  }
}

class _DetailRow extends StatelessWidget {
  const _DetailRow({required this.icon, required this.text});

  final IconData icon;
  final String text;

  @override
  Widget build(BuildContext context) {
    final color = Theme.of(context).colorScheme.onSurfaceVariant;
    return Padding(
      padding: const EdgeInsets.only(top: 3),
      child: Row(
        children: [
          Icon(icon, size: 14, color: color),
          const SizedBox(width: 5),
          Expanded(
            child: Text(text, style: TextStyle(fontSize: 12, color: color)),
          ),
        ],
      ),
    );
  }
}

class _HistoryPaginationBar extends StatelessWidget {
  const _HistoryPaginationBar({
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
