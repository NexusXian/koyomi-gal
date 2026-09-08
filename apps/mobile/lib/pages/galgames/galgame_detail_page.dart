import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../core/api/api_client.dart';
import '../../core/constants/domain.dart';
import '../../core/utils/format.dart';
import '../../models/galgame_models.dart';
import '../../models/post_models.dart';
import '../../providers/app_providers.dart';
import '../../services/galgame_service.dart';
import '../../widgets/app_image.dart';
import '../../widgets/common_views.dart';
import '../../widgets/markdown_view.dart';
import '../../widgets/user_widgets.dart';

class GalgameDetailPage extends ConsumerStatefulWidget {
  const GalgameDetailPage({super.key, required this.id});

  final int id;

  @override
  ConsumerState<GalgameDetailPage> createState() => _GalgameDetailPageState();
}

class _GalgameDetailPageState extends ConsumerState<GalgameDetailPage> {
  GalgameDetail? _detail;
  GalgameUserRelation? _relation;
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
      final service = ref.read(galgameServiceProvider);
      final auth = ref.read(authControllerProvider);
      // Detail and per-user relation are independent; fetch in parallel.
      final relationFuture = auth.isAuthenticated
          ? service
                .myRelation(widget.id)
                .then<GalgameUserRelation?>((relation) => relation)
                .catchError((_) => null)
          : Future<GalgameUserRelation?>.value();
      final detail = await service.get(widget.id);
      final relation = await relationFuture;
      if (!mounted) {
        return;
      }
      setState(() {
        _detail = detail;
        _relation = relation;
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

  Future<void> _toggleFavorite() async {
    final service = ref.read(galgameServiceProvider);
    final favorited = _relation?.favorite?.favorited ?? false;
    try {
      if (favorited) {
        await service.removeFavorite(widget.id);
      } else {
        await service.addFavorite(widget.id);
      }
      await _reloadRelation();
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    }
  }

  Future<void> _reloadRelation() async {
    try {
      final relation =
          await ref.read(galgameServiceProvider).myRelation(widget.id);
      if (mounted) {
        setState(() => _relation = relation);
      }
    } catch (_) {}
  }

  Future<void> _rate() async {
    final current = _relation?.rating?.score;
    final score = await showModalBottomSheet<int>(
      context: context,
      builder: (context) => _RatingSheet(currentScore: current),
    );
    if (score == null) {
      return;
    }
    try {
      if (score == 0) {
        await ref.read(galgameServiceProvider).deleteRating(widget.id);
      } else {
        await ref.read(galgameServiceProvider).upsertRating(widget.id, score);
      }
      await _load();
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    }
  }

  Future<void> _setState_(int state) async {
    try {
      if (state == 0) {
        await ref.read(galgameServiceProvider).deleteState(widget.id);
      } else {
        await ref.read(galgameServiceProvider).upsertState(widget.id, state);
      }
      await _reloadRelation();
    } catch (error) {
      if (mounted) {
        showAppSnackBar(context, apiErrorMessage(error), error: true);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: _buildBody(),
    );
  }

  Widget _buildBody() {
    if (_loading) {
      return const LoadingView();
    }
    if (_error != null) {
      return Scaffold(
        appBar: AppBar(),
        body: ErrorView(message: _error!, onRetry: _load),
      );
    }
    final detail = _detail;
    if (detail == null) {
      return Scaffold(appBar: AppBar(), body: const EmptyView());
    }

    final relation = _relation;
    return DefaultTabController(
      length: 6,
      child: NestedScrollView(
        headerSliverBuilder: (context, innerBoxIsScrolled) => [
          SliverAppBar(
            pinned: true,
            expandedHeight: 210,
            flexibleSpace: FlexibleSpaceBar(
              background: AppImage(url: detail.bannerUrl ?? detail.coverUrl),
            ),
            actions: [
              IconButton(
                icon: const Icon(Icons.edit_outlined),
                tooltip: '编辑',
                onPressed: () => context.push('/galgames/${widget.id}/edit'),
              ),
            ],
          ),
          SliverToBoxAdapter(child: _buildHeaderCard(detail, relation)),
          SliverToBoxAdapter(child: _buildDescription(detail)),
          SliverToBoxAdapter(
            child: Column(
              children: [
                TabBar(
                  isScrollable: true,
                  tabAlignment: TabAlignment.start,
                  tabs: [
                    const Tab(text: '角色'),
                    const Tab(text: '画廊'),
                    Tab(text: '资源 ${detail.statistics?.resourceCount ?? 0}'),
                    Tab(text: '帖子 ${detail.statistics?.postCount ?? 0}'),
                    const Tab(text: '相关小说'),
                    const Tab(text: '贡献者'),
                  ],
                ),
              ],
            ),
          ),
        ],
        body: TabBarView(
          children: [
            _CharactersTab(galgameId: widget.id),
            _GalleryTab(galgameId: widget.id),
            _ResourcesTab(galgameId: widget.id),
            _RelatedPostsTab(galgameId: widget.id),
            _RelatedNovelsTab(novels: detail.relatedNovels ?? const []),
            _ContributorsTab(galgameId: widget.id),
          ],
        ),
      ),
    );
  }

  Widget _buildHeaderCard(GalgameDetail detail, GalgameUserRelation? relation) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  AppImage(
                    url: detail.coverUrl,
                    sensitive: detail.coverSensitive,
                    width: 84,
                    height: 112,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          detail.title ?? '',
                          style: const TextStyle(
                            fontSize: 17,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                        if (detail.originalTitle?.isNotEmpty == true)
                          Text(
                            detail.originalTitle!,
                            style: TextStyle(
                              fontSize: 12,
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        const SizedBox(height: 6),
                        Text(
                          [
                            detail.developer?.name ?? '未知开发商',
                            formatDate(detail.releaseDate),
                            ageRatingLabel(detail.ageRating),
                          ].join(' · '),
                          style: TextStyle(
                            fontSize: 12,
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                        const SizedBox(height: 6),
                        Row(
                          children: [
                            if (detail.rating?.average != null)
                              Padding(
                                padding: const EdgeInsets.only(right: 12),
                                child: Row(
                                  children: [
                                    Icon(Icons.star,
                                        size: 16,
                                        color: Colors.amber.shade700),
                                    const SizedBox(width: 2),
                                    Text(
                                      '${detail.rating!.average!.toStringAsFixed(1)} (${detail.rating!.count ?? 0})',
                                      style: const TextStyle(
                                          fontSize: 13,
                                          fontWeight: FontWeight.w600),
                                    ),
                                  ],
                                ),
                              ),
                            StatRow(
                              icon: Icons.favorite_outline,
                              label:
                                  formatCount(detail.statistics?.favoriteCount),
                            ),
                          ],
                        ),
                        if (detail.tags.isNotEmpty)
                          Padding(
                            padding: const EdgeInsets.only(top: 8),
                            child: TagChips(
                              names: detail.tags
                                  .map((tag) => tag.name ?? '')
                                  .where((name) => name.isNotEmpty)
                                  .toList(),
                              max: 6,
                            ),
                          ),
                      ],
                    ),
                  ),
                ],
              ),
              if (detail.aliases?.isNotEmpty == true)
                Padding(
                  padding: const EdgeInsets.only(top: 8),
                  child: Text(
                    '别名：${detail.aliases!.join(' / ')}',
                    style: TextStyle(
                      fontSize: 12,
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ),
              const Divider(height: 20),
              _buildActionRow(relation),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildActionRow(GalgameUserRelation? relation) {
    final auth = ref.read(authControllerProvider);
    if (!auth.isAuthenticated) {
      return Row(
        children: [
          TextButton.icon(
            onPressed: () => context.push('/login'),
            icon: const Icon(Icons.login),
            label: const Text('登录后收藏 / 评分'),
          ),
        ],
      );
    }

    final favorited = relation?.favorite?.favorited ?? false;
    final myScore = relation?.rating?.score;
    final myState = relation?.state?.state;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            _ActionChip(
              icon: favorited ? Icons.favorite : Icons.favorite_border,
              label: favorited ? '已收藏' : '收藏',
              active: favorited,
              onTap: _toggleFavorite,
            ),
            const SizedBox(width: 8),
            _ActionChip(
              icon: Icons.star_outline,
              label: myScore != null ? '我的评分 $myScore' : '评分',
              active: myScore != null,
              onTap: _rate,
            ),
          ],
        ),
        const SizedBox(height: 8),
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(
            children: [
              _StateChip(
                label: '清除状态',
                value: 0,
                selectedValue: myState,
                onSelect: _setState_,
              ),
              for (final option in userStateOptions)
                Padding(
                  padding: const EdgeInsets.only(left: 8),
                  child: _StateChip(
                    label: option.label,
                    value: option.value,
                    selectedValue: myState,
                    onSelect: _setState_,
                  ),
                ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildDescription(GalgameDetail detail) {
    final description = detail.description;
    if (description == null || description.isEmpty) {
      return const SizedBox.shrink();
    }
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: MarkdownView(data: description, selectable: false),
        ),
      ),
    );
  }
}

class _ActionChip extends StatelessWidget {
  const _ActionChip({
    required this.icon,
    required this.label,
    required this.onTap,
    this.active = false,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;
  final bool active;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return OutlinedButton.icon(
      onPressed: onTap,
      icon: Icon(
        icon,
        size: 16,
        color: active ? theme.colorScheme.primary : null,
      ),
      label: Text(
        label,
        style: TextStyle(
          fontSize: 13,
          color: active ? theme.colorScheme.primary : null,
        ),
      ),
    );
  }
}

class _StateChip extends StatelessWidget {
  const _StateChip({
    required this.label,
    required this.value,
    required this.selectedValue,
    required this.onSelect,
  });

  final String label;
  final int value;
  final int? selectedValue;
  final void Function(int) onSelect;

  @override
  Widget build(BuildContext context) {
    final selected = selectedValue == value && !(value == 0 && selectedValue == null);
    return ChoiceChip(
      label: Text(label),
      selected: selected,
      onSelected: (_) => onSelect(value),
    );
  }
}

class _RatingSheet extends StatelessWidget {
  const _RatingSheet({this.currentScore});

  final int? currentScore;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 0, 20, 8),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('评分', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (var score = 1; score <= 10; score++)
                ChoiceChip(
                  label: Text('$score'),
                  selected: currentScore == score,
                  onSelected: (_) => Navigator.of(context).pop(score),
                ),
              if (currentScore != null)
                ActionChip(
                  label: const Text('删除评分'),
                  avatar: const Icon(Icons.delete_outline, size: 16),
                  onPressed: () => Navigator.of(context).pop(0),
                ),
            ],
          ),
          const SizedBox(height: 16),
        ],
      ),
    );
  }
}

class _CharactersTab extends ConsumerStatefulWidget {
  const _CharactersTab({required this.galgameId});

  final int galgameId;

  @override
  ConsumerState<_CharactersTab> createState() => _CharactersTabState();
}

class _CharactersTabState extends ConsumerState<_CharactersTab> {
  List<GalgameCharacter>? _characters;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final characters =
          await ref.read(galgameServiceProvider).characters(widget.galgameId);
      if (mounted) {
        setState(() => _characters = characters);
      }
    } catch (error) {
      if (mounted) {
        setState(() => _error = apiErrorMessage(error));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_error != null) {
      return ErrorView(message: _error!, onRetry: _load);
    }
    if (_characters == null) {
      return const LoadingView();
    }
    if (_characters!.isEmpty) {
      return const EmptyView(hint: '暂无角色信息');
    }
    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: _characters!.length,
      separatorBuilder: (_, _) => const SizedBox(height: 10),
      itemBuilder: (context, index) {
        final character = _characters![index];
        return Card(
          child: Padding(
            padding: const EdgeInsets.all(10),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                AppImage(
                  url: character.imageUrl,
                  width: 64,
                  height: 88,
                  borderRadius: BorderRadius.circular(8),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        character.name ?? '',
                        style: const TextStyle(
                            fontSize: 15, fontWeight: FontWeight.w600),
                      ),
                      if (character.originalName?.isNotEmpty == true)
                        Text(
                          character.originalName!,
                          style: const TextStyle(fontSize: 12),
                        ),
                      const SizedBox(height: 4),
                      Text(
                        [
                          if (character.role != null) 'role: ${character.role}',
                          if (character.gender != null) character.gender!,
                          if (character.height != null) '${character.height}cm',
                          if (character.birthday?.isNotEmpty == true)
                            character.birthday!,
                          if (character.bloodType?.isNotEmpty == true)
                            character.bloodType!,
                        ].join(' · '),
                        style: TextStyle(
                          fontSize: 12,
                          color: Theme.of(context).colorScheme.onSurfaceVariant,
                        ),
                      ),
                      if (character.publicDescription?.isNotEmpty == true)
                        Padding(
                          padding: const EdgeInsets.only(top: 6),
                          child: Text(
                            character.publicDescription!,
                            maxLines: 4,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(fontSize: 13),
                          ),
                        ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _GalleryTab extends ConsumerStatefulWidget {
  const _GalleryTab({required this.galgameId});

  final int galgameId;

  @override
  ConsumerState<_GalleryTab> createState() => _GalleryTabState();
}

class _GalleryTabState extends ConsumerState<_GalleryTab> {
  List<GalleryImage>? _images;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final images =
          await ref.read(galgameServiceProvider).gallery(widget.galgameId);
      if (mounted) {
        setState(() => _images = images);
      }
    } catch (error) {
      if (mounted) {
        setState(() => _error = apiErrorMessage(error));
      }
    }
  }

  void _openViewer(int index) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => _GalleryViewer(images: _images!, initialIndex: index),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    if (_error != null) {
      return ErrorView(message: _error!, onRetry: _load);
    }
    if (_images == null) {
      return const LoadingView();
    }
    if (_images!.isEmpty) {
      return const EmptyView(hint: '暂无游戏画面');
    }
    return GridView.builder(
      padding: const EdgeInsets.all(16),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 3,
        mainAxisSpacing: 8,
        crossAxisSpacing: 8,
      ),
      itemCount: _images!.length,
        itemBuilder: (context, index) {
          final image = _images![index];
          return GestureDetector(
            onTap: () => _openViewer(index),
            child: AppImage(
              url: image.url,
              sensitive: image.isSpoiler,
              borderRadius: BorderRadius.circular(8),
              maxCacheWidth: 480,
            ),
          );
        },
    );
  }
}

class _GalleryViewer extends StatefulWidget {
  const _GalleryViewer({required this.images, required this.initialIndex});

  final List<GalleryImage> images;
  final int initialIndex;

  @override
  State<_GalleryViewer> createState() => _GalleryViewerState();
}

class _GalleryViewerState extends State<_GalleryViewer> {
  late final _pageController = PageController(initialPage: widget.initialIndex);

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(backgroundColor: Colors.black),
      body: PageView.builder(
        controller: _pageController,
        itemCount: widget.images.length,
        itemBuilder: (context, index) {
          final image = widget.images[index];
          return InteractiveViewer(
            maxScale: 4,
            child: Center(
              child: AppImage(
                url: image.url,
                sensitive: image.isSpoiler,
                fit: BoxFit.contain,
                maxCacheWidth: 1600,
              ),
            ),
          );
        },
      ),
    );
  }
}

class _ResourcesTab extends ConsumerStatefulWidget {
  const _ResourcesTab({required this.galgameId});

  final int galgameId;

  @override
  ConsumerState<_ResourcesTab> createState() => _ResourcesTabState();
}

class _ResourcesTabState extends ConsumerState<_ResourcesTab> {
  static const _pageSize = 20;
  List<ResourceDataLite> _items = [];
  int _page = 1;
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
    setState(() => _loading = true);
    try {
      final result = await ref.read(resourceServiceProvider).listByGalgame(
            widget.galgameId,
            page: _page,
            limit: _pageSize,
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
    if (_error != null && _items.isEmpty) {
      return ErrorView(message: _error!, onRetry: () => _load(reset: true));
    }
    if (_items.isEmpty && _loading) {
      return const LoadingView();
    }
    if (_items.isEmpty) {
      return const EmptyView(hint: '暂无资源');
    }
    return NotificationListener<ScrollNotification>(
      onNotification: (notification) {
        if (notification is ScrollEndNotification &&
            notification.metrics.pixels >=
                notification.metrics.maxScrollExtent - 100 &&
            _hasMore &&
            !_loading) {
          _load();
        }
        return false;
      },
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        itemCount: _items.length,
        separatorBuilder: (_, _) => const SizedBox(height: 10),
        itemBuilder: (context, index) {
          final resource = _items[index];
          return Card(
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                                    resource.title ?? '',
                    style: const TextStyle(
                        fontSize: 15, fontWeight: FontWeight.w600),
                  ),
                  Text(
                    domainLabel(resourceTypeOptions, resource.type),
                    style: TextStyle(
                      fontSize: 12,
                      color: Theme.of(context).colorScheme.primary,
                    ),
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
                  const SizedBox(height: 8),
                  Wrap(
                    spacing: 8,
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
              ),
            ),
          );
        },
      ),
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
}

class _RelatedPostsTab extends ConsumerStatefulWidget {
  const _RelatedPostsTab({required this.galgameId});

  final int galgameId;

  @override
  ConsumerState<_RelatedPostsTab> createState() => _RelatedPostsTabState();
}

class _RelatedPostsTabState extends ConsumerState<_RelatedPostsTab> {
  static const _pageSize = 20;
  List<PostData> _items = [];
  int _page = 1;
  bool _loading = false;
  bool _hasMore = true;

  @override
  void initState() {
    super.initState();
    _load(reset: true);
  }

  Future<void> _load({bool reset = false}) async {
    if (_loading) {
      return;
    }
    setState(() => _loading = true);
    try {
      final result = await ref.read(postServiceProvider).list(
            galgameId: widget.galgameId,
            page: _page,
            limit: _pageSize,
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
    if (_items.isEmpty && _loading) {
      return const LoadingView();
    }
    if (_items.isEmpty) {
      return EmptyView(
        hint: '暂无讨论帖',
        child: FilledButton.tonal(
          onPressed: () =>
              context.push('/posts/new?galgameId=${widget.galgameId}'),
          child: const Text('发起讨论'),
        ),
      );
    }
    return NotificationListener<ScrollNotification>(
      onNotification: (notification) {
        if (notification is ScrollEndNotification &&
            notification.metrics.pixels >=
                notification.metrics.maxScrollExtent - 100 &&
            _hasMore &&
            !_loading) {
          _load();
        }
        return false;
      },
      child: ListView.separated(
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
                        child: CircularProgressIndicator(strokeWidth: 2),
                      ),
                    ),
                  )
                : const SizedBox.shrink();
          }
          final post = _items[index];
          return Card(
            child: ListTile(
              title: Text(
                post.title ?? '',
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontSize: 14),
              ),
              subtitle: Text(
                '${post.author?.displayName ?? post.authorName ?? ''} · ${formatRelative(post.createdAt)}',
                style: const TextStyle(fontSize: 12),
              ),
              trailing: StatRow(
                icon: Icons.chat_bubble_outline,
                label: formatCount(post.commentCount),
              ),
              onTap: () => context.push('/posts/${post.id}'),
            ),
          );
        },
      ),
    );
  }
}

class _RelatedNovelsTab extends StatelessWidget {
  const _RelatedNovelsTab({required this.novels});

  final List<RelatedNovelData> novels;

  @override
  Widget build(BuildContext context) {
    if (novels.isEmpty) {
      return const EmptyView(hint: '暂无关联小说');
    }
    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: novels.length,
      separatorBuilder: (_, _) => const SizedBox(height: 10),
      itemBuilder: (context, index) {
        final novel = novels[index];
        return Card(
          child: ListTile(
            leading: AppImage(
              url: novel.coverUrl,
              width: 44,
              height: 60,
              borderRadius: BorderRadius.circular(6),
            ),
            title: Text(novel.title ?? '',
                maxLines: 1, overflow: TextOverflow.ellipsis),
            subtitle: novel.relationType != null
                ? Text(
                    domainLabel(novelRelationTypeOptions,
                        domainValueFromSlug(novelRelationTypeOptions, novel.relationType)),
                    style: const TextStyle(fontSize: 12),
                  )
                : null,
            onTap: novel.workId == null
                ? null
                : () => context.push('/novels/${novel.workId}'),
          ),
        );
      },
    );
  }
}

class _ContributorsTab extends ConsumerStatefulWidget {
  const _ContributorsTab({required this.galgameId});

  final int galgameId;

  @override
  ConsumerState<_ContributorsTab> createState() => _ContributorsTabState();
}

class _ContributorsTabState extends ConsumerState<_ContributorsTab> {
  List<ContributorData>? _contributors;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final result = await ref
          .read(galgameServiceProvider)
          .contributors(widget.galgameId, pageSize: 100);
      if (mounted) {
        setState(() => _contributors = result.items);
      }
    } catch (_) {
      if (mounted) {
        setState(() => _contributors = []);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_contributors == null) {
      return const LoadingView();
    }
    if (_contributors!.isEmpty) {
      return const EmptyView(hint: '暂无贡献者');
    }
    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: _contributors!.length,
      separatorBuilder: (_, _) => const Divider(height: 1),
      itemBuilder: (context, index) {
        final contributor = _contributors![index];
        return ListTile(
          contentPadding: EdgeInsets.zero,
          leading: UserAvatar(url: contributor.avatarUrl, size: 36),
          title: Text(contributor.username ?? ''),
          subtitle: Text(
            '贡献 ${contributor.contributionCount ?? 0} 次',
            style: const TextStyle(fontSize: 12),
          ),
          trailing: Text(
            formatRelative(contributor.lastContributedAt),
            style: TextStyle(
              fontSize: 12,
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            ),
          ),
          onTap: contributor.username == null
              ? null
              : () => context.push('/user/${contributor.username}'),
        );
      },
    );
  }
}
