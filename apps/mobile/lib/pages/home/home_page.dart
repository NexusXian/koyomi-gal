import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/utils/format.dart';
import '../../models/misc_models.dart';
import '../../models/post_models.dart';
import '../../providers/app_providers.dart';
import '../../providers/message_providers.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';
import '../../widgets/user_widgets.dart';
import '../galgames/widgets.dart';

class HomePage extends ConsumerStatefulWidget {
  const HomePage({super.key});

  @override
  ConsumerState<HomePage> createState() => _HomePageState();
}

class _HomePageState extends ConsumerState<HomePage> {
  HomeData? _data;
  String? _error;
  bool _loading = true;
  Timer? _autoScrollTimer;
  final _bannerController = PageController(viewportFraction: 0.94);
  int _bannerIndex = 0;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _autoScrollTimer?.cancel();
    _bannerController.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final data = await ref.read(homeServiceProvider).getHome();
      if (!mounted) {
        return;
      }
      setState(() {
        _data = data;
        _loading = false;
      });
      _startBannerTimer();
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

  void _startBannerTimer() {
    _autoScrollTimer?.cancel();
    if (_data == null || _data!.banners.length < 2) {
      return;
    }
    _autoScrollTimer = Timer.periodic(const Duration(seconds: 5), (_) {
      if (!_bannerController.hasClients) {
        return;
      }
      final next = (_bannerIndex + 1) % _data!.banners.length;
      _bannerController.animateToPage(
        next,
        duration: const Duration(milliseconds: 350),
        curve: Curves.easeOutCubic,
      );
    });
  }

  void _openBanner(HomeBanner banner) {
    final linkType = banner.linkType ?? 'none';
    final value = banner.linkValue;
    if (value == null || value.isEmpty) {
      return;
    }
    switch (linkType) {
      case 'galgame':
        final id = int.tryParse(value);
        if (id != null) {
          context.push('/galgames/$id');
        }
      case 'post':
        final id = int.tryParse(value);
        if (id != null) {
          context.push('/posts/$id');
        }
      case 'news':
        final id = int.tryParse(value);
        if (id != null) {
          context.push('/articles/$id');
        }
      case 'none':
        break;
      default:
        break;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Koyomi Gal'),
        actions: [
          IconButton(
            onPressed: () => context.push('/messages'),
            icon: Badge(
              isLabelVisible: ref.watch(
                    unreadMessagesProvider.select((value) => value.count),
                  ) >
                  0,
              label: Text(
                '${ref.watch(unreadMessagesProvider.select((value) => value.count))}',
              ),
              child: const Icon(Icons.mail_outlined),
            ),
          ),
          IconButton(
            onPressed: () => context.push('/notifications'),
            icon: Badge(
              isLabelVisible: ref.watch(
                    unreadNotificationsProvider.select((value) => value.count),
                  ) >
                  0,
              child: const Icon(Icons.notifications_outlined),
            ),
          ),
          IconButton(
            onPressed: () {
              final auth = ref.read(authControllerProvider);
              if (auth.isAuthenticated) {
                context.push(
                    '/user/${auth.user?.username ?? ''}');
              } else {
                context.push('/login');
              }
            },
            icon: const Icon(Icons.account_circle_outlined),
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
      return ErrorView(message: _error!, onRetry: _load);
    }
    final data = _data;
    if (data == null) {
      return const EmptyView();
    }

    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.only(bottom: 24),
        children: [
          if (data.banners.isNotEmpty) _buildBanners(data.banners),
          if (data.announcements.isNotEmpty)
            _buildAnnouncements(data.announcements),
          _buildQuickEntries(),
          if (data.latestGalgames.isNotEmpty) ...[
            SectionHeader(
              title: '最新 Galgame',
              onMore: () => context.go('/galgames'),
            ),
            GalgameHorizontalList(galgames: data.latestGalgames),
          ],
          if (data.popularGalgames.isNotEmpty) ...[
            SectionHeader(
              title: '热门 Galgame',
              onMore: () => context.go('/galgames'),
            ),
            GalgameHorizontalList(galgames: data.popularGalgames),
          ],
          if (data.latestPosts.isNotEmpty) ...[
            SectionHeader(
              title: '最新帖子',
              onMore: () => context.go('/posts'),
            ),
            ...data.latestPosts.take(5).map(_buildHomePostTile),
          ],
          SectionHeader(
            title: '站内资讯',
            onMore: () => context.push('/articles'),
          ),
          _buildArticlesEntry(),
        ],
      ),
    );
  }

  Widget _buildBanners(List<HomeBanner> banners) {
    return SizedBox(
      height: 200,
      child: Stack(
        children: [
          PageView.builder(
            controller: _bannerController,
            itemCount: banners.length,
            onPageChanged: (index) => _bannerIndex = index,
            itemBuilder: (context, index) {
              final banner = banners[index];
              return GestureDetector(
                onTap: () => _openBanner(banner),
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 6),
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(14),
                    child: Stack(
                      fit: StackFit.expand,
                      children: [
                        AppImage(url: banner.imageUrl),
                        // Left-to-right + bottom-to-top dark gradient shade
                        Container(
                          decoration: const BoxDecoration(
                            gradient: LinearGradient(
                              begin: Alignment.centerLeft,
                              end: Alignment.centerRight,
                              colors: [
                                Color(0xCC0C0916),
                                Color(0x73110D1D),
                                Colors.transparent,
                              ],
                              stops: [0.0, 0.5, 0.78],
                            ),
                          ),
                        ),
                        Container(
                          decoration: const BoxDecoration(
                            gradient: LinearGradient(
                              begin: Alignment.bottomCenter,
                              end: Alignment.topCenter,
                              colors: [
                                Color(0x94090710),
                                Colors.transparent,
                              ],
                              stops: [0.0, 0.55],
                            ),
                          ),
                        ),
                        // Text copy
                        Positioned(
                          bottom: 28,
                          left: 20,
                          right: 40,
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              ShaderMask(
                                shaderCallback: (bounds) =>
                                    const LinearGradient(
                                      begin: Alignment(-0.9, 0),
                                      end: Alignment(0.8, 0),
                                      colors: [
                                        Colors.white,
                                        Color(0xFFF2DFFF),
                                        Color(0xFFD9ECFF),
                                      ],
                                    ).createShader(bounds),
                                child: Text(
                                  banner.title ?? '',
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                  style: const TextStyle(
                                    fontSize: 26,
                                    fontWeight: FontWeight.w800,
                                    letterSpacing: -0.035,
                                    height: 1.12,
                                    color: Colors.white,
                                  ),
                                ),
                              ),
                              if (banner.subtitle != null &&
                                  banner.subtitle!.isNotEmpty) ...[
                                const SizedBox(height: 7),
                                Text(
                                  banner.subtitle!,
                                  maxLines: 2,
                                  overflow: TextOverflow.ellipsis,
                                  style: TextStyle(
                                    fontSize: 14,
                                    color: Colors.white.withValues(alpha: 0.82),
                                    height: 1.6,
                                  ),
                                ),
                              ],
                            ],
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              );
            },
          ),
          // Dot indicators
          if (banners.length > 1)
            Positioned(
              right: 15,
              bottom: 12,
              child: _buildBannerIndicators(banners.length),
            ),
        ],
      ),
    );
  }

  Widget _buildBannerIndicators(int count) {
    return ListenableBuilder(
      listenable: _bannerController,
      builder: (context, _) {
        final current = _bannerController.positions.isNotEmpty
            ? (_bannerController.page ?? _bannerIndex).round()
            : _bannerIndex;
        return Row(
          mainAxisSize: MainAxisSize.min,
          children: List.generate(count, (i) {
            final isActive = i == current;
            return AnimatedContainer(
              duration: const Duration(milliseconds: 250),
              margin: const EdgeInsets.only(left: 6),
              width: isActive ? 20 : 7,
              height: 7,
              decoration: BoxDecoration(
                color: isActive
                    ? Colors.white
                    : Colors.white.withValues(alpha: 0.48),
                borderRadius: BorderRadius.circular(99),
              ),
            );
          }),
        );
      },
    );
  }

  Widget _buildAnnouncements(List<AnnouncementData> announcements) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
      child: Card(
        child: Column(
          children: announcements.take(3).map((item) {
            return ListTile(
              dense: true,
              visualDensity: VisualDensity.compact,
              leading: Icon(
                item.isPinned
                    ? Icons.push_pin
                    : Icons.campaign_outlined,
                size: 18,
                color: Theme.of(context).colorScheme.primary,
              ),
              title: Text(
                item.title ?? '',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontSize: 14),
              ),
              trailing: Text(
                formatRelative(item.publishedAt),
                style: TextStyle(
                  fontSize: 12,
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
              ),
              onTap: () => context.push('/articles/${item.id}'),
            );
          }).toList(),
        ),
      ),
    );
  }

  Widget _buildQuickEntries() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceAround,
        children: [
          _QuickEntry(
            icon: Icons.gamepad_outlined,
            label: 'Galgame',
            onTap: () => context.go('/galgames'),
          ),
          _QuickEntry(
            icon: Icons.book_outlined,
            label: '小说',
            onTap: () => context.go('/novels'),
          ),
          _QuickEntry(
            icon: Icons.forum_outlined,
            label: '帖子',
            onTap: () => context.go('/posts'),
          ),
          _QuickEntry(
            icon: Icons.newspaper_outlined,
            label: '资讯',
            onTap: () => context.push('/articles'),
          ),
        ],
      ),
    );
  }

  Widget _buildHomePostTile(HomePost post) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: Card(
        child: ListTile(
          title: Text(
            post.title ?? '',
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w500),
          ),
          subtitle: post.galgame?.title != null
              ? Text(
                  '讨论：${post.galgame!.title}',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(fontSize: 12),
                )
              : null,
          trailing: Wrap(
            spacing: 10,
            children: [
              StatRow(
                icon: Icons.favorite_outline,
                label: formatCount(post.likeCount),
              ),
              StatRow(
                icon: Icons.chat_bubble_outline,
                label: formatCount(post.commentCount),
              ),
            ],
          ),
          onTap: () => context.push('/posts/${post.id}'),
        ),
      ),
    );
  }

  Widget _buildArticlesEntry() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Card(
        child: ListTile(
          leading: const Icon(Icons.newspaper),
          title: const Text('公告 / 新闻 / 活动 / 更新'),
          subtitle: const Text('查看全部站内资讯'),
          trailing: const Icon(Icons.chevron_right),
          onTap: () => context.push('/articles'),
        ),
      ),
    );
  }
}

class _QuickEntry extends StatelessWidget {
  const _QuickEntry({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return GestureDetector(
      onTap: onTap,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 52,
            height: 52,
            decoration: BoxDecoration(
              color: theme.colorScheme.primaryContainer.withValues(alpha: 0.4),
              borderRadius: BorderRadius.circular(16),
            ),
            child: Icon(icon, color: theme.colorScheme.primary),
          ),
          const SizedBox(height: 6),
          Text(label, style: const TextStyle(fontSize: 12)),
        ],
      ),
    );
  }
}
