import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/user_models.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';

class SettingsExperiencePage extends ConsumerStatefulWidget {
  const SettingsExperiencePage({super.key});

  @override
  ConsumerState<SettingsExperiencePage> createState() =>
      _SettingsExperiencePageState();
}

class _SettingsExperiencePageState extends ConsumerState<SettingsExperiencePage> {
  UserLevelData? _level;
  String? _error;
  bool _loading = true;

  final _scrollController = ScrollController();
  List<ExperienceLogData> _logs = [];
  int _page = 1;
  bool _loadingLogs = false;
  bool _hasMore = true;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
    _load();
    _loadLogs(reset: true);
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.position.pixels >=
            _scrollController.position.maxScrollExtent - 200 &&
        !_loadingLogs &&
        _hasMore) {
      _loadLogs();
    }
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final level = await ref.read(meServiceProvider).experience();
      if (!mounted) {
        return;
      }
      setState(() {
        _level = level;
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

  Future<void> _loadLogs({bool reset = false}) async {
    if (_loadingLogs) {
      return;
    }
    setState(() => _loadingLogs = true);
    try {
      final result = await ref
          .read(meServiceProvider)
          .experienceLogs(page: _page, limit: 20);
      if (!mounted) {
        return;
      }
      setState(() {
        _logs = reset ? result.items : [..._logs, ...result.items];
        _hasMore = result.hasMore;
        _page = reset ? 2 : _page + 1;
        _loadingLogs = false;
      });
    } catch (_) {
      if (mounted) {
        setState(() => _loadingLogs = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('等级与经验')),
      body: _buildBody(),
    );
  }

  Widget _buildBody() {
    if (_loading) {
      return const LoadingView();
    }
    if (_error != null) {
      return ErrorView(message: _error!, onRetry: _load);
    }
    final level = _level!;
    return ListView(
      controller: _scrollController,
      padding: const EdgeInsets.all(16),
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              children: [
                Text(
                  level.levelName ?? 'Lv.${level.level}',
                  style: const TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  '总经验 ${level.totalExp ?? 0} EXP · 连续签到 ${level.consecutiveDays ?? 0} 天',
                  style: TextStyle(
                    fontSize: 13,
                    color: Theme.of(context).colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 12),
                ClipRRect(
                  borderRadius: BorderRadius.circular(4),
                  child: LinearProgressIndicator(
                    value: level.isMaxLevel
                        ? 1
                        : (level.progress?.clamp(0, 1) ?? 0),
                    minHeight: 10,
                  ),
                ),
                const SizedBox(height: 6),
                Text(
                  level.isMaxLevel == true
                      ? '已达最高等级'
                      : '当前 ${level.currentLevelExp ?? 0} / ${level.nextLevelExp ?? 0} EXP',
                  style: TextStyle(
                    fontSize: 12,
                    color: Theme.of(context).colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
        ),
        const SizedBox(height: 16),
        const Text(
          '经验流水',
          style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 8),
        if (_logs.isEmpty && _loadingLogs)
          const Padding(
            padding: EdgeInsets.all(24),
            child: Center(
              child: SizedBox(
                width: 22,
                height: 22,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            ),
          )
        else if (_logs.isEmpty)
          const EmptyView(hint: '暂无经验记录')
        else
          Card(
            clipBehavior: Clip.antiAlias,
            child: Column(
              children: [
                for (var i = 0; i < _logs.length; i++) ...[
                  if (i > 0) const Divider(height: 1, indent: 16, endIndent: 16),
                  _buildLogTile(_logs[i]),
                ],
              ],
            ),
          ),
      ],
    );
  }

  Widget _buildLogTile(ExperienceLogData log) {
    final positive = (log.expDelta ?? 0) >= 0;
    return ListTile(
      dense: true,
      title: Text(
        log.description ?? log.eventType ?? '',
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: const TextStyle(fontSize: 14),
      ),
      subtitle: Text(
        formatDateTime(log.createdAt),
        style: const TextStyle(fontSize: 12),
      ),
      trailing: Text(
        '${positive ? '+' : ''}${log.expDelta ?? 0}',
        style: TextStyle(
          fontSize: 14,
          fontWeight: FontWeight.w600,
          color: positive
              ? Theme.of(context).colorScheme.tertiary
              : Theme.of(context).colorScheme.error,
        ),
      ),
    );
  }
}
