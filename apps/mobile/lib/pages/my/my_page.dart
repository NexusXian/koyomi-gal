import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../models/galgame_models.dart';
import '../../models/user_models.dart';
import '../../pages/galgames/widgets.dart';
import '../../providers/app_providers.dart';
import '../../widgets/common_views.dart';
import '../../widgets/user_widgets.dart';

class MyPage extends ConsumerStatefulWidget {
  const MyPage({super.key});

  @override
  ConsumerState<MyPage> createState() => _MyPageState();
}

class _MyPageState extends ConsumerState<MyPage> {
  PublicUserProfile? _profile;
  UserLevelData? _level;
  CheckinStatusData? _checkinStatus;
  bool _checkingIn = false;
  String? _error;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    ref.listenManual(authControllerProvider, (previous, next) {
      if (next.status != AuthStatus.authenticated) {
        setState(() {
          _profile = null;
          _level = null;
          _checkinStatus = null;
          _loading = false;
        });
      } else {
        _load();
      }
    });
    _load();
  }

  Future<void> _load() async {
    final auth = ref.read(authControllerProvider);
    if (!auth.isAuthenticated) {
      setState(() => _loading = false);
      return;
    }
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final me = ref.read(meServiceProvider);
      final results = await Future.wait([
        me.profile(),
        me.experience(),
        me.checkinStatus(),
      ]);
      if (!mounted) {
        return;
      }
      setState(() {
        _profile = results[0] as PublicUserProfile;
        _level = results[1] as UserLevelData;
        _checkinStatus = results[2] as CheckinStatusData;
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

  Future<void> _doCheckin() async {
    if (_checkingIn || _checkinStatus?.checkedInToday == true) {
      return;
    }
    setState(() => _checkingIn = true);
    try {
      final result = await ref.read(meServiceProvider).checkin();
      if (!mounted) {
        return;
      }
      showAppSnackBar(
        context,
        '签到成功，经验 +${result.expGained ?? 0}',
      );
      _load();
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    } finally {
      if (mounted) {
        setState(() => _checkingIn = false);
      }
    }
  }

  Future<void> _logout() async {
    final confirmed = await showConfirmDialog(
      context,
      title: '退出登录',
      content: '确定要退出当前账号吗？',
      confirmText: '退出',
      danger: true,
    );
    if (!confirmed) {
      return;
    }
    await ref.read(authControllerProvider).logout();
  }

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(authControllerProvider.select(
      (controller) => controller.status,
    ));
    return Scaffold(
      appBar: AppBar(
        title: const Text('我的'),
        actions: [
          IconButton(
            icon: Icon(
              Theme.of(context).brightness == Brightness.dark
                  ? Icons.light_mode_outlined
                  : Icons.dark_mode_outlined,
            ),
            tooltip: '切换主题',
            onPressed: () => ref.read(themeControllerProvider).toggle(),
          ),
        ],
      ),
      body: _buildBody(auth),
    );
  }

  Widget _buildBody(AuthStatus status) {
    if (status == AuthStatus.loading) {
      return const LoadingView();
    }
    if (status != AuthStatus.authenticated) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.lock_outline, size: 44),
            const SizedBox(height: 12),
            const Text('登录后查看个人中心'),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: () => context.push('/login'),
              child: const Text('去登录'),
            ),
          ],
        ),
      );
    }
    if (_loading) {
      return const LoadingView();
    }
    if (_error != null) {
      return ErrorView(message: _error!, onRetry: _load);
    }
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          _buildProfileCard(),
          const SizedBox(height: 12),
          if (_level != null) _buildLevelCard(),
          const SizedBox(height: 12),
          _buildMyGalgames(),
          const SizedBox(height: 12),
          _buildMenuCard(),
        ],
      ),
    );
  }

  Widget _buildProfileCard() {
    final profile = _profile;
    final username = ref.read(authControllerProvider).user?.username ?? '';
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            UserAvatar(url: profile?.avatarUrl, size: 60),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    profile?.displayName ?? username,
                    style: const TextStyle(
                      fontSize: 17,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  Text(
                    '@$username',
                    style: TextStyle(
                      fontSize: 12,
                      color: Theme.of(context).colorScheme.onSurfaceVariant,
                    ),
                  ),
                  const SizedBox(height: 4),
                  LevelBadge(level: profile?.level),
                  if (profile?.bio?.isNotEmpty == true)
                    Padding(
                      padding: const EdgeInsets.only(top: 6),
                      child: Text(
                        profile!.bio!,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(fontSize: 13),
                      ),
                    ),
                ],
              ),
            ),
            TextButton(
              onPressed: () => context.push('/user/$username'),
              child: const Text('主页'),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildLevelCard() {
    final level = _level!;
    final checkedIn = _checkinStatus?.checkedInToday ?? level.checkedInToday;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    '${level.levelName ?? 'Lv.${level.level}'} · ${level.totalExp ?? 0} EXP',
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                FilledButton.tonal(
                  onPressed: checkedIn ? null : _doCheckin,
                  child: _checkingIn
                      ? const SizedBox(
                          width: 16,
                          height: 16,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : Text(
                          checkedIn
                              ? '已签到 ${level.consecutiveDays ?? 0} 天'
                              : '签到',
                          style: const TextStyle(fontSize: 13),
                        ),
                ),
              ],
            ),
            const SizedBox(height: 10),
            ClipRRect(
              borderRadius: BorderRadius.circular(4),
              child: LinearProgressIndicator(
                value: level.isMaxLevel
                    ? 1
                    : (level.progress?.clamp(0, 1) ?? 0),
                minHeight: 8,
              ),
            ),
            const SizedBox(height: 6),
            Text(
              level.isMaxLevel == true
                  ? '已达最高等级'
                  : '距离 ${level.nextLevelName ?? '下一级'} 还需 ${level.remainingExp ?? 0} EXP',
              style: TextStyle(
                fontSize: 12,
                color: Theme.of(context).colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildMyGalgames() {
    return Card(
      clipBehavior: Clip.antiAlias,
      child: Column(
        children: [
          ListTile(
            leading: const Icon(Icons.upload_outlined),
            title: const Text('我上传的 Galgame'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => _openMyGalgames('uploaded'),
          ),
          const Divider(height: 1, indent: 16, endIndent: 16),
          ListTile(
            leading: const Icon(Icons.favorite_outline),
            title: const Text('我收藏的 Galgame'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => _openMyGalgames('favorite'),
          ),
        ],
      ),
    );
  }

  void _openMyGalgames(String type) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => _MyGalgamesPage(type: type),
      ),
    );
  }

  Widget _buildMenuCard() {
    return Card(
      clipBehavior: Clip.antiAlias,
      child: Column(
        children: [
          ListTile(
            leading: const Icon(Icons.notifications_outlined),
            title: const Text('通知'),
            trailing: Badge(
              isLabelVisible: ref.watch(
                    unreadNotificationsProvider.select((value) => value.count),
                  ) >
                  0,
              label: Text('${ref.watch(unreadNotificationsProvider.select((value) => value.count))}'),
              child: const Icon(Icons.chevron_right),
            ),
            onTap: () => context.push('/notifications'),
          ),
          const Divider(height: 1, indent: 16, endIndent: 16),
          ListTile(
            leading: const Icon(Icons.edit_outlined),
            title: const Text('编辑资料'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/settings/profile'),
          ),
          const Divider(height: 1, indent: 16, endIndent: 16),
          ListTile(
            leading: const Icon(Icons.password_outlined),
            title: const Text('账号与安全'),
            subtitle: const Text('修改密码'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/settings/security'),
          ),
          const Divider(height: 1, indent: 16, endIndent: 16),
          ListTile(
            leading: const Icon(Icons.privacy_tip_outlined),
            title: const Text('隐私设置'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/settings/privacy'),
          ),
          const Divider(height: 1, indent: 16, endIndent: 16),
          ListTile(
            leading: const Icon(Icons.military_tech_outlined),
            title: const Text('等级与经验'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/settings/experience'),
          ),
          const Divider(height: 1, indent: 16, endIndent: 16),
          ListTile(
            leading: const Icon(Icons.feedback_outlined),
            title: const Text('意见反馈'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/feedback'),
          ),
          const Divider(height: 1, indent: 16, endIndent: 16),
          ListTile(
            leading: Icon(
              Icons.logout,
              color: Theme.of(context).colorScheme.error,
            ),
            title: Text(
              '退出登录',
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
            onTap: _logout,
          ),
        ],
      ),
    );
  }
}

class _MyGalgamesPage extends ConsumerStatefulWidget {
  const _MyGalgamesPage({required this.type});

  final String type;

  @override
  ConsumerState<_MyGalgamesPage> createState() => _MyGalgamesPageState();
}

class _MyGalgamesPageState extends ConsumerState<_MyGalgamesPage> {
  final _scrollController = ScrollController();
  List<GalgameListItem> _items = [];
  int _page = 1;
  bool _loading = false;
  bool _hasMore = true;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
    _load(reset: true);
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.position.pixels >=
            _scrollController.position.maxScrollExtent - 200 &&
        !_loading &&
        _hasMore) {
      _load();
    }
  }

  Future<void> _load({bool reset = false}) async {
    if (_loading) {
      return;
    }
    setState(() => _loading = true);
    try {
      final result = await ref.read(meServiceProvider).galgames(
            type: widget.type,
            page: _page,
            limit: 20,
          );
      if (!mounted) {
        return;
      }
      setState(() {
        _items = reset ? result.items : [..._items, ...result.items];
        _hasMore = result.hasMore;
        _page = reset ? 2 : _page + 1;
        _loading = false;
      });
    } catch (_) {
      if (mounted) {
        setState(() => _loading = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.type == 'favorite' ? '我收藏的 Galgame' : '我上传的 Galgame'),
      ),
      body: _items.isEmpty && _loading
          ? const LoadingView()
          : _items.isEmpty
              ? const EmptyView(hint: '暂无内容')
              : ListView.separated(
                  controller: _scrollController,
                  padding: const EdgeInsets.all(16),
                  itemCount: _items.length + 1,
                  separatorBuilder: (_, _) => const SizedBox(height: 10),
                  itemBuilder: (context, index) {
                    if (index >= _items.length) {
                      return _loading
                          ? const Padding(
                              padding: EdgeInsets.all(12),
                              child: Center(
                                child: SizedBox(
                                  width: 20,
                                  height: 20,
                                  child:
                                      CircularProgressIndicator(strokeWidth: 2),
                                ),
                              ),
                            )
                          : const SizedBox.shrink();
                    }
                    final galgame = _items[index];
                    return GalgameCard(
                      galgame: galgame,
                      onTap: galgame.id == null
                          ? null
                          : () => context.push('/galgames/${galgame.id}'),
                    );
                  },
                ),
    );
  }
}
